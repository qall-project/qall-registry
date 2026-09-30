package datastores

import (
	"context"
	"qall-registry-server/pkg/objects"
)

type DataStore interface {
	PutTag(ctx context.Context, tag objects.Tag) error
	PutTags(ctx context.Context, tags []objects.Tag) error
	ResolveTag(ctx context.Context, name, version string) (string, bool, error)
	ListTags(ctx context.Context, name string) ([]objects.Tag, error)
	DeleteTag(ctx context.Context, name, version string) (bool, error)
}
