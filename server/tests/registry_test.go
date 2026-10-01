package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"
	"testing"

	bs "github.com/qall-project/qall-registry/server/pkg/blockstores/local"
	"github.com/qall-project/qall-registry/server/pkg/core"
	ds "github.com/qall-project/qall-registry/server/pkg/datastores/local"
	"github.com/qall-project/qall-registry/server/pkg/objects"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/multiformats/go-multihash"
)

func generateRandomString(t *testing.T) string {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	return hex.EncodeToString(bytes)
}

func createValidIPLDBlock(t *testing.T, blockType string) objects.Block {
	nBig, _ := rand.Int(rand.Reader, big.NewInt(1000000))

	canonicalJSON := fmt.Sprintf(
		`{"code":"def rand_%s(): pass","environment":{"image":"python:alpine","requirements":[]},"nonce":%d,"type":"%s"}`,
		generateRandomString(t), nBig.Int64(), blockType,
	)
	data := []byte(canonicalJSON)

	prefix := cid.Prefix{
		Version:  1,
		Codec:    cid.DagJSON, // 0x0129
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}

	computedCID, err := prefix.Sum(data)
	if err != nil {
		t.Fatalf("Failed to compute IPLD CID: %v", err)
	}

	return objects.Block{
		Hash: computedCID.String(),
		Data: data,
	}
}

func createValidIPLDCBORBlock(t *testing.T, blockType string) objects.Block {
	builder := basicnode.Prototype.Any.NewBuilder()
	ma, err := builder.BeginMap(3)
	if err != nil {
		t.Fatalf("%v", err)
	}

	k1, _ := ma.AssembleEntry("type")
	k1.AssignString(blockType)
	k2, _ := ma.AssembleEntry("code")
	k2.AssignString(fmt.Sprintf("def cbor_%s(): pass", generateRandomString(t)))
	k3, _ := ma.AssembleEntry("nonce")
	nBig, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	k3.AssignInt(nBig.Int64())

	if err := ma.Finish(); err != nil {
		t.Fatalf("%v", err)
	}
	node := builder.Build()

	var buf bytes.Buffer
	if err := dagcbor.Encode(node, &buf); err != nil {
		t.Fatalf("Failed to encode node to DAG-CBOR: %v", err)
	}
	data := buf.Bytes()

	prefix := cid.Prefix{
		Version:  1,
		Codec:    cid.DagCBOR, // Codec 0x71
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}

	computedCID, err := prefix.Sum(data)
	if err != nil {
		t.Fatalf("Failed to compute CBOR CID: %v", err)
	}

	return objects.Block{
		Hash: computedCID.String(),
		Data: data,
	}
}

func setupTestEnvironment(t *testing.T) (*core.RegistryCore, func()) {
	tmpDir, err := os.MkdirTemp("", "qall-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for testing: %v", err)
	}

	bsl, err := bs.NewLocalBlockStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init LocalBlockStore: %v", err)
	}

	dsl, err := ds.NewLocalDataStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init LocalDataStore: %v", err)
	}

	coreEngine := core.NewRegistryCore(bsl, dsl)

	cleanup := func() {
		dsl.Close()
		os.RemoveAll(tmpDir)
	}

	return coreEngine, cleanup
}

func TestPushAndCheckBlocks_Success(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	b1 := createValidIPLDBlock(t, "payload")
	b2 := createValidIPLDBlock(t, "constraints")
	b3 := createValidIPLDBlock(t, "node")
	batch := []objects.Block{b1, b2, b3}
	hashes := []string{b1.Hash, b2.Hash, b3.Hash}

	missing, err := coreEngine.CheckBlocks(ctx, hashes)
	if err != nil {
		t.Errorf("CheckBlocks failed unexpectedly: %v", err)
	}
	if len(missing) != 3 {
		t.Errorf("Expected 3 missing blocks, got %d", len(missing))
	}

	written, err := coreEngine.PushBlocks(ctx, batch)
	if err != nil {
		t.Errorf("PushBlocks failed unexpectedly: %v", err)
	}

	if written != 3 {
		t.Errorf("Expected 3 written blocks, got %d", written)
	}

	missingAfter, err := coreEngine.CheckBlocks(ctx, hashes)
	if err != nil {
		t.Errorf("Second CheckBlocks failed: %v", err)
	}

	if len(missingAfter) != 0 {
		t.Errorf("Expected 0 missing blocks after push, still missing: %v", missingAfter)
	}
}

