package storage

import (
	"context"
	"mime/multipart"
	"os"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
)

const (
	storageTransactionMsg      string = "Storage"
	storageAlreadyConnectedMsg string = "storage provider already connected"
	storageConnectedMsg        string = "storage provider connected"
	connectionErrorMsg         string = "an error occurred when trying to connect to the storage provider"
)

type storage interface {
	downloadFile(ctx context.Context, bucket, key string) (*os.File, error)
	uploadFile(ctx context.Context, bucket, key string, file *multipart.File) (string, error)
	deleteFile(ctx context.Context, bucket, key string) error
}

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

	initMetrics()

	logging.Info(context.Background()).Msg(storageConnectedMsg)
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

	start := time.Now()
	file, err := instance.downloadFile(ctx, bucket, key)
	record(ctx, downloadAttrs, start, downloadSize(file), err)

	return file, err
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

	size := uploadSize(file)
	start := time.Now()
	url, err := instance.uploadFile(ctx, bucket, key, file)
	record(ctx, uploadAttrs, start, size, err)

	return url, err
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

	start := time.Now()
	err := instance.deleteFile(ctx, bucket, key)
	record(ctx, deleteAttrs, start, noTransfer, err)

	return err
}
