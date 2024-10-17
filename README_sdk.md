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
|TitanStorage.ListRegions|Retrieve the list of area IDs from the scheduler|
|TitanStorage.CreateFolder|Create directories, including root and subdirectories|
|TitanStorage.ListDirectoryContents|Retrieve a list of all folders and files|
|TitanStorage.RenameFolder|Rename a specific folder|
|TitanStorage.RenameAsset|Rename a specific file|
|TitanStorage.DeleteFolder|Delete a specific folder|
|TitanStorage.DeleteAsset|Delete a specific file|
|TitanStorage.GetUserProfile|Retrieve user-related information|
|TitanStorage.GetltemDetails|Get detailed information about files/folders|
|TitanStorage.CreateSharedLink|Share file/folder data|
|TitanStorage.UploadAsset|Upload files/folders|
|TitanStorage.DownloadAsset|Download files/folders|