func TestTagAndResolveWorkflow_Success(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	rootNodeBlock := createValidIPLDBlock(t, "node")
	_, err := coreEngine.PushBlocks(ctx, []objects.Block{rootNodeBlock})
	if err != nil {
		t.Fatalf("Failed to push root node requirement: %v", err)
	}

	wfName := "quantum_vqe_" + generateRandomString(t)
	tag := objects.Tag{
		Name:     wfName,
		Version:  "v1.0.0",
		RootHash: rootNodeBlock.Hash,
	}

	uri, err := coreEngine.Tag(ctx, tag)
	if err != nil {
		t.Errorf("Tag failed unexpectedly: %v", err)
	}
	expectedURI := fmt.Sprintf("%s:v1.0.0", wfName)
	if uri != expectedURI {
		t.Errorf("Expected URI %s, got %s", expectedURI, uri)
	}

	resolvedHash, found, err := coreEngine.Resolve(ctx, wfName, "v1.0.0")
	if err != nil {
		t.Errorf("Resolve failed unexpectedly: %v", err)
	}

	if !found {
		t.Errorf("Workflow tag was not found during resolution")
	}

	if resolvedHash != rootNodeBlock.Hash {
		t.Errorf("Resolved hash mismatch. Expected %s, got %s", rootNodeBlock.Hash, resolvedHash)
	}
}

func TestPushBlock_InvalidIPLDStructure_MustFail(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	validCIDBlock := createValidIPLDBlock(t, "payload")

	corruptedBlock := objects.Block{
		Hash: validCIDBlock.Hash,
		Data: []byte("THIS_IS_NOT_VALID_JSON_STRUCTURALLY_BROKEN_RANDOM_BYTES"),
	}

	_, err := coreEngine.PushBlocks(ctx, []objects.Block{corruptedBlock})
	if err == nil {
		t.Errorf("Expected PushBlocks to FAIL due to corrupted non-JSON structure, but it passed.")
	} else {
		t.Logf("Success: PushBlocks failed as expected with message: %v", err)
	}
}

func TestPushBlock_CryptographicHashMismatch_MustFail(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	validBlock := createValidIPLDBlock(t, "payload")

	modifiedData := []byte(string(validBlock.Data) + " ")

	tamperedBlock := objects.Block{
		Hash: validBlock.Hash,
		Data: modifiedData,
	}

	_, err := coreEngine.PushBlocks(ctx, []objects.Block{tamperedBlock})
	if err == nil {
		t.Errorf("Expected PushBlocks to FAIL cryptographic integrity validation, but it passed.")
	} else {
		t.Logf("Success: PushBlocks caught the hash manipulation as expected: %v", err)
	}
}

func TestTagWorkflow_MissingRootNode_MustFail(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	ghostBlock := createValidIPLDBlock(t, "node")

	tag := objects.Tag{
		Name:     "ghost_workflow",
		Version:  "latest",
		RootHash: ghostBlock.Hash, // Pointe vers un bloc inexistant
	}

	_, err := coreEngine.Tag(ctx, tag)
	if err == nil {
		t.Errorf("Expected Tag to FAIL because root node block does not exist, but it passed.")
	} else {
		t.Logf("Success: Tag correctly rejected the broken reference: %v", err)
	}
}

func TestPushBlocks_DAGCBOR_Success(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	cborBlock := createValidIPLDCBORBlock(t, "payload")

	written, err := coreEngine.PushBlocks(ctx, []objects.Block{cborBlock})

	if err != nil {
		t.Errorf("Server rejected a valid DAG-CBOR block: %v", err)
	}

	if written != 1 {
		t.Errorf("Expected 1 block written, got %d", written)
	}
}

func TestTagWorkflow_AutoCreatesLatest_Success(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	rootBlock := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootBlock})

	wfName := "quantum_simulation_" + generateRandomString(t)

	tag := objects.Tag{
		Name:     wfName,
		Version:  "v1.0.0",
		RootHash: rootBlock.Hash,
	}

	_, err := coreEngine.Tag(ctx, tag)
	if err != nil {
		t.Fatalf("Failed to apply initial tag: %v", err)
	}

	latestHash, found, err := coreEngine.Resolve(ctx, wfName, "latest")
	if err != nil {
		t.Fatalf("Failed to resolve latest: %v", err)
	}
	if !found {
		t.Errorf("The 'latest' tag was not automatically generated by the server")
	}
	if latestHash != rootBlock.Hash {
		t.Errorf("The 'latest' pointer does not point to the expected root node")
	}
}

