package blockstores

import "context"

type BlockStore interface {
	Has(ctx context.Context, hash string) (bool, error)
	Put(ctx context.Context, hash string, data []byte) error
	Get(ctx context.Context, hash string) ([]byte, error)
}
