package s3client

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"os"
	"time"
)

type S3Client struct {
	Client     *minio.Client
	BucketName string
}

func NewS3Client() *S3Client {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("S3_BUCKET")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		println(err)
	}

	found, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		fmt.Println("No S3 storage found")
	}

	if !found {
		err = client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: "us-east-1"})
		if err != nil {
			fmt.Println("No S3 storage found, can't create S3 bucket")
		} else {
			fmt.Println("Bucket created")
		}
	}

	return &S3Client{
		Client:     client,
		BucketName: bucketName,
	}
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