func TestTagWorkflow_LatestRollover_Success(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	wfName := "vqe_pipeline_" + generateRandomString(t)

	rootV1 := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootV1})
	_, _ = coreEngine.Tag(ctx, objects.Tag{Name: wfName, Version: "v1.0.0", RootHash: rootV1.Hash})

	rootV2 := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootV2})
	_, _ = coreEngine.Tag(ctx, objects.Tag{Name: wfName, Version: "v2.0.0", RootHash: rootV2.Hash})

	currentLatestHash, found, err := coreEngine.Resolve(ctx, wfName, "latest")
	if err != nil {
		t.Fatalf("Failed to resolve latest reference: %v", err)
	}
	if !found {
		t.Fatalf("Latest tag disappeared")
	}

	if currentLatestHash == rootV1.Hash {
		t.Errorf("Stale logic: 'latest' is still stuck on the old v1.0.0 root hash")
	}
	if currentLatestHash != rootV2.Hash {
		t.Errorf("Rollover failed: 'latest' should point to v2.0.0 (%s), got %s", rootV2.Hash, currentLatestHash)
	}
}

func TestTagWorkflow_LatestUniquenessAndList_Success(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	wfName := "uniqueness_pipeline_" + generateRandomString(t)

	rootV1 := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootV1})
	_, _ = coreEngine.Tag(ctx, objects.Tag{Name: wfName, Version: "v1.0.0", RootHash: rootV1.Hash})

	rootV2 := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootV2})
	_, _ = coreEngine.Tag(ctx, objects.Tag{Name: wfName, Version: "v2.0.0", RootHash: rootV2.Hash})

	tags, err := coreEngine.ListTags(ctx, wfName)
	if err != nil {
		t.Fatalf("Failed to list workflow versions from datastore: %v", err)
	}

	expectedTotalTags := 3
	if len(tags) != expectedTotalTags {
		t.Errorf("Database Corruption: expected exactly %d tags indexed for this workflow, found %d: %v",
			expectedTotalTags, len(tags), tags)
	}

	latestOccurrences := 0
	var finalLatestHash string

	for _, tag := range tags {
		if tag.Version == "latest" {
			latestOccurrences++
			finalLatestHash = tag.RootHash
		}
	}

	if latestOccurrences != 1 {
		t.Errorf("Expected exactly 1 'latest' tag for this workflow, found %d",
			latestOccurrences)
	}

	if finalLatestHash != rootV2.Hash {
		t.Errorf("The 'latest' tag is still pointing to the old version (%s) instead of the new version (%s)",
			rootV1.Hash, rootV2.Hash)
	}
}

func TestPushBlocks_ConcurrentIdempotency(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	block := createValidIPLDBlock(t, "payload")

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := coreEngine.PushBlocks(ctx, []objects.Block{block})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("Concurrent push failed: %v", err)
	}

	blocks, err := coreEngine.GetBlocks(ctx, []string{block.Hash})
	if err != nil {
		t.Fatalf("Block not readable after concurrent push: %v", err)
	}
	if !bytes.Equal(blocks[0].Data, block.Data) {
		t.Errorf("Block data corrupted after concurrent push")
	}
}

func TestPushBlock_ExceedsMaxSize_MustFail(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	oversizedData := make([]byte, 2*1024*1024) // 2 MB
	rand.Read(oversizedData)

	block := objects.Block{
		Hash: "bafyreiabc123",
		Data: oversizedData,
	}

	_, err := coreEngine.PushBlocks(ctx, []objects.Block{block})
	if err == nil {
		t.Errorf("Expected rejection of oversized block, but push succeeded")
	}
}

func TestTag_WorkflowNameWithColon_MustFail(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	rootBlock := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootBlock})

	tag := objects.Tag{
		Name:     "namespace:workflow",
		Version:  "v1",
		RootHash: rootBlock.Hash,
	}

	_, err := coreEngine.Tag(ctx, tag)
	if err == nil {
		t.Errorf("Expected Tag to reject workflow name containing ':'")
	}
}

