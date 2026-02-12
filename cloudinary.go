package kloudinary

// AssetUploadManager wraps around cloudinary upload package,
// increasing the number of uploads that can be performed concurrent
// and also ensures that files are uploaded to the correct location
// base on the mimetype of the file being uploaded

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
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

	am.MaxAssetSize = 4 * 1024 * 1024 // Max size of 4 MB by default

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
	if file == nil {
		return nil, errors.New("nil file provided")
	}

	switch v := file.(type) {
	case string:
		return am.uploadBasedOnFilePath(ctx, v)
	case []byte:
		return am.upload(ctx, bytes.NewReader(v))
	case io.Reader:
		return am.upload(ctx, v)
	default:
		return nil, errors.New("data type not supported for interface")
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
	if len(files) == 0 {
		return []FileUploadResult{}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	concurrency := am.MaxNumberOfConcurrentUploads
	if concurrency < 1 {
		concurrency = 1
	}

	var (
		ret     = make([]FileUploadResult, 0, len(files))
		uploads = make(chan FileUploadResult, len(files))
		guard   = make(chan struct{}, concurrency)
	)

	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for upload := range uploads {
			ret = append(ret, upload)
		}
	}()

	var workerWg sync.WaitGroup
	for _, file := range files {
		guard <- struct{}{}
		workerWg.Add(1)
		go func(file interface{}) {
			defer workerWg.Done()
			defer func() { <-guard }()

			start := time.Now()
			workerCtx, cancel := context.WithTimeout(ctx, am.MaxUploadTimeout)
			defer cancel()
			result, err := am.UploadSingleFile(workerCtx, file)
			finish := time.Since(start)
			uploads <- FileUploadResult{file: file, err: err, result: result, latency: finish}
		}(file)
	}

	go func() {
		workerWg.Wait()
		close(uploads)
	}()

	collectWg.Wait()
	return ret
}

func (am *AssetUploadManager) upload(ctx context.Context, file io.Reader) (*uploader.UploadResult, error) {
	head := make([]byte, 261)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, err
	}
	head = head[:n]

	kind, _ := filetype.Match(head)
	reader := io.MultiReader(bytes.NewReader(head), file)

	return am.cld.Upload.Upload(
		ctx,
		reader,
		uploader.UploadParams{
			PublicID: uuid.NewString(),
			Folder:   am.getLogicalFolderBasedOnExtension(kind.Extension),
			Metadata: api.Metadata(am.Metadata),
		},
	)
}

func (am *AssetUploadManager) uploadBasedOnFilePath(ctx context.Context, filePath string) (*uploader.UploadResult, error) {
	base := filepath.Base(filePath)

	extension := strings.TrimPrefix(filepath.Ext(filePath), ".")
	extension = strings.ToLower(extension)

	if !am.isFileSupported(extension) {
		return nil, errors.New("invalid MIME type")
	}

	stat, err := os.Lstat(filePath)

	if err != nil {
		return nil, err
	}

	if !stat.Mode().IsRegular() {
		return nil, errors.New("invalid file type")
	}

	if stat.Size() > am.MaxAssetSize {
		return nil, errors.New("max asset size exceeded")
	}

	return am.cld.Upload.Upload(
		ctx, filePath,
		uploader.UploadParams{
			PublicID: base,
			Folder:   am.getLogicalFolderBasedOnExtension(extension),
			Metadata: api.Metadata(am.Metadata),
		},
	)
}
