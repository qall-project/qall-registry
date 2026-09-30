package pgsql

import (
	"context"
	"errors"
	"qall-registry-server/pkg/objects"
)

type PostgresDataStore struct{}

func NewPostgresDataStore() *PostgresDataStore { return &PostgresDataStore{} }

func (p *PostgresDataStore) PutTag(ctx context.Context, tag objects.Tag) error {
	return errors.New("postgres datastore not implemented")
}
func (p *PostgresDataStore) ResolveTag(ctx context.Context, name, version string) (string, bool, error) {
	return "", false, errors.New("postgres datastore not implemented")
}
func (p *PostgresDataStore) ListTags(ctx context.Context, name string) ([]objects.Tag, error) {
	return nil, errors.New("postgres datastore not implemented")
}
func (p *PostgresDataStore) DeleteTag(ctx context.Context, name, version string) (bool, error) {
	return false, errors.New("postgres datastore not implemented")
}
