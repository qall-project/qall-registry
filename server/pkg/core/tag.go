package core

import (
	"context"
	"fmt"
	"log"
	"qall-registry-server/pkg/objects"
	"strings"
)

func (c *RegistryCore) ListTags(ctx context.Context, name string) ([]objects.Tag, error) {
	return c.dStore.ListTags(ctx, name)
}

func (c *RegistryCore) Untag(ctx context.Context, name string, version string) (bool, error) {
	return c.dStore.DeleteTag(ctx, name, version)
}

func (c *RegistryCore) Tag(ctx context.Context, tag objects.Tag) (string, error) {
	if err := validateName(tag.Name); err != nil {
		return "", err
	}

	if tag.Version == "" {
		tag.Version = "latest"
	}

	exists, err := c.bStore.Has(ctx, tag.RootHash)

	if err != nil {
		return "", err
	}

	if !exists {
		log.Printf("cannot create tag %s:%s, %s block not present", tag.Name, tag.Version, tag.RootHash)
		return "", fmt.Errorf("cannot tag workflow: root node hash %s does not exist", tag.RootHash)
	}

	if err := c.dStore.PutTag(ctx, tag); err != nil {
		return "", err
	}

	tagsToWrite := []objects.Tag{tag}

	if tag.Version != "latest" {
		tagsToWrite = append(tagsToWrite, objects.Tag{
			Name: tag.Name, Version: "latest", RootHash: tag.RootHash,
		})
	}

	ver, err := fmt.Sprintf("%s:%s", tag.Name, tag.Version), c.dStore.PutTags(ctx, tagsToWrite)

	if err != nil {
		return "", err
	}

	log.Printf("sucessfully created tag %s:%s with root hash %s", tag.Name, tag.Version, tag.RootHash)

	return ver, err
}

func (c *RegistryCore) Resolve(ctx context.Context, name, version string) (string, bool, error) {
	if version == "" {
		version = "latest"
	}

	return c.dStore.ResolveTag(ctx, name, version)
}

func validateName(name string) error {
	if strings.ContainsAny(name, ":/") {
		return fmt.Errorf("workflow name must not contain ':' or '/'")
	}

	if len(name) == 0 || len(name) > 128 {
		return fmt.Errorf("workflow name must be between 1 and 128 characters")
	}

	return nil
}
