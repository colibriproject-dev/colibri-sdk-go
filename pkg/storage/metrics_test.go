package storage

import (
	"context"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/monitoringtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errFakeStorage = errors.New("storage unavailable")

// fakeStorage stands in for a provider, returning a file of a known size on download.
type fakeStorage struct {
	err          error
	downloadPath string
	uploaded     []byte
}

func (f *fakeStorage) downloadFile(context.Context, string, string) (*os.File, error) {
	if f.err != nil {
		return nil, f.err
	}

	return os.Open(f.downloadPath)
}

func (f *fakeStorage) uploadFile(_ context.Context, _, _ string, file *multipart.File) (string, error) {
	if f.err != nil {
		return "", f.err
	}

	buf := make([]byte, 64)
	n, _ := (*file).Read(buf)
	f.uploaded = buf[:n]

	return "url", nil
}

func (f *fakeStorage) deleteFile(context.Context, string, string) error {
	return f.err
}

// setupStorageMetricsTest swaps the provider for a fake and creates the instruments against a
// metric recorder, the way Initialize does.
func setupStorageMetricsTest(t *testing.T, err error) (*fakeStorage, *monitoringtest.Recorder) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "download.txt")
	require.NoError(t, os.WriteFile(path, []byte("0123456789"), 0o600))

	fake := &fakeStorage{err: err, downloadPath: path}
	previousInstance := instance
	previousMetrics := moduleMetrics.Load()
	instance = fake

	recorder := monitoringtest.Install(t)
	initMetrics()

	t.Cleanup(func() {
		instance = previousInstance
		moduleMetrics.Store(previousMetrics)
	})

	return fake, recorder
}

// uploadTestFile returns a multipart file holding content, the way a request handler gets it.
func uploadTestFile(t *testing.T, content string) *multipart.File {
	t.Helper()

	path := filepath.Join(t.TempDir(), "upload.txt")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	var file multipart.File = f
	return &file
}

func TestStorageMetrics(t *testing.T) {
	ctx := context.Background()

	t.Run("Should count successful operations with their duration and transferred bytes", func(t *testing.T) {
		fake, recorder := setupStorageMetricsTest(t, nil)

		_, err := UploadFile(ctx, BUCKET, ID, uploadTestFile(t, "hello"))
		require.NoError(t, err)
		downloaded, err := DownloadFile(ctx, BUCKET, ID)
		require.NoError(t, err)
		_ = downloaded.Close()
		require.NoError(t, DeleteFile(ctx, BUCKET, ID))

		assert.Equal(t, "hello", string(fake.uploaded), "measuring the size must not consume the upload")

		operations := recorder.Metric(t, metricOperation)
		monitoringtest.AssertShape(t, operations, "{operation}", attrOperation, attrResult)
		for _, operation := range []string{operationUpload, operationDownload, operationDelete} {
			assert.Equal(t, int64(1), monitoringtest.CounterValue(t, operations,
				attrOperation, operation, attrResult, resultSuccess), operation)
		}

		duration := recorder.Metric(t, metricOperationDuration)
		monitoringtest.AssertShape(t, duration, "s", attrOperation, attrResult)
		count, _ := monitoringtest.HistogramCount(t, duration, attrOperation, operationDelete, attrResult, resultSuccess)
		assert.Equal(t, uint64(1), count)

		transferred := recorder.Metric(t, metricTransferred)
		monitoringtest.AssertShape(t, transferred, "By", attrOperation)
		_, uploaded := monitoringtest.HistogramCount(t, transferred, attrOperation, operationUpload)
		assert.InDelta(t, 5.0, uploaded, 1e-9)
		_, downloadedBytes := monitoringtest.HistogramCount(t, transferred, attrOperation, operationDownload)
		assert.InDelta(t, 10.0, downloadedBytes, 1e-9)
	})

	t.Run("Should count failed operations without transferred bytes", func(t *testing.T) {
		_, recorder := setupStorageMetricsTest(t, errFakeStorage)

		_, err := UploadFile(ctx, BUCKET, ID, uploadTestFile(t, "hello"))
		assert.ErrorIs(t, err, errFakeStorage)
		_, err = DownloadFile(ctx, BUCKET, ID)
		assert.ErrorIs(t, err, errFakeStorage)
		assert.ErrorIs(t, DeleteFile(ctx, BUCKET, ID), errFakeStorage)

		operations := recorder.Metric(t, metricOperation)
		for _, operation := range []string{operationUpload, operationDownload, operationDelete} {
			assert.Equal(t, int64(1), monitoringtest.CounterValue(t, operations,
				attrOperation, operation, attrResult, resultError), operation)
		}
		assert.NotContains(t, recorder.Collect(t), metricTransferred)
	})
}

func TestUploadSize(t *testing.T) {
	t.Run("Should measure the bytes left to read and keep the position", func(t *testing.T) {
		file := uploadTestFile(t, "0123456789")
		_, err := (*file).Seek(4, 0)
		require.NoError(t, err)

		assert.Equal(t, int64(6), uploadSize(file))

		position, err := (*file).Seek(0, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(4), position)
	})

	t.Run("Should report an unknown size for a missing file", func(t *testing.T) {
		assert.Equal(t, noTransfer, uploadSize(nil))

		var empty multipart.File
		assert.Equal(t, noTransfer, uploadSize(&empty))
	})
}

func TestDownloadSize(t *testing.T) {
	t.Run("Should report an unknown size for a missing file", func(t *testing.T) {
		assert.Equal(t, noTransfer, downloadSize(nil))
	})
}
