package s3client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Client struct {
	Client     *minio.Client
	BucketName string
}

func NewS3Client() (*S3Client, error) {
	endpoint, err := requiredEnv("S3_ENDPOINT")
	if err != nil {
		return nil, err
	}
	accessKeyID, err := requiredEnv("AWS_ACCESS_KEY_ID")
	if err != nil {
		return nil, err
	}
	secretAccessKey, err := requiredEnv("AWS_SECRET_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	region, err := requiredEnv("AWS_REGION")
	if err != nil {
		return nil, err
	}
	bucketName, err := requiredEnv("S3_BUCKET")
	if err != nil {
		return nil, err
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	found, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("check S3 bucket %q: %w", bucketName, err)
	}

	if !found {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: region})
		if err != nil {
			return nil, fmt.Errorf("create S3 bucket %q: %w", bucketName, err)
		}
	}

	return &S3Client{
		Client:     client,
		BucketName: bucketName,
	}, nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s environment variable is required", name)
	}
	return value, nil
}

func (s *S3Client) PutObject(key string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	_, err := s.Client.PutObject(ctx, s.BucketName, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

func (s *S3Client) RemoveObject(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	err := s.Client.RemoveObject(ctx, s.BucketName, key, minio.RemoveObjectOptions{})
	return err
}

func (s *S3Client) GetObject(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	obj, err := s.Client.GetObject(ctx, s.BucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer func(obj *minio.Object) {
		err := obj.Close()
		if err != nil {
			return
		}
	}(obj)

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *S3Client) ObjectExists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := s.Client.StatObject(ctx, s.BucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}
