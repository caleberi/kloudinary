![badge](https://github.com/caleberi/kloudinary/actions/workflows/go.yml/badge.svg)
# Kloudinary

Kloudinary is a Go wrapper around the official Cloudinary SDK. It streamlines uploads, supports concurrent batch uploads, and provides helper methods for asset destruction and image transformations.

## Features

- Simplifies file uploads to Cloudinary.
- Supports multiple file uploads in one go.
- Easily destroy assets by public ID.
- Transform images with minimal code.
- Configurable asset upload settings.

## Installation

To install Kloudinary, run:

```sh
go get github.com/caleberi/kloudinary
```

## Usage

### Initialization

First, you need to initialize the AssetUploadManager with your Cloudinary credentials:

```go
package main

import (
	"github.com/caleberi/kloudinary"
	"log"
)

func main() {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	am, err := kloudinary.NewAssetUploadManager(cloudName, apiKey, apiSecret)
	if err != nil {
		log.Fatalf("Failed to initialize AssetUploadManager: %v", err)
	}

	// Use am to manage your assets
}
```

### Uploading Files

Create a context with a timeout for upload calls:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

1. Upload a Single File
To upload a single file, use the UploadSingleFile method:

```go
result, err := am.UploadSingleFile(ctx, "path/to/your/file.jpg")
if err != nil {
    log.Fatalf("Failed to upload file: %v", err)
}
log.Printf("Upload successful: %v", result)
```

You can also upload from an `io.Reader` (for example, a file handle or a bytes buffer):

```go
file, err := os.Open("path/to/your/file.jpg")
if err != nil {
	log.Fatal(err)
}
defer file.Close()

result, err := am.UploadSingleFile(ctx, file)
if err != nil {
	log.Fatalf("Failed to upload file: %v", err)
}
```

2. Upload Multiple Files
To upload multiple files at once, use the UploadMultipleFiles method:

```go
results := am.UploadMultipleFiles(ctx, "path/to/file1.jpg", "path/to/file2.png")
for _, res := range results {
    log.Printf("Upload result: %v", res)
}
```

3. Destroying Assets
To destroy an asset by its public ID, use the DestroyAsset method:

```go
destroyResult, err := am.DestroyAsset(ctx, "public_id_of_asset", "")
if err != nil {
    log.Fatalf("Failed to destroy asset: %v", err)
}
log.Printf("Destroy result: %v", destroyResult)
```

4. Transforming Images
To transform an image, use the TransformImage method:

```go
url, err := am.TransformImage(ctx, "public_id_of_image", "transformation_string")
if err != nil {
    log.Fatalf("Failed to transform image: %v", err)
}
log.Printf("Transformed image URL: %s", url)
```


## Configuration

The `AssetUploadManager` can be configured with various settings:
```go
am.FileTypeSupported = []string{"jpg", "png", "gif"}
am.MaxAssetSize = 10 * 1024 * 1024 // 10 MB
am.MaxUploadTimeout = 30 * time.Second
am.MaxNumberOfConcurrentUploads = 5
```

Notes:

- `MaxAssetSize` is in bytes.
- `MaxNumberOfConcurrentUploads` controls the number of parallel uploads used by `UploadMultipleFiles`.
- File extension detection is case-insensitive.

## Environment Variables

Recommended variables for initialization:

- `CLOUDINARY_CLOUD_NAME`
- `CLOUDINARY_API_KEY`
- `CLOUDINARY_API_SECRET`

## License
This project is licensed under the MIT License. See the LICENSE file for details.

## Contributing
Feel free to open issues or submit pull requests with improvements. Contributions are always welcome!

## Acknowledgments
 
- The official Cloudinary [SDK](https://github.com/cloudinary/cloudinary-go)