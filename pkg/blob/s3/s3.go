package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config contém as credenciais e parâmetros de conexão com o S3.
type Config struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

// Storage implementa a interface blob.Provider para AWS S3 ou MinIO.
type Storage struct {
	config Config
}

// New cria uma nova instância do provedor S3.
func New(cfg Config) *Storage {
	return &Storage{config: cfg}
}

func (s *Storage) getClient(ctx context.Context) (*s3.Client, error) {
	if s.config.Bucket == "" {
		return nil, fmt.Errorf("s3_bucket not configured")
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(s.config.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s.config.AccessKey,
			s.config.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	if s.config.Endpoint != "" {
		cfg.BaseEndpoint = aws.String(s.config.Endpoint)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

// Put implementa blob.Provider.
func (s *Storage) Put(ctx context.Context, key string, data []byte) error {
	client, err := s.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

// Get implementa blob.Provider.
func (s *Storage) Get(ctx context.Context, key string) ([]byte, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// Close satisfaz a interface (neste caso, o client da AWS não exige fechamento manual).
func (s *Storage) Close() error {
	return nil
}
