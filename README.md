# titan-storage
Titan Storage is an advanced cloud storage application, integrating a visual interface with efficient functionality. Through our SDK, both developers and enterprises can easily integrate and utilize its features.

## Test example
The test example implements file uploading, listing files that have been uploaded, fetching file, fetching file sharing links, deleting file


### 1 Register from https://storage.titannet.io, and create API Key
![Alt text](doc/c52301810bb6b88e31a73a9d257574b.png)

### 2 Build example
    git clone github.com/Filecoin-Titan/titan-storage-sdk.git
    cd /titan-storage-sdk/example
    go build


### 3 Setting environment variable
	export API_KEY=YOUR-API-KEY
	export TITAN_URL=https://locator.titannet.io:5000/rpc/v0

### 4 run test
##### 4.1 Upload file
	./example upload /path/to/file
##### 4.2 List file
	./example list
##### 4.3 Get file
	./example get --cid=your-file-cid --out=/path/to/save/file
##### 4.4 Get file url (in order to share file)
	./example url your-file-cid
##### 4.5 Delete file
	./example delete your-file-cid

## Using api in code

###  Installation
To use the titan storage sdk, you'll first need to install Go and set up a Go development environment. Once you have Go installed and configured, you can install the titan storage sdk using Go modules:

	go get github.com/Filecoin-Titan/titan-storage-sdk.git

### API 
	UploadFilesWithPath(ctx context.Context, filePath string, progress ProgressFunc) (cid.Cid, error)
	UploadFileWithURL(ctx context.Context, url string, progress ProgressFunc) (string, string, error)
	UploadStream(ctx context.Context, r io.Reader, name string, progress ProgressFunc) (cid.Cid, error)
	ListUserAssets(ctx context.Context, limit, offset int) (*client.ListAssetRecordRsp, error)
	Delete(ctx context.Context, rootCID string) error
	GetURL(ctx context.Context, rootCID string) (string, error)
	GetFileWithCid(ctx context.Context, rootCID string) (io.ReadCloser, error)

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	storage "github.com/Filecoin-Titan/titan-storage-sdk"
)

func main() {
	titanURL := os.Getenv("TITAN_URL")
	apiKey := os.Getenv("API_KEY")

	if len(titanURL) == 0 {
		fmt.Println("please set environment variable TITAN_URL, example: export TITAN_URL=Your_titan_url")
		return
	}

	if len(apiKey) == 0 {
		fmt.Println("please set environment variable API_KEY, example: export API_KEY=Your_API_KEY")
		return
	}

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Please specify the name of the file to be uploaded")
		return
	}
	filePath := args[0]

	storage, close, err := storage.NewStorage(titanURL, apiKey)
	if err != nil {
		fmt.Println("NewSchedulerAPI error ", err.Error())
		return
	}
	defer close()

	//show upload progress
	progress := func(doneSize int64, totalSize int64) {
		fmt.Printf("upload %d, total %d\n", doneSize, totalSize)
		if doneSize == totalSize {
			fmt.Printf("upload success\n")
		}
	}

	root, err := storage.UploadFilesWithPath(context.Background(), filePath, progress)
	if err != nil {
		fmt.Println("UploadFile error ", err.Error())
		return
	}

	fmt.Printf("upload %s success\n", root.String())
}
```