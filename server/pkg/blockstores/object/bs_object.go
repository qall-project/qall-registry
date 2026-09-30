package object

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
)

type S3Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
}

type S3BlockStore struct {
	client *s3.Client
	bucket string
}

func NewS3BlockStore(config S3Config) (*S3BlockStore, error) {
	ctx := context.Background()

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(config.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				config.AccessKey,
				config.SecretKey,
				"",
			),
		),
	)

	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(config.Endpoint)
		o.UsePathStyle = true
	})

	return &S3BlockStore{client: client, bucket: config.Bucket}, nil
}

func (s *S3BlockStore) key(hash string) string {
	return path.Join(hash[:2], hash[2:])
}

func (s *S3BlockStore) Has(ctx context.Context, hash string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(hash)),
	})

	if err != nil {
		if isNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (s *S3BlockStore) Put(ctx context.Context, hash string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(hash)),
		Body:   bytes.NewReader(data),
	})

	if err != nil {
		return fmt.Errorf("failed to put block %s: %w", hash, err)
	}

	return nil
}

func (s *S3BlockStore) Get(ctx context.Context, hash string) ([]byte, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(hash)),
	})

	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("block not found: %s", hash)
		}

		return nil, err
	}

	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func isNotFound(err error) bool {
	var responseError *awshttp.ResponseError
	if errors.As(err, &responseError) && responseError.HTTPStatusCode() == http.StatusNotFound {
		return true
	}

	return false
}
