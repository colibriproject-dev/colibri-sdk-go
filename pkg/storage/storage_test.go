package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"testing"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
)

const (
	BUCKET = "my-bucket"
	ID     = "a0264d4f-cb3b-41e7-9632-2f8f86f6b28d"
)

func TestMain(m *testing.M) {
	test.InitializeTestLocalstack()

	Initialize()

	m.Run()
}

func TestStorage(t *testing.T) {
	ctx := context.Background()
	file, err := generateFile()

	t.Run("Should upload with success", func(t *testing.T) {
		expected := fmt.Sprintf("/%s/%s", BUCKET, ID)

		assert.NoError(t, err)

		result, err := UploadFile(ctx, BUCKET, ID, file)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, strings.Contains(result, expected))
	})

	t.Run("Should download with success", func(t *testing.T) {
		_, err := UploadFile(ctx, BUCKET, ID, file)
		assert.NoError(t, err)
		result, err := DownloadFile(ctx, BUCKET, ID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Should delete with success", func(t *testing.T) {
		err := DeleteFile(ctx, BUCKET, ID)
		assert.NoError(t, err)
	})
}

func generateFile() (*multipart.File, error) {
	filePath := "./../../development-environment/storage/file.txt"
	fieldName := "file"
	body := new(bytes.Buffer)

	multipartWriter := multipart.NewWriter(body)
	defer multipartWriter.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	fileWriter, err := multipartWriter.CreateFormFile(fieldName, filePath)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(fileWriter, file); err != nil {
		return nil, err
	}

	filePtr := new(multipart.File)
	*filePtr = file
	return filePtr, nil
}
