# titan-storage-sdk
Titan Storage is an advanced cloud storage application, integrating a visual interface with efficient functionality. Through our SDK, both developers and enterprises can easily integrate and utilize its features.

## Test
### 1 Build
    git clone https://github.com/zscboy/titan-storage-sdk.git
    cd titan-storage-sdk/example
    go build


### 2 Register from https://storage.titannet.io, and create API Key
![Alt text](doc/c52301810bb6b88e31a73a9d257574b.png)

### 3 upload file
    ./example --api-key YOUR-API-KEY --locator-url https://locator.titannet.io:5000/rpc/v0 YOUR-FILE


## Usage

```go
package main

import (
	"context"
	"flag"
	"fmt"

	storage "github.com/Filecoin-Titan/titan-storage-sdk"
)

func main() {
	locatorURL := flag.String("locator-url", "https://locator.titannet.io:5000/rpc/v0", "locator url")
	apiKey := flag.String("api-key", "", "api key")

	// 解析命令行参数
	flag.Parse()

	if len(*locatorURL) == 0 {
		fmt.Println("locator-url can not empty")
		return
	}

	if len(*apiKey) == 0 {
		fmt.Println("api-key can not empty")
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("please input file path")
		return
	}
	filePath := args[0]

	storage, close, err := storage.NewStorage(*locatorURL, *apiKey)
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

	root, err := storage.UploadFile(context.Background(), filePath, progress)
	if err != nil {
		fmt.Println("UploadFile error ", err.Error())
		return
	}

	if err := storage.DeleteFile(context.Background(), root.String()); err != nil {
		fmt.Println("UploadFile error ", err.Error())
		return
	}

	fmt.Printf("delete %s success\n", root.String())
}
```