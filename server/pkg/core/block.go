package core

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"qall-registry-server/pkg/blockstores"
	"qall-registry-server/pkg/datastores"
	"qall-registry-server/pkg/objects"
	"runtime"
	"sync"
	"sync/atomic"

	cid "github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"golang.org/x/sync/singleflight"
)

const maxBlockSize = 1 * 1024 * 1024 // 1 MB

type RegistryCore struct {
	bLock  singleflight.Group
	bStore blockstores.BlockStore
	dStore datastores.DataStore
}

func NewRegistryCore(bs blockstores.BlockStore, ds datastores.DataStore) *RegistryCore {
	return &RegistryCore{bStore: bs, dStore: ds}
}

func (c *RegistryCore) CheckBlocks(ctx context.Context, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	uniqueHashes := make([]string, 0, len(hashes))
	seen := make(map[string]bool)

	for _, h := range hashes {
		if !seen[h] {
			seen[h] = true
			uniqueHashes = append(uniqueHashes, h)
		}
	}

	maxWorkers := runtime.GOMAXPROCS(0) * 2
	semaphore := make(chan struct{}, maxWorkers)

	var wg sync.WaitGroup
	missingChan := make(chan string, len(uniqueHashes))
	errChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	process := func(hash string) {
		defer wg.Done()

		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
		case <-ctx.Done():
			return
		}

		exists, err := c.bStore.Has(ctx, hash)
		if err != nil {
			select {
			case errChan <- err:
				cancel()
			default:
			}
			return
		}

		if !exists {
			missingChan <- hash
		}
	}

	for _, h := range uniqueHashes {
		if _, err := cid.Decode(h); err != nil {
			return nil, fmt.Errorf("invalid CID format: %w", err)
		}

		wg.Add(1)
		go process(h)
	}

	wg.Wait()
	close(missingChan)

	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	var missing_blocks []string
	for h := range missingChan {
		missing_blocks = append(missing_blocks, h)
	}

	return missing_blocks, nil
}

func (c *RegistryCore) PushBlocks(ctx context.Context, blocks []objects.Block) (int32, error) {
	if len(blocks) == 0 {
		return 0, nil
	}

	uniqueBlocks := make([]objects.Block, 0, len(blocks))
	seen := make(map[string]bool)
	for _, b := range blocks {
		if !seen[b.Hash] {
			seen[b.Hash] = true
			uniqueBlocks = append(uniqueBlocks, b)
		}
	}

	maxWorkers := runtime.GOMAXPROCS(0)
	semaphore := make(chan struct{}, maxWorkers)

	var wg sync.WaitGroup
	var writtenCount int32
	var firstError error
	var errOnce sync.Once

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	process := func(block objects.Block) {
		defer wg.Done()

		if len(block.Data) > maxBlockSize {
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("block %s exceeds max size (%d bytes)", block.Hash, len(block.Data)))
			return
		}

		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
		case <-ctx.Done():
			return
		}

		targetCID, err := cid.Decode(block.Hash)
		if err != nil {
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("failed to decode CID %s: %w", block.Hash, err))
			return
		}

		nodeBuilder := basicnode.Prototype.Any.NewBuilder()
		dataReader := bytes.NewReader(block.Data)

		switch targetCID.Prefix().Codec {
		case cid.DagCBOR:
			if err := dagcbor.Decode(nodeBuilder, dataReader); err != nil {
				c.handleError(&firstError, &errOnce, cancel,
					fmt.Errorf("block %s failed DAG-CBOR validation: %w", block.Hash, err))
				return
			}
		case cid.DagJSON:
			if err := dagjson.Decode(nodeBuilder, dataReader); err != nil {
				c.handleError(&firstError, &errOnce, cancel,
					fmt.Errorf("block %s failed DAG-JSON validation: %w", block.Hash, err))
				return
			}
		default:
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("unsupported IPLD codec (%d) for block %s", targetCID.Prefix().Codec, block.Hash))
			return
		}

		computedCID, err := targetCID.Prefix().Sum(block.Data)
		if err != nil {
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("failed to compute checksum for %s: %w", block.Hash, err))
			return
		}

		if !computedCID.Equals(targetCID) {
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("cryptographic integrity violation for CID %s", block.Hash))
			return
		}

		result, err, _ := c.bLock.Do(block.Hash, func() (interface{}, error) {
			exists, err := c.bStore.Has(ctx, block.Hash)
			if err != nil {
				return false, err
			}

			if exists {
				log.Printf("block %s already exists", block.Hash)

				return false, nil
			}

			if err := c.bStore.Put(ctx, block.Hash, block.Data); err != nil {
				return false, err
			}

			log.Printf("new block %s", block.Hash)

			return true, nil
		})

		if err != nil {
			c.handleError(&firstError, &errOnce, cancel, err)
			return
		}

		if result.(bool) {
			atomic.AddInt32(&writtenCount, 1)
		}
	}

	for _, b := range uniqueBlocks {
		wg.Add(1)
		go process(b)
	}

	wg.Wait()

	if firstError != nil {
		return 0, firstError
	}

	return writtenCount, nil
}

func (c *RegistryCore) GetBlocks(ctx context.Context, hashes []string) ([]objects.Block, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	seen := make(map[string]int)
	uniqueHashes := make([]string, 0, len(hashes))
	for _, h := range hashes {
		if _, exists := seen[h]; !exists {
			seen[h] = len(uniqueHashes)
			uniqueHashes = append(uniqueHashes, h)
		}
	}

	targetCIDs := make([]cid.Cid, len(uniqueHashes))
	for i, hash := range uniqueHashes {
		parsed, err := cid.Decode(hash)
		if err != nil {
			return nil, fmt.Errorf("invalid CID format %s: %w", hash, err)
		}
		targetCIDs[i] = parsed
	}

	maxWorkers := runtime.GOMAXPROCS(0) * 2
	semaphore := make(chan struct{}, maxWorkers)

	results := make([]objects.Block, len(uniqueHashes))

	var wg sync.WaitGroup
	var firstError error
	var errOnce sync.Once

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	process := func(idx int, h string, targetCID cid.Cid) {
		defer wg.Done()

		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
		case <-ctx.Done():
			return
		}

		raw, err, _ := c.bLock.Do("get:"+h, func() (interface{}, error) {
			return c.bStore.Get(ctx, h)
		})

		if err != nil {
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("failed to retrieve block %s: %w", h, err))
			return
		}

		data := raw.([]byte)

		computedCID, err := targetCID.Prefix().Sum(data)
		if err != nil {
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("failed to compute checksum for %s: %w", h, err))
			return
		}

		if !computedCID.Equals(targetCID) {
			c.handleError(&firstError, &errOnce, cancel,
				fmt.Errorf("integrity violation for block %s: stored data does not match its CID", h))
			return
		}

		results[idx] = objects.Block{Hash: h, Data: data}
	}

	for i, hash := range uniqueHashes {
		wg.Add(1)
		go process(i, hash, targetCIDs[i])
	}

	wg.Wait()

	if firstError != nil {
		return nil, firstError
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	finalResults := make([]objects.Block, len(hashes))
	for i, h := range hashes {
		finalResults[i] = results[seen[h]]
	}

	return finalResults, nil
}

func (c *RegistryCore) handleError(firstErr *error, once *sync.Once, cancel context.CancelFunc, err error) {
	once.Do(func() {
		*firstErr = err
		cancel()
	})
}
