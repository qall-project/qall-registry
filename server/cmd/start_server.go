package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"google.golang.org/grpc"

	"qall-registry-server/internal/server"
	"qall-registry-server/pkg/blockstores"
	bslocal "qall-registry-server/pkg/blockstores/local"
	bs3 "qall-registry-server/pkg/blockstores/object"
	"qall-registry-server/pkg/core"
	ds "qall-registry-server/pkg/datastores/local"
	pb "qall-registry-server/pkg/server/protobuf/registry_api_v1"

	srv "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"google.golang.org/grpc/metadata"
)

func main() {
	dir := flag.String("directory", "/.registry-data", "base folder path")
	grpcPort := flag.String("grpc-port", "50051", "gRPC server port")
	restPort := flag.String("port", "50052", "REST server port")
	blockstore := flag.String("blockstore", "local", "block store backend: local or s3")
	s3Endpoint := flag.String("s3-endpoint", "", "S3 endpoint, with scheme (e.g. http://s3:9000 or https://s3.example.com)")
	s3Region := flag.String("s3-region", "", "S3 region")
	s3AccessKey := flag.String("s3-access-key", "", "S3 access key")
	s3SecretKey := flag.String("s3-secret-key", "", "S3 secret key")
	s3Bucket := flag.String("s3-bucket", "", "S3 bucket name")
	flag.Parse()

	log.Println(*dir, *grpcPort, *restPort)

	workingDir, _ := os.Getwd()
	dataRoot := filepath.Join(workingDir, *dir)
	_ = os.MkdirAll(dataRoot, 0755)

	log.Printf("Starting Qall Registry Server with root: %s", dataRoot)

	var bStore blockstores.BlockStore
	var err error

	switch *blockstore {
	case "local":
		bStore, err = bslocal.NewLocalBlockStore(dataRoot)
		if err != nil {
			log.Fatalf("Failed to initialize Local BlockStore: %v", err)
		}
	case "s3":
		bStore, err = bs3.NewS3BlockStore(bs3.S3Config{
			Endpoint:  *s3Endpoint,
			Region:    *s3Region,
			AccessKey: *s3AccessKey,
			SecretKey: *s3SecretKey,
			Bucket:    *s3Bucket,
		})
		if err != nil {
			log.Fatalf("Failed to initialize S3 BlockStore: %v", err)
		}
	default:
		log.Fatalf("unknown block store backend: %s", *blockstore)
	}

	ds, err := ds.NewLocalDataStore(dataRoot)
	if err != nil {
		log.Fatalf("Failed to initialize Local DataStore: %v", err)
	}
	defer ds.Close()

	registryCore := core.NewRegistryCore(bStore, ds)

	apiImpl := server.NewApiV1Server(registryCore)

	if err = start(context.Background(), apiImpl, *grpcPort, *restPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func start(ctx context.Context, apiImpl *server.ApiV1Server, grpcPort string, restPort string) error {
	go func() error {
		mux := srv.NewServeMux(
			srv.WithMetadata(func(_ context.Context, req *http.Request) metadata.MD {
				return metadata.New(map[string]string{
					"grpcgateway-http-path": req.URL.Path,
				})
			}))

		err := pb.RegisterApiHandlerServer(ctx, mux, apiImpl)
		if err != nil {
			return err
		}

		err = http.ListenAndServe("0.0.0.0:"+restPort, mux)
		if err != nil {
			return err
		}

		return nil
	}()

	maxMsgSize := 1024 * 1024 * 2000
	grpcServer := grpc.NewServer(
		grpc.EmptyServerOption{},
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)
	pb.RegisterApiServer(grpcServer, apiImpl)

	listener, err := net.Listen("tcp", ":"+grpcPort)

	if err != nil {
		return err
	}

	err = grpcServer.Serve(listener)

	if err != nil {
		return err
	}

	return nil
}
