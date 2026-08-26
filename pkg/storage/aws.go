package storage

import (
	"context"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"
)

type awsStorage struct {
	s3Service *s3.Client
	transfer  *transfermanager.Client
}

// newAwsStorage creates a new awsStorage instance and initializes the S3 service and the
// transfer manager, which handles the multipart upload and download of large objects.
//
// No parameters.
// Returns a pointer to the awsStorage instance.
func newAwsStorage() *awsStorage {
	// path style keeps the bucket in the URL path instead of the host, which is what the
	// emulators used outside a cloud environment serve
	client := s3.NewFromConfig(cloud.GetAwsConfig(), func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &awsStorage{
		s3Service: client,
		transfer:  transfermanager.New(client),
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

	if _, err := s.transfer.DownloadObject(ctx, &transfermanager.DownloadObjectInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		WriterAt: file,
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
	result, err := s.transfer.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   *file,
	})
	if err != nil {
		return "", err
	}

	return aws.ToString(result.Location), nil
}

// deleteFile deletes a file from the storage provider.
//
// ctx: the context for the operation.
// bucket: the storage bucket from which the file is deleted.
// key: the key or identifier of the file to be deleted.
// Returns an error.
func (s *awsStorage) deleteFile(ctx context.Context, bucket, key string) error {
	_, err := s.s3Service.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return err
}
