## Titan Storage Go SDK
The Titan Storage Go SDK provides functionalities for file uploading, downloading, deleting, renaming, sharing, and creating folders.

The Go SDK consists of TitanStorage.

### Install Go SDK And initialize the go SDK
```bash
go get -u github.com/utopiosphe/titan-storage-sdk
```

Retrieve the apikey and use it to initialize the go SDK

![](doc/access_key.jpg)

```go
package main

import (
    storage "github.com/utopiosphe/titan-storage-sdk"
)

const (
    titanStorageURL = "https://api-test1.container1.titannet.io"
)

var storageCli storage.Storage

func init() {
    var err error

    storageCli,err = storage.NewStorage(&storage.Config{
        TitanURL: titanStorageURL,
        APIKey: os.Getenv("apikey"),
    })
    if err != nil {
        panic(fmt.Errorf("new client of titan storage error:%w",err))
    }
}
```

### Go SDK Method With Client
|Method|Description|
|:-|:-|
|UploadFilesWithPath|upload files from the local file system|
|UploadFileWithURL|uploads a file from the specified URL|
|UploadStream|upload data from an io.Reader stream|
|UploadStreamV2|upload data from an io.Reader stream without making car|
|ListUserAssets|retrieves a list of user assets from the titan storage|
|Delete|remove the data associated with the specified rootCID from the titan storage|
|GetURL|retrieves the URL and asset size associated with the specified rootCID from the titan storage|
|GetFileWithCid|retrieves the file content associated with the specified rootCID from the titan storage|
|CreateGroup|create a group|
|ListGroups|list groups|
|DeleteGroup|delete special group|
