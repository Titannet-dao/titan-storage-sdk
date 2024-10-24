# titan-storage
Titan Storage is an advanced cloud storage application, integrating a visual interface with efficient functionality. Through our SDK, both developers and enterprises can easily integrate and utilize its features.


### Register from https://storage.titannet.io, and create tenant API Key Secret pair

## Using api in code

###  Installation
To use the titan storage sdk, you'll first need to install Go and set up a Go development environment. Once you have Go installed and configured, you can install the titan storage sdk using Go modules:

	go get github.com/Titannet-dao/titan-storage-sdk

### API 
	SSOLogin(ctx context.Context, req SubUserInfo) (*SSOLoginRsp, error)
	SyncUser(ctx context.Context, req SubUserInfo) error
	DeleteUser(ctx context.Context, entryUUID string, withAssets bool) error
	RefreshToken(ctx context.Context, token string) (*SSOLoginRsp, error)
    ValidateUploadCallback(ctx context.Context, apiSecret string, r *http.Request) (*AssetUploadNotifyCallback, error)

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	storage "github.com/Titannet-dao/titan-storage-sdk"
)

func main() {
	titanURL := "TITAN_URL"
	tenantKey := "YOUR_TENANT_KEY"
	// ctx := context.TODO()

	tenant, err := NewTenant(titanURL, tenantKey)
	if err != nil {
		fmt.Println("Error:", err)
	}

	tenant.SSOLogin()

	tenant.SyncUser()

	tenant.DeleteUser()

	tenant.RefreshToken()

	tenant.ValidateUploadCallback()
}
```