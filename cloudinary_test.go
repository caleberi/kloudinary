package kloudinary

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	clientSecretEnv = "clientSecret"
	clientKeyEnv    = "clientKey"
	clientNameEnv   = "clientName"
)

func TestAssetUploadManager_UploadSingleFile(t *testing.T) {
	assetManager := newTestAssetManager(t)
	fileToUpload := testUploadPath(t, "test0.png")
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	result, err := assetManager.UploadSingleFile(ctx, fileToUpload)
	require.NoError(t, err)

	assert.NotNil(t, result)
	r := *result
	assert.Empty(t, r.Error.Message)
	assert.NotEmpty(t, r.PublicID)
	assert.NotEmpty(t, r.SecureURL)
}

func TestAssetUploadManager_UploadSingleFile_BinaryData(t *testing.T) {
	assetManager := newTestAssetManager(t)
	fileToUpload := testUploadPath(t, "test5.xml")
	file, err := os.Open(fileToUpload)
	assert.NoError(t, err)
	defer func() {
		_ = file.Close()
	}()

	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	result, err := assetManager.UploadSingleFile(ctx, file)
	require.NoError(t, err)

	assert.NotNil(t, result)
	r := *result
	assert.Empty(t, r.Error.Message)
	assert.NotEmpty(t, r.PublicID)
	assert.NotEmpty(t, r.SecureURL)

}

func TestAssetUploadManager_UploadMultipleFiles(t *testing.T) {
	assetManager := newTestAssetManager(t)
	var filesToUpload []any

	for i := range 3 {
		filesToUpload = append(
			filesToUpload,
			testUploadPath(t, fmt.Sprintf("test%d.png", i)),
		)
	}

	assetManager.MaxNumberOfConcurrentUploads = int64(len(filesToUpload))
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	startTime := time.Now()
	results := assetManager.UploadMultipleFiles(ctx, filesToUpload...)
	assert.NotEmpty(t, results)
	printTestStats(t, results, startTime, filesToUpload)
}

func TestLargeAssetUploadManager_UploadMultipleFiles(t *testing.T) {
	assetManager := newTestAssetUploadManagerLarge(t)
	uploadDir := testUploadDir(t)
	filesToUpload, err := getAllFilePaths(uploadDir)
	assert.NoError(t, err)
	assetManager.MaxNumberOfConcurrentUploads = 4

	ctx, cancelFunc := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancelFunc()

	filesToUploadInterface := Map(filesToUpload, func(v string) interface{} { return interface{}(v) })

	startTime := time.Now()
	results := assetManager.UploadMultipleFiles(ctx, filesToUploadInterface...)
	assert.NotNil(t, results)
	printTestStats(t, results, startTime, filesToUploadInterface)
}

func printTestStats(t *testing.T, results []FileUploadResult, startTime time.Time, filesToUpload []interface{}) {
	for _, upload := range results {
		assert.NotNil(t, upload)
		speed := upload.latency
		if upload.err != nil {
			log.Printf("error: [%s] = %v", upload.file, upload.err)
			log.Printf("upload speed (%.2fs)", float64(speed.Milliseconds())/1000)
			continue
		}
		r := upload.result
		assert.NotNil(t, r)
		if r.Error.Message != "" {
			log.Printf("error: [%s] = %v", upload.file, r.Error.Message)
			log.Printf("upload speed (%.2fs)", float64(speed.Milliseconds())/1000)
			continue
		}
		assert.NotEmpty(t, r.PublicID)
		assert.NotEmpty(t, r.SecureURL)
		log.Printf("upload speed (%.2fs) : ( %v ) ", float64(speed.Milliseconds())/1000, upload.file)
		log.Printf("Secure-url for [%s] = %s", r.PublicID, r.SecureURL)
	}
	endTime := time.Since(startTime)
	log.Printf("Average upload time = (%.2fs) totalTime = (%.2f)", (endTime.Seconds())/float64(len(filesToUpload)), endTime.Seconds())
}

func newTestAssetManager(t *testing.T) *AssetUploadManager {
	clientSecret := os.Getenv(clientSecretEnv)
	clientKey := os.Getenv(clientKeyEnv)
	clientName := os.Getenv(clientNameEnv)
	if clientSecret == "" || clientKey == "" || clientName == "" {
		t.Skip("cloudinary credentials not set in environment")
	}

	am, err := NewAssetUploadManager(clientName, clientKey, clientSecret)
	require.NoError(t, err)
	return am
}

func newTestAssetUploadManagerLarge(t *testing.T) *AssetUploadManager {
	return newTestAssetManager(t)
}

func testUploadDir(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(dir, "upload")
}

func testUploadPath(t *testing.T, filename string) string {
	return filepath.Join(testUploadDir(t), filename)
}

func getAllFilePaths(dir string) ([]string, error) {
	var filePaths []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	return filePaths, err
}
