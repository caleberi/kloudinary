# Kloudinary: A Golang Cloudinary Wrapper

Kloudinary is a Golang wrapper around the official Cloudinary SDK. It streamlines some redundant parts of using the official SDK and can get you up and running within minutes of installation.

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
	cloudName := "your-cloud-name"
	apiKey := "your-api-key"
	apiSecret := "your-api-secret"

	am, err := kloudinary.NewAssetUploadManager(cloudName, apiKey, apiSecret)
	if err != nil {
		log.Fatalf("Failed to initialize AssetUploadManager: %v", err)
	}

	// Use am to manage your assets
}
```

### Uploading Files

1. Upload a Single File
To upload a single file, use the UploadSingleFile method:

```go
result, err := am.UploadSingleFile(ctx, "path/to/your/file.jpg")
if err != nil {
    log.Fatalf("Failed to upload file: %v", err)
}
log.Printf("Upload successful: %v", result)
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

The AssetUploadManager can be configured with various settings:
```go
am.FileTypeSupported = []string{"jpg", "png", "gif"}
am.MaxAssetSize = 10 * 1024 * 1024 // 10 MB
am.MaxUploadTimeout = 30 * time.Second
am.MaxNumberOfConcurrentUploads = 5
```

## License
This project is licensed under the MIT License. See the LICENSE file for details.

## Contributing
Feel free to open issues or submit pull requests with improvements. Contributions are always welcome!

## Acknowledgments
 
- The official Cloudinary [SDK](https://github.com/cloudinary/cloudinary-go)