func TestUntag_BlocksRemainAfterTagDeletion(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	rootBlock := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootBlock})

	wfName := "disposable_" + generateRandomString(t)
	_, err := coreEngine.Tag(ctx, objects.Tag{
		Name: wfName, Version: "v1", RootHash: rootBlock.Hash,
	})
	if err != nil {
		t.Fatalf("Setup: Tag failed: %v", err)
	}

	deleted, err := coreEngine.Untag(ctx, wfName, "v1")
	if err != nil || !deleted {
		t.Fatalf("Untag failed: %v", err)
	}

	_, found, _ := coreEngine.Resolve(ctx, wfName, "v1")
	if found {
		t.Errorf("Tag still resolves after Untag")
	}

	blocks, err := coreEngine.GetBlocks(ctx, []string{rootBlock.Hash})
	if err != nil {
		t.Errorf("Block was incorrectly removed after Untag: %v", err)
	}
	if !bytes.Equal(blocks[0].Data, rootBlock.Data) {
		t.Errorf("Block data corrupted after Untag")
	}
}

func TestListTags_UnknownWorkflow_ReturnsEmpty(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tags, err := coreEngine.ListTags(context.Background(), "nonexistent_workflow")
	if err != nil {
		t.Fatalf("ListTags on unknown workflow should not error: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("Expected empty list, got %d tags", len(tags))
	}
}

func TestTag_VersionAndLatestAreAtomic(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	rootV1 := createValidIPLDBlock(t, "node")
	rootV2 := createValidIPLDBlock(t, "node")
	_, _ = coreEngine.PushBlocks(ctx, []objects.Block{rootV1, rootV2})

	wfName := "atomic_test_" + generateRandomString(t)
	_, _ = coreEngine.Tag(ctx, objects.Tag{Name: wfName, Version: "v1", RootHash: rootV1.Hash})
	_, _ = coreEngine.Tag(ctx, objects.Tag{Name: wfName, Version: "v2", RootHash: rootV2.Hash})

	tags, _ := coreEngine.ListTags(ctx, wfName)

	latestCount := 0
	var latestHash string

	for _, tag := range tags {
		if tag.Version == "latest" {
			latestCount++
			latestHash = tag.RootHash
		}
	}

	if latestCount != 1 {
		t.Errorf("Expected exactly 1 'latest' tag, found %d", latestCount)
	}

	if latestHash != rootV2.Hash {
		t.Errorf("latest should point to v2 (%s), points to %s", rootV2.Hash, latestHash)
	}

	if len(tags) != 3 {
		t.Errorf("Expected 3 tags total (v1, v2, latest), got %d", len(tags))
	}
}

func TestGetBlocks_SingleMissingBlock_ReturnsError(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	ghost := createValidIPLDBlock(t, "payload")

	_, err := coreEngine.GetBlocks(ctx, []string{ghost.Hash})
	if err == nil {
		t.Fatalf("Expected error for missing block, got nil")
	}
}

func TestGetBlocks_MixedPresentAndMissing_ReturnsError(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	present := createValidIPLDBlock(t, "payload")
	missing := createValidIPLDBlock(t, "node")

	_, err := coreEngine.PushBlocks(ctx, []objects.Block{present})
	if err != nil {
		t.Fatalf("Setup push failed: %v", err)
	}

	_, err = coreEngine.GetBlocks(ctx, []string{present.Hash, missing.Hash})
	if err == nil {
		t.Fatalf("Expected error when one block is missing, got nil")
	}
}

func TestGetBlocks_AllPresent_ReturnsAll(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	b1 := createValidIPLDBlock(t, "payload")
	b2 := createValidIPLDBlock(t, "node")
	b3 := createValidIPLDBlock(t, "constraints")

	_, err := coreEngine.PushBlocks(ctx, []objects.Block{b1, b2, b3})
	if err != nil {
		t.Fatalf("Setup push failed: %v", err)
	}

	results, err := coreEngine.GetBlocks(ctx, []string{b1.Hash, b2.Hash, b3.Hash})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 blocks, got %d", len(results))
	}

	for i, expected := range []objects.Block{b1, b2, b3} {
		if !bytes.Equal(results[i].Data, expected.Data) {
			t.Errorf("Block %d data mismatch", i)
		}
	}
}

func TestGetBlocks_PreCancelledContext_ReturnsContextError(t *testing.T) {
	coreEngine, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	block := createValidIPLDBlock(t, "payload")
	_, err := coreEngine.PushBlocks(context.Background(), []objects.Block{block})
	if err != nil {
		t.Fatalf("Setup push failed: %v", err)
	}

	_, err = coreEngine.GetBlocks(ctx, []string{block.Hash})

	if err == nil {
		t.Errorf("Expected context error for pre-cancelled ctx, got nil — silent failure")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}
