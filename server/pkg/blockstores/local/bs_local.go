package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type LocalBlockStore struct {
	rootDir string
}

func NewLocalBlockStore(root string) (*LocalBlockStore, error) {
	absPath, err := filepath.Abs(root)

	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Join(absPath, "blocks"), 0755); err != nil {
		return nil, err
	}

	return &LocalBlockStore{rootDir: absPath}, nil
}

func (l *LocalBlockStore) getPath(hash string) string {
	return filepath.Join(l.rootDir, "blocks", hash[:2], hash[2:])
}

func (l *LocalBlockStore) Has(ctx context.Context, hash string) (bool, error) {
	_, err := os.Stat(l.getPath(hash))

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func (l *LocalBlockStore) Put(ctx context.Context, hash string, data []byte) error {
	path := l.getPath(hash)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, path)
}

func (l *LocalBlockStore) Get(ctx context.Context, hash string) ([]byte, error) {
	data, err := os.ReadFile(l.getPath(hash))

	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("block not found: %s", hash)
	}

	return data, err
}
