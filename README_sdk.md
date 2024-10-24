## Titan Storage Go SDK
The Titan Storage Go SDK provides functionalities for file uploading, downloading, deleting, renaming, sharing, and creating folders.

The Go SDK consists of TitanStorage.

### Install Go SDK And initialize the go SDK
```bash
go get -u github.com/Titannet-dao/titan-storage-sdk
```

Retrieve the apikey and use it to initialize the go SDK

![](doc/access_key.jpg)

```go
package main

import (
    storage "github.com/Titannet-dao/titan-storage-sdk"
)

const (
    titanStorageURL = "https://api-test1.container1.titannet.io"
)

var TitanStorage storage.Storage

func init() {
    var err error

    TitanStorage,err = storage.Initialize(&storage.Config{
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
|[TitanStorage.ListRegions](example/storage_test.go#L32)|Retrieve the list of area IDs from the scheduler|
|[TitanStorage.CreateFolder](example/storage_test.go#L47)|Create directories, including root and subdirectories|
|[TitanStorage.ListDirectoryContents](example/storage_test.go#L56)|Retrieve a list of all folders and files|
|[TitanStorage.RenameFolder](example/storage_test.go#L67)|Rename a specific folder|
|[TitanStorage.RenameAsset](example/storage_test.go#L76)|Rename a specific file|
|[TitanStorage.DeleteFolder](example/storage_test.go#L85)|Delete a specific folder|
|[TitanStorage.DeleteAsset](example/storage_test.go#L94)|Delete a specific file|
|[TitanStorage.GetUserProfile](example/storage_test.go#174)|Retrieve user-related information|
|[TitanStorage.GetltemDetails](example/storage_test.go#L103)|Get detailed information about files/folders|
|[TitanStorage.CreateSharedLink](example/storage_test.go#L114)|Share file/folder data|
|[TitanStorage.UploadAsset](example/storage_test.go#L126)|Upload files/folders|
|[TitanStorage.DownloadAsset](example/storage_test.go#L149)|Download files/folders|
