package storage

import (
	"context"
	"mime/multipart"
	"os"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

const (
	storageTransactionMsg      string = "Storage"
	storageAlreadyConnectedMsg string = "storage provider already connected"
	storageConnectedMsg        string = "storage provider connected"
	connectionErrorMsg         string = "an error occurred when trying to connect to the storage provider"
	closeErrorMsg              string = "an error occurred when trying to close the storage provider client"
)

type storage interface {
	downloadFile(ctx context.Context, bucket, key string) (*os.File, error)
	uploadFile(ctx context.Context, bucket, key string, file *multipart.File) (string, error)
	deleteFile(ctx context.Context, bucket, key string) error
}

// storageCloser is implemented by the providers holding a client that has to be released. The
// AWS provider builds its services on a session and keeps no connection, so it does not
// implement it.
type storageCloser interface {
	close() error
}

type storageObserver struct{}

var instance storage

// Initialize initializes the storage provider based on the configured cloud provider.
func Initialize() {
	if instance != nil {
		logging.Info(context.Background()).Msg(storageAlreadyConnectedMsg)
		return
	}

	switch config.CLOUD {
	case config.CLOUD_AWS:
		instance = newAwsStorage()
	case config.CLOUD_GCP, config.CLOUD_FIREBASE:
		instance = newGcpStorage()
	}

	observer.AttachWithPriority(storageObserver{}, observer.PriorityDefault)
	logging.Info(context.Background()).Msg(storageConnectedMsg)
}

// Close releases the storage client. Uploading and deleting are side effects, so it runs in
// the closing phase of the graceful shutdown, once the work using them has been drained.
func (o storageObserver) Close() {
	closer, ok := instance.(storageCloser)
	if !ok {
		return
	}

	if err := closer.close(); err != nil {
		logging.Error(context.Background()).Err(err).Msg(closeErrorMsg)
	}
}

// DownloadFile downloads a file from the storage provider.
func DownloadFile(ctx context.Context, bucket, key string) (*os.File, error) {
	txn := monitoring.GetTransactionInContext(ctx)
	if txn != nil {
		segment := monitoring.StartTransactionSegment(ctx, storageTransactionMsg, map[string]string{
			"method": "Download",
			"bucket": bucket,
			"key":    key,
		})
		defer monitoring.EndTransactionSegment(segment)
	}

	return instance.downloadFile(ctx, bucket, key)
}

// UploadFile uploads a file to the storage provider.
func UploadFile(ctx context.Context, bucket, key string, file *multipart.File) (string, error) {
	txn := monitoring.GetTransactionInContext(ctx)
	if txn != nil {
		segment := monitoring.StartTransactionSegment(ctx, storageTransactionMsg, map[string]string{
			"method": "Upload",
			"bucket": bucket,
			"key":    key,
		})
		defer monitoring.EndTransactionSegment(segment)
	}

	return instance.uploadFile(ctx, bucket, key, file)
}

// DeleteFile deletes a file from the storage provider.
func DeleteFile(ctx context.Context, bucket, key string) error {
	txn := monitoring.GetTransactionInContext(ctx)
	if txn != nil {
		segment := monitoring.StartTransactionSegment(ctx, storageTransactionMsg, map[string]string{
			"method": "Delete",
			"bucket": bucket,
			"key":    key,
		})
		defer monitoring.EndTransactionSegment(segment)
	}

	return instance.deleteFile(ctx, bucket, key)
}
