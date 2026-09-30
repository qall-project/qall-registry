package tests

import (
	"bytes"
	"context"
	"os"
	"testing"

	bs3 "github.com/qall-project/qall-registry/pkg/blockstores/object"
	"github.com/qall-project/qall-registry/pkg/core"
	ds "github.com/qall-project/qall-registry/pkg/datastores/local"
	"github.com/qall-project/qall-registry/pkg/objects"
)

func s3ConfigFromEnv(t *testing.T) (bs3.S3Config, bool) {
	t.Helper()

	if os.Getenv("S3_ENDPOINT") == "" {
		return bs3.S3Config{}, false
	}

	return bs3.S3Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		Region:    os.Getenv("S3_REGION"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		Bucket:    os.Getenv("S3_BUCKET"),
	}, true
}

func setupS3TestEnvironment(t *testing.T) (*core.RegistryCore, func()) {
	config, ok := s3ConfigFromEnv(t)
	if !ok {
		t.Skip("S3_ENDPOINT not set, skipping S3 integration test")
	}

	bstore, err := bs3.NewS3BlockStore(config)
	if err != nil {
		t.Fatalf("Failed to init S3BlockStore: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "qall-s3-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for testing: %v", err)
	}

	dsl, err := ds.NewLocalDataStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init LocalDataStore: %v", err)
	}

	coreEngine := core.NewRegistryCore(bstore, dsl)

	cleanup := func() {
		dsl.Close()
		os.RemoveAll(tmpDir)
	}

	return coreEngine, cleanup
}

func TestS3PushAndGetBlocks_Success(t *testing.T) {
	coreEngine, cleanup := setupS3TestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	b1 := createValidIPLDBlock(t, "payload")
	b2 := createValidIPLDBlock(t, "node")

	written, err := coreEngine.PushBlocks(ctx, []objects.Block{b1, b2})
	if err != nil {
		t.Fatalf("PushBlocks failed: %v", err)
	}
	if written != 2 {
		t.Errorf("Expected 2 written blocks, got %d", written)
	}

	missing, err := coreEngine.CheckBlocks(ctx, []string{b1.Hash, b2.Hash})
	if err != nil {
		t.Fatalf("CheckBlocks failed: %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("Expected 0 missing blocks, got %v", missing)
	}

	blocks, err := coreEngine.GetBlocks(ctx, []string{b1.Hash, b2.Hash})
	if err != nil {
		t.Fatalf("GetBlocks failed: %v", err)
	}
	if !bytes.Equal(blocks[0].Data, b1.Data) || !bytes.Equal(blocks[1].Data, b2.Data) {
		t.Errorf("Block data mismatch after S3 roundtrip")
	}
}

func TestS3GetBlocks_MissingBlock_ReturnsError(t *testing.T) {
	coreEngine, cleanup := setupS3TestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	ghost := createValidIPLDBlock(t, "payload")

	_, err := coreEngine.GetBlocks(ctx, []string{ghost.Hash})
	if err == nil {
		t.Fatalf("Expected error for missing block on S3, got nil")
	}
}

func TestS3InvalidCredentials_NotTreatedAsMissing(t *testing.T) {
	config, ok := s3ConfigFromEnv(t)
	if !ok {
		t.Skip("S3_ENDPOINT not set, skipping S3 integration test")
	}

	config.AccessKey = "invalid-access-key"
	config.SecretKey = "invalid-secret-key"

	bstore, err := bs3.NewS3BlockStore(config)
	if err != nil {
		t.Fatalf("Failed to init S3BlockStore with invalid credentials: %v", err)
	}

	ctx := context.Background()
	ghost := createValidIPLDBlock(t, "payload")

	exists, err := bstore.Has(ctx, ghost.Hash)
	if err == nil {
		t.Fatalf("Expected error (AccessDenied) from Has with invalid credentials, got exists=%v", exists)
	}
	if exists {
		t.Errorf("Has returned true on access denied")
	}

	if _, err := bstore.Get(ctx, ghost.Hash); err == nil {
		t.Fatalf("Expected error (AccessDenied) from Get with invalid credentials, got nil")
	}
}

func TestS3CheckBlocks_MissingBlock_ReturnsError(t *testing.T) {
	coreEngine, cleanup := setupS3TestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	ghost := createValidIPLDBlock(t, "payload")

	missing, err := coreEngine.CheckBlocks(ctx, []string{ghost.Hash})
	if err != nil {
		t.Fatalf("Expected no error from CheckBlocks for a missing block, got: %v", err)
	}
	if len(missing) != 1 || missing[0] != ghost.Hash {
		t.Errorf("Expected %q to be reported missing, got %v", ghost.Hash, missing)
	}
}

func TestS3TagWorkflow_ResolvesRootHash(t *testing.T) {
	coreEngine, cleanup := setupS3TestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	rootBlock := createValidIPLDBlock(t, "node")
	_, err := coreEngine.PushBlocks(ctx, []objects.Block{rootBlock})
	if err != nil {
		t.Fatalf("Failed to push root node: %v", err)
	}

	wfName := "s3_workflow_" + os.Getenv("S3_BUCKET")
	_, err = coreEngine.Tag(ctx, objects.Tag{Name: wfName, Version: "v1", RootHash: rootBlock.Hash})
	if err != nil {
		t.Fatalf("Tag failed: %v", err)
	}

	resolved, found, err := coreEngine.Resolve(ctx, wfName, "v1")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if !found {
		t.Errorf("Tag not found during resolve")
	}
	if resolved != rootBlock.Hash {
		t.Errorf("Resolved hash mismatch: expected %s, got %s", rootBlock.Hash, resolved)
	}

	blocks, err := coreEngine.GetBlocks(ctx, []string{resolved})
	if err != nil {
		t.Fatalf("Failed to retrieve blocks through S3: %v", err)
	}
	if !bytes.Equal(blocks[0].Data, rootBlock.Data) {
		t.Errorf("Block data corrupted through S3")
	}
}
