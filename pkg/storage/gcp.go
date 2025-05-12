package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	gcp_storage "cloud.google.com/go/storage"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

type gcpStorage struct {
	client *gcp_storage.Client
}

// newGcpStorage creates a new instance of gcpStorage by initializing the client.
//
// No parameters.
// Returns a pointer to gcpStorage.
func newGcpStorage() *gcpStorage {
	ctx := context.Background()

	client, err := gcp_storage.NewClient(ctx)
	if err != nil {
		logging.Fatal(ctx).Err(err).Msg(connectionErrorMsg)
	}

	return &gcpStorage{client}

}

// downloadFile downloads a file from the storage provider.
//
// ctx: the context for the operation.
// bucket: the storage bucket from which the file is downloaded.
// key: the key or identifier of the file to be downloaded.
// Returns a file pointer and an error.
func (s *gcpStorage) downloadFile(ctx context.Context, bucket, key string) (*os.File, error) {
	file, err := os.CreateTemp("", "tempFile")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Bucket(bucket).Object(key).NewReader(ctx); err != nil {
		return nil, err
	}

	fileReader, err := s.client.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer fileReader.Close()

	if _, err := io.Copy(file, fileReader); err != nil {
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
func (s *gcpStorage) uploadFile(ctx context.Context, bucket, key string, file *multipart.File) (string, error) {
	writer := s.client.Bucket(bucket).Object(key).NewWriter(ctx)
	writer.CacheControl = "no-cache, max-age=1"
	if _, err := io.Copy(writer, *file); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, key), nil
}

// deleteFile deletes a file from the storage provider.
//
// ctx: the context for the operation.
// bucket: the storage bucket from which the file is deleted.
// key: the key or identifier of the file to be deleted.
// Returns an error.
func (s *gcpStorage) deleteFile(ctx context.Context, bucket, key string) error {
	return s.client.Bucket(bucket).Object(key).Delete(ctx)
}
