package storage

import (
	"context"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"
)

type awsStorage struct {
	s3Service  *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
}

// newAwsStorage creates a new awsStorage instance and initializes the S3 service, uploader, and downloader.
//
// No parameters.
// Returns a pointer to the awsStorage instance.
func newAwsStorage() *awsStorage {
	return &awsStorage{
		s3Service:  s3.New(cloud.GetAwsSession()),
		uploader:   s3manager.NewUploader(cloud.GetAwsSession()),
		downloader: s3manager.NewDownloader(cloud.GetAwsSession()),
	}
}

// downloadFile downloads a file from the storage provider.
//
// ctx: the context for the operation.
// bucket: the storage bucket from which the file is downloaded.
// key: the key or identifier of the file to be downloaded.
// Returns a file pointer and an error.
func (s *awsStorage) downloadFile(ctx context.Context, bucket, key string) (*os.File, error) {
	file, err := os.CreateTemp("", "tempFile")
	if err != nil {
		return nil, err
	}

	if _, err := s.downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		return nil, err
	}

	return file, nil
}

// uploadFile uploads a file to the storage provider.
//
// ctx: the context for the operation.
// bucket: the storage bucket to upload the file to.
// key: the key or identifier of the file to be uploaded.
// file: the file to be uploaded.
// Returns the location of the uploaded file and an error, if any.
func (s *awsStorage) uploadFile(ctx context.Context, bucket, key string, file *multipart.File) (string, error) {
	result, err := s.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   *file,
	})
	if err != nil {
		return "", err
	}

	return result.Location, nil
}

// deleteFile deletes a file from the storage provider.
//
// ctx: the context for the operation.
// bucket: the storage bucket from which the file is deleted.
// key: the key or identifier of the file to be deleted.
// Returns an error.
func (s *awsStorage) deleteFile(ctx context.Context, bucket, key string) error {
	_, err := s.s3Service.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return err
}
