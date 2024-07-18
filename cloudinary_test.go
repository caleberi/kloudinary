package kloudinary

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/exp/rand"

	"github.com/stretchr/testify/assert"
)

var assetManager *AssetUploadManager

var (
	clientSecret = os.Getenv("clientSecret")
	clientKey    = os.Getenv("clientKey")
	clientName   = os.Getenv("clientName")
)

func init() {
	am, err := NewAssetUploadManager(clientName, clientKey, clientSecret)
	if err != nil {
		log.Fatal(err)
	}
	assetManager = am
}

func TestAssetUploadManager_UploadSingleFile(t *testing.T) {
	dir, err := os.Getwd()
	assert.NoError(t, err)

	fileToUpload := path.Clean(path.Join(dir, "./upload/test0.png"))
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	result, err := assetManager.UploadSingleFile(ctx, fileToUpload)

	// Check for any errors
	if err != nil {
		t.Errorf("UploadSingleFile() error = %v", err)
		return
	}

	assert.NotNil(t, result)
	r := *result
	log.Printf("result : ( %v )", r)
	assert.Empty(t, r.Error.Message)
	assert.NotEmpty(t, r.PublicID)
	assert.NotEmpty(t, r.SecureURL)
	log.Printf("Secure-url for [%s] = %s", result.PublicID, result.SecureURL)
}

func TestAssetUploadManager_UploadSingleFile_BinaryData(t *testing.T) {
	dir, err := os.Getwd()
	assert.NoError(t, err)

	fileToUpload := path.Clean(path.Join(dir, "./upload/test5.xml"))
	file, err := os.Open(fileToUpload)
	assert.NoError(t, err)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	result, err := assetManager.UploadSingleFile(ctx, file)

	// Check for any errors
	if err != nil {
		t.Errorf("UploadSingleFile() error = %v", err)
		return
	}

	assert.NotNil(t, result)
	r := *result
	log.Printf("result : ( %v )", r)
	assert.Empty(t, r.Error.Message)
	assert.NotEmpty(t, r.PublicID)
	assert.NotEmpty(t, r.SecureURL)
	log.Printf("Secure-url for [%s] = %s", result.PublicID, result.SecureURL)

}

func TestAssetUploadManager_UploadMultipleFiles(t *testing.T) {
	dir, err := os.Getwd()
	assert.NoError(t, err)

	var filesToUpload []interface{}

	for i := 0; i < 3; i++ {
		filesToUpload = append(
			filesToUpload,
			path.Clean(path.Join(dir, fmt.Sprintf("./upload/test%d.png", i))),
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
	dir, err := os.Getwd()
	assert.NoError(t, err)

	filesToUpload, err := getAllFilePaths(path.Join(dir, "upload"))
	assert.NoError(t, err)

	// since we don't want to use the total number go routine ratio to file
	numberOfRoutine := int64(rand.Intn(10) + 1)

	assetManager.MaxNumberOfConcurrentUploads = numberOfRoutine

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
