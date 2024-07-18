package kloudinary

// AssetUploadManager wraps around cloudinary upload package,
// increasing the number of uploads that can be performed concurrent
// and also ensures that files are uploaded to the correct location
// base on the mimetype of the file being uploaded

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api"
	"github.com/cloudinary/cloudinary-go/api/uploader"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"golang.org/x/exp/slices"
)

var (
	imageExtensions    = []string{"jpg", "jpeg", "png", "gif", "svg", "ico"}
	videoExtensions    = []string{"mp4", "webm", "ogg", "ogv", "avi", "mov"}
	audioExtensions    = []string{"mp3", "wav", "ogg", "oga", "m4a"}
	documentExtensions = []string{
		"pdf", "doc", "docx", "xls", "xlsx", "odf", "ppt", "pptx",
		"txt", "rtf", "csv", "odt", "html", "htm", "xml", "json", "yaml", "js", "yml",
		"md", "markdown", "csv", "tsv", "css", "less", "scss", "sass", "styl", "stylus",
	}
	extensionMap = map[string]string{
		"mp3":      "audio",
		"wav":      "audio",
		"ogg":      "audio",
		"oga":      "audio",
		"m4a":      "audio",
		"jpg":      "images",
		"jpeg":     "images",
		"png":      "images",
		"gif":      "images",
		"svg":      "images",
		"ico":      "images",
		"mp4":      "videos",
		"webm":     "videos",
		"ogv":      "videos",
		"avi":      "videos",
		"mov":      "videos",
		"pdf":      "documents",
		"doc":      "documents",
		"docx":     "documents",
		"xls":      "documents",
		"xlsx":     "documents",
		"odf":      "documents",
		"ppt":      "documents",
		"pptx":     "documents",
		"txt":      "documents",
		"rtf":      "documents",
		"csv":      "documents",
		"odt":      "documents",
		"html":     "documents",
		"htm":      "documents",
		"xml":      "documents",
		"json":     "documents",
		"yaml":     "documents",
		"js":       "documents",
		"yml":      "documents",
		"md":       "documents",
		"markdown": "documents",
		"tsv":      "documents",
		"css":      "documents",
		"less":     "documents",
		"scss":     "documents",
		"sass":     "documents",
		"styl":     "documents",
		"stylus":   "documents",
	}
)

type Meta map[string]interface{}

func (m Meta) has(key string) bool {
	_, ok := m[key]
	return ok
}

func (m Meta) Add(key string, value interface{}) {
	k := strings.ToLower(key)
	m[k] = value
}

func (m Meta) Remove(key string) {
	k := strings.ToLower(key)
	if m.has(k) {
		delete(m, k)
	}
}

type FileUploadResult struct {
	file    interface{}
	err     error
	result  *uploader.UploadResult
	latency time.Duration
}

// AssetUploadManager wraps around cloudinary upload package,
// increasing the number of uploads that can be performed concurrent
// and also ensures that files are uploaded to the correct location
// base on the mimetype of the file being uploaded
type AssetUploadManager struct {
	FileTypeSupported            []string      // supported file types
	MaxAssetSize                 int64         // maximum size for an asset
	MaxUploadTimeout             time.Duration // maximum upload timeout for assets to be uploaded
	MaxNumberOfConcurrentUploads int64         // number of concurrent uploads
	Metadata                     Meta          // metadata about asset upload management for this instance

	cld *cloudinary.Cloudinary
}

// NewAssetUploadManager New creates a new asset uploader which will upload files and other supported
// assets to cloudinary server. Inorder, for this to work, you need to configure
// AssetUploadManager with the following configuration `cloudName`, `apikey`,and `apiSecret`
// for more information visit: https://cloudinary.com/documentation/
func NewAssetUploadManager(cloudName string, apiKey string, apiSecret string) (*AssetUploadManager, error) {

	//  create a new asset upload manager instance
	//  Note:  go can decide whether it is necessary to put it
	//  on the heap or stack
	am := new(AssetUploadManager)

	am.FileTypeSupported = make([]string, 1)

	// set up default file support for uploaded files
	am.FileTypeSupported = append(am.FileTypeSupported, imageExtensions...)
	am.FileTypeSupported = append(am.FileTypeSupported, audioExtensions...)
	am.FileTypeSupported = append(am.FileTypeSupported, videoExtensions...)
	am.FileTypeSupported = append(am.FileTypeSupported, documentExtensions...)

	am.MaxAssetSize = 1024 * 4 // Max size of 4 mb by default

	am.Metadata = Meta{}                  // metadata to store on each asset manger configuration
	am.MaxNumberOfConcurrentUploads = 1   // default to single
	am.MaxUploadTimeout = 1 * time.Minute // maximum upload timeout for push requests to cloudinary server

	cld, err := cloudinary.NewFromParams(
		cloudName,
		apiKey,
		apiSecret,
	)

	if err != nil {
		return nil, err
	}

	am.cld = cld
	return am, nil
}

