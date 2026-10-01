package local

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/qall-project/qall-registry/server/pkg/objects"

	"go.etcd.io/bbolt"
)

type BBoltDataStore struct {
	db         *bbolt.DB
	bucketName []byte
}

func NewLocalDataStore(root string) (*BBoltDataStore, error) {
	dbPath := filepath.Join(root, "datastore.db")

	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})

	if err != nil {
		return nil, err
	}

	bucketName := []byte("datastore")
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return &BBoltDataStore{db: db, bucketName: bucketName}, nil
}

func (b *BBoltDataStore) Close() error {
	return b.db.Close()
}

func (b *BBoltDataStore) PutTag(ctx context.Context, tag objects.Tag) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		key := fmt.Sprintf("%s:%s", tag.Name, tag.Version)
		data, err := json.Marshal(tag)

		if err != nil {
			return err
		}

		return bucket.Put([]byte(key), data)
	})
}

func (b *BBoltDataStore) PutTags(ctx context.Context, tags []objects.Tag) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		for _, tag := range tags {
			data, err := json.Marshal(tag)

			if err != nil {
				return err
			}

			key := fmt.Sprintf("%s:%s", tag.Name, tag.Version)

			if err := bucket.Put([]byte(key), data); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BBoltDataStore) ResolveTag(ctx context.Context, name, version string) (string, bool, error) {
	var rootHash string
	var found bool

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		key := fmt.Sprintf("%s:%s", name, version)
		val := bucket.Get([]byte(key))

		if val == nil {
			return nil
		}

		var tag objects.Tag

		if err := json.Unmarshal(val, &tag); err != nil {
			return err
		}

		rootHash = tag.RootHash
		found = true

		return nil
	})

	return rootHash, found, err
}

func (b *BBoltDataStore) ListTags(ctx context.Context, name string) ([]objects.Tag, error) {
	var tags []objects.Tag

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		cursor := bucket.Cursor()
		prefix := []byte(name + ":")

		for k, v := cursor.Seek(prefix); k != nil && bytesHasPrefix(k, prefix); k, v = cursor.Next() {
			var tag objects.Tag

			if err := json.Unmarshal(v, &tag); err != nil {
				return err
			}

			tags = append(tags, tag)
		}
		return nil
	})

	return tags, err
}

func (b *BBoltDataStore) DeleteTag(ctx context.Context, name, version string) (bool, error) {
	existed := false
	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		key := []byte(fmt.Sprintf("%s:%s", name, version))

		if bucket.Get(key) != nil {
			existed = true
			return bucket.Delete(key)
		}

		return nil
	})

	return existed, err
}

func bytesHasPrefix(s, prefix []byte) bool {
	return len(s) >= len(prefix) && string(s[:len(prefix)]) == string(prefix)
}
