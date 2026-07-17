package s3

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
)

var _ ports.ObjectStorage = (*Storage)(nil)

type Storage struct {
	client     *awss3.Client
	downloader *manager.Downloader
	uploader   *manager.Uploader
}

func NewStorage(client *awss3.Client) *Storage {
	return &Storage{
		client:     client,
		downloader: manager.NewDownloader(client),
		uploader:   manager.NewUploader(client),
	}
}

func (s *Storage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key),
	})
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		// alguns backends retornam NoSuchKey; tratar também
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return false, nil
		}
		return false, fmt.Errorf("head object %s: %w", key, err)
	}
	return true, nil
}

func (s *Storage) Download(ctx context.Context, bucket, key, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create dest file: %w", err)
	}
	defer f.Close()
	if _, err := s.downloader.Download(ctx, f, &awss3.GetObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key),
	}); err != nil {
		return fmt.Errorf("download %s: %w", key, err)
	}
	return nil
}

func (s *Storage) Upload(ctx context.Context, bucket, key, srcPath, contentType string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src file: %w", err)
	}
	defer f.Close()
	if _, err := s.uploader.Upload(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key),
		Body: f, ContentType: aws.String(contentType),
	}); err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}

func (s *Storage) Delete(ctx context.Context, bucket, key string) error {
	if _, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}
