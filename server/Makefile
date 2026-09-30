.PHONY: build test test-s3-local
MAKEFLAGS += --no-print-directory

build:
	go build -ldflags="-s -w" -o qall-registry-server ./cmd/start_server.go

# local.env values are injected here.
-include ../infrastructure/local/vars/local.env

# The environment variable bellow are defaulted to the local MinIO value from local.env
S3_REGION := $(or $(S3_REGION),$(OBJECT_STORAGE_REGION))
S3_ACCESS_KEY := $(or $(S3_ACCESS_KEY),$(OBJECT_STORAGE_ACCESS_KEY))
S3_SECRET_KEY := $(or $(S3_SECRET_KEY),$(OBJECT_STORAGE_SECRET_KEY))
S3_BUCKET := $(or $(S3_BUCKET),$(OBJECT_STORAGE_BUCKET_NAME))

test:
	S3_ENDPOINT="$(S3_ENDPOINT)" S3_REGION="$(S3_REGION)" S3_ACCESS_KEY="$(S3_ACCESS_KEY)" S3_SECRET_KEY="$(S3_SECRET_KEY)" S3_BUCKET="$(S3_BUCKET)" go test -v ./...

# Run the S3 integration tests against the local MinIO stack.
# Warning: start the local MinIO server first, from infrastructure/local/:
#   make start
test-s3-local:
	$(MAKE) test S3_ENDPOINT=http://localhost:9000