// isFileSupported returns true if the asset to be uploaded is supported by the
// asset upload manager
func (am *AssetUploadManager) isFileSupported(extension string) bool {
	return len(am.FileTypeSupported) == 0 || slices.Contains(am.FileTypeSupported, extension)
}

// setLogicalFolderBasedOnExtension configures the upload directory folder based on the
// given extension of the provided asset to be uploaded
func (am *AssetUploadManager) getLogicalFolderBasedOnExtension(extension string) string {
	if folder, ok := extensionMap[extension]; ok {
		return folder
	} else {
		return "others"
	}
}

// UploadSingleFile is used to upload a single file to the server
// the file can either be a byte slice or a string
func (am *AssetUploadManager) UploadSingleFile(ctx context.Context, file interface{}) (*uploader.UploadResult, error) {

	value := reflect.TypeOf(file)

	switch value.Kind() {
	case reflect.String:
		return am.uploadBasedOnFilePath(ctx, file)
	default:
		// the interface type provide can be
		// asserted the file type from the interface
		// since it is an io.Reader type. we can check the first 261-byte header
		// and assert the mimetype of the file
		file, ok := file.(io.Reader)
		if !ok {
			return nil, errors.New("data type not supported for interface")
		}
		return am.upload(ctx, file)
	}
}

// TransformImage is used to transform image property of a single file on a cloudinary server
func (am *AssetUploadManager) TransformImage(ctx context.Context, publicId string, transformation string) (string, error) {
	img, err := am.cld.Image(publicId)

	if err != nil {
		return "", err
	}

	img.Transformation = transformation

	// Generate the delivery URL
	url, err := img.String()

	if err != nil {
		return "", err
	}

	return url, nil
}

func (am *AssetUploadManager) DestroyAsset(ctx context.Context, publicId string, transformation string) (*uploader.DestroyResult, error) {
	return am.cld.Upload.Destroy(
		ctx,
		uploader.DestroyParams{
			PublicID: publicId,
		},
	)

}

func (am *AssetUploadManager) UploadMultipleFiles(ctx context.Context, files ...interface{}) []FileUploadResult {

	var (
		ret       = make([]FileUploadResult, 0)
		uploads   = make(chan FileUploadResult, len(files)) //  upload retrieval channel
		semaphore = make(chan struct{}, am.MaxNumberOfConcurrentUploads)
		wg        = sync.WaitGroup{}
	)

	wg.Add(1)
	go func(wg *sync.WaitGroup, ret *[]FileUploadResult, uploads chan FileUploadResult) {
		defer wg.Done()
		for upload := range uploads {
			*ret = append(*ret, upload)
		}
	}(&wg, &ret, uploads)

	for id, file := range files {
		semaphore <- struct{}{}
		go func(id int, file interface{}) {
			defer func(id int, semaphore chan struct{}) {
				<-semaphore
				if id == len(files)-1 {
					close(uploads)
					close(semaphore)
				}
			}(id, semaphore)

			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), am.MaxUploadTimeout)
			defer cancel()
			result, err := am.UploadSingleFile(ctx, file)
			finish := time.Since(start)
			uploads <- FileUploadResult{file: file, err: err, result: result, latency: finish}

		}(id, file)
	}

	wg.Wait()
	return ret
}

func (am *AssetUploadManager) upload(ctx context.Context, file io.Reader) (*uploader.UploadResult, error) {
	head := make([]byte, 261)
	_, err := file.Read(head)
	if err != nil {
		return nil, err
	}

	kind, _ := filetype.Match(head)

	return am.cld.Upload.Upload(
		ctx,
		file,
		uploader.UploadParams{
			PublicID: uuid.NewString(),
			Folder:   am.getLogicalFolderBasedOnExtension(kind.Extension),
			Metadata: api.Metadata(am.Metadata),
		},
	)
}

func (am *AssetUploadManager) uploadBasedOnFilePath(ctx context.Context, file interface{}) (*uploader.UploadResult, error) {
	f := file.(string)

	base := filepath.Base(f)

	extension := strings.TrimPrefix(filepath.Ext(f), ".")

	if !am.isFileSupported(extension) {
		return nil, errors.New("invalid MIME type")
	}

	stat, err := os.Lstat(file.(string))

	if err != nil {
		return nil, err
	}

	// Convert file size from bytes to megabytes
	sizeInMB := float64(stat.Size()) / (1024 * 1024)

	// Check if the file size exceeds the maximum allowed size in megabytes
	if sizeInMB > float64(am.MaxAssetSize) {
		if !(stat.Mode().IsDir() || stat.Mode().IsRegular()) {
			return nil, errors.New("max asset size exceeded")
		}
	}

	f, _ = file.(string)
	return am.cld.Upload.Upload(
		ctx, f,
		uploader.UploadParams{
			PublicID: base,
			Folder:   am.getLogicalFolderBasedOnExtension(extension),
			Metadata: api.Metadata(am.Metadata),
		},
	)
}
