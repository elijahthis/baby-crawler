package storage

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(ctx context.Context, bucket string, endpoint, user, password string) (*S3Storage, error) {
	creds := credentials.NewStaticCredentialsProvider(user, password, "")
	const defaultRegion = "us-east-1"

	// cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(creds), config.WithRegion(defaultRegion))
	// if err != nil {
	// 	return nil, err
	// }

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(creds),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               endpoint, // e.g. "http://localhost:9000"
					HostnameImmutable: true,     // Required for MinIO
				}, nil
			},
		)),
	)
	if err != nil {
		return nil, err
	}

	return &S3Storage{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

func (s *S3Storage) Save(ctx context.Context, key string, data []byte) error {
	// key = shared.CleanKey(key)

	log.Info().Msgf("Saving to S3. Key: %s\n\n", key)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})

	if err == nil {
		log.Info().Msgf("Uploaded to S3: %s\n", key)
	}
	return err
}

func (s *S3Storage) Load(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
