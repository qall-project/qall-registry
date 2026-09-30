package server

import (
	"context"
	pb "qall-registry-server/internal/server/protobuf/registry_api_v1"
	"qall-registry-server/pkg/core"
	"qall-registry-server/pkg/objects"
)

type ApiV1Server struct {
	pb.UnimplementedApiServer
	core *core.RegistryCore
}

func NewApiV1Server(core *core.RegistryCore) *ApiV1Server {
	return &ApiV1Server{core: core}
}

func (s *ApiV1Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	missing, err := s.core.CheckBlocks(ctx, req.Hashes)

	if err != nil {
		return nil, err
	}

	return &pb.CheckResponse{MissingHashes: missing, TotalCount: int32(len(missing))}, nil
}

func (s *ApiV1Server) Push(ctx context.Context, req *pb.PushRequest) (*pb.PushResponse, error) {
	coreBlocks := make([]objects.Block, len(req.Blocks))

	for i, b := range req.Blocks {
		coreBlocks[i] = objects.Block{Hash: b.Hash, Data: b.Data}
	}

	written_blocks_count, err := s.core.PushBlocks(ctx, coreBlocks)

	if err != nil {
		return nil, err
	}

	return &pb.PushResponse{WrittenBlocksCount: written_blocks_count}, nil
}

func (s *ApiV1Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	blocks, err := s.core.GetBlocks(ctx, req.Hashes)

	if err != nil {
		return nil, err
	}

	pbBlocks := make([]*pb.Block, len(blocks))

	for i, b := range blocks {
		pbBlocks[i] = &pb.Block{Hash: b.Hash, Data: b.Data}
	}

	pbResp := &pb.GetResponse{Blocks: pbBlocks, TotalCount: int32(len(pbBlocks))}

	return pbResp, nil
}

func (s *ApiV1Server) Tag(ctx context.Context, req *pb.TagRequest) (*pb.TagResponse, error) {
	tag := objects.Tag{
		Name:     req.Tag.Name,
		Version:  req.Tag.Version,
		RootHash: req.Tag.RootHash,
	}

	uri, err := s.core.Tag(ctx, tag)

	if err != nil {
		return nil, err
	}

	return &pb.TagResponse{Uri: uri}, nil
}

func (s *ApiV1Server) Resolve(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveResponse, error) {
	rootHash, _, err := s.core.Resolve(ctx, req.Name, req.Version)

	if err != nil {
		return nil, err
	}

	return &pb.ResolveResponse{RootHash: rootHash}, nil
}

func (s *ApiV1Server) ListTags(ctx context.Context, req *pb.ListTagsRequest) (*pb.ListTagsResponse, error) {
	tags, err := s.core.ListTags(ctx, req.Name)

	if err != nil {
		return nil, err
	}

	pbVersions := make([]*pb.Tag, len(tags))

	for i, t := range tags {
		pbVersions[i] = &pb.Tag{Name: t.Name, Version: t.Version, RootHash: t.RootHash}
	}

	return &pb.ListTagsResponse{Tags: pbVersions, TotalCount: int32(len(pbVersions))}, nil
}

func (s *ApiV1Server) Untag(ctx context.Context, req *pb.UntagRequest) (*pb.UntagResponse, error) {
	deleted, err := s.core.Untag(ctx, req.Name, req.Version)

	if err != nil {
		return nil, err
	}

	return &pb.UntagResponse{Deleted: deleted}, nil
}
