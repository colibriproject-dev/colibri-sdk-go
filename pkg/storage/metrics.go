package storage

import (
	"context"
	"io"
	"mime/multipart"
	"os"
	"sync/atomic"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
)

// Metric names and attributes emitted by the storage module. They move to the SDK metric
// catalog once it exists (#233).
//
// bucket and key are deliberately absent: key is unbounded and both belong on the spans only.
const (
	metricOperation         = "storage.operation"
	metricOperationDuration = "storage.operation.duration"
	metricTransferred       = "storage.transferred"

	attrOperation = "operation"
	attrResult    = "result"

	operationUpload   = "upload"
	operationDownload = "download"
	operationDelete   = "delete"

	resultSuccess = "success"
	resultError   = "error"
)

// storageMetrics holds the instruments of the module, created once per Initialize.
type storageMetrics struct {
	operation   colibrimonitoringbase.Counter
	duration    colibrimonitoringbase.HistogramRecorder
	transferred colibrimonitoringbase.HistogramRecorder
}

var moduleMetrics atomic.Pointer[storageMetrics]

// noopMetrics is what the module records into before Initialize has created the instruments.
var noopMetrics = &storageMetrics{
	operation:   colibrimonitoringbase.NoopCounter(),
	duration:    colibrimonitoringbase.NoopHistogram(),
	transferred: colibrimonitoringbase.NoopHistogram(),
}

func initMetrics() {
	moduleMetrics.Store(&storageMetrics{
		operation: monitoring.Counter(metricOperation,
			"Number of storage operations", "{operation}"),
		duration: monitoring.Histogram(metricOperationDuration,
			"Duration of storage operations", "s"),
		transferred: monitoring.Histogram(metricTransferred,
			"Size of the files uploaded to and downloaded from the storage", "By"),
	})
}

func currentMetrics() *storageMetrics {
	if m := moduleMetrics.Load(); m != nil {
		return m
	}

	return noopMetrics
}

// operationAttrs are the attribute sets of an operation, built once since both the
// operations and the results are a fixed set.
type operationAttrs struct {
	success     colibrimonitoringbase.Attrs
	failure     colibrimonitoringbase.Attrs
	transferred colibrimonitoringbase.Attrs
}

func newOperationAttrs(operation string) operationAttrs {
	return operationAttrs{
		success:     colibrimonitoringbase.NewAttrs(attrOperation, operation, attrResult, resultSuccess),
		failure:     colibrimonitoringbase.NewAttrs(attrOperation, operation, attrResult, resultError),
		transferred: colibrimonitoringbase.NewAttrs(attrOperation, operation),
	}
}

var (
	uploadAttrs   = newOperationAttrs(operationUpload)
	downloadAttrs = newOperationAttrs(operationDownload)
	deleteAttrs   = newOperationAttrs(operationDelete)
)

// noTransfer marks an operation that moves no file content, or whose size is unknown.
const noTransfer int64 = -1

// record counts an operation and its duration and, when it succeeded, the bytes it moved.
func record(ctx context.Context, attrs operationAttrs, start time.Time, transferred int64, err error) {
	m := currentMetrics()
	elapsed := time.Since(start).Seconds()

	if err != nil {
		m.operation.AddAttrs(ctx, 1, attrs.failure)
		m.duration.RecordAttrs(ctx, elapsed, attrs.failure)
		return
	}

	m.operation.AddAttrs(ctx, 1, attrs.success)
	m.duration.RecordAttrs(ctx, elapsed, attrs.success)
	if transferred >= 0 {
		m.transferred.RecordAttrs(ctx, float64(transferred), attrs.transferred)
	}
}

// uploadSize returns how many bytes of the file are left to read, restoring its position so
// the upload still reads it whole. It returns noTransfer when the file cannot seek.
func uploadSize(file *multipart.File) int64 {
	if file == nil || *file == nil {
		return noTransfer
	}

	f := *file
	current, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return noTransfer
	}

	end, err := f.Seek(0, io.SeekEnd)
	if _, restoreErr := f.Seek(current, io.SeekStart); err != nil || restoreErr != nil {
		return noTransfer
	}

	return end - current
}

// downloadSize returns the size of the downloaded file, or noTransfer when it is unknown.
func downloadSize(file *os.File) int64 {
	if file == nil {
		return noTransfer
	}

	info, err := file.Stat()
	if err != nil {
		return noTransfer
	}

	return info.Size()
}
