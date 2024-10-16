## titan storage sdk 使用说明

You can easily integrate Storacha into your Go apps using titan-storage-sdk, our Go client for the titan storage platform.

In this guide, we'll walk through the following steps:

1. [安装客户端库](#安装客户端)
2. [生成api key](#注册用户)
3. [初始化客户端](#初始化客户端)
4. [文件操作](#文件操作)
    - [x] 获取支持的节点区域
    - [上传文件](#上传文件)
    - [分享文件/获取下载链接](#获取文件)
    - [删除文件](#删除文件)
5. [文件夹操作](#文件夹操作)
    - [创建文件夹](#创建文件夹)
    - [删除文件夹](#删除文件夹)
    - [获取文件夹信息详情](#获取文件夹内容详情)
6. [错误处理](#错误处理)
7. [最佳实践](#最佳实践)

### 安装客户端
你需要[Go](https://go.dev/dl/)版本1.21或更高版本。

```bash
go get -u github.com/utopiosphe/titan-storage-sdk
```

### 注册用户
访问[titan storage官网](https://storage.titannet.io/)并登陆，然后创建密钥并保存好。

![](doc/access_key.jpg)

### 初始化客户端
初始化 titan storage 客户端，后续操作读需要基于客户端去操作。
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
        AreaID: "Asia-HongKong", // 不传的话，默认传到全球节点，首个节点为距离最近的节点
    })
    if err != nil {
        panic(fmt.Errorf("new client of titan storage error:%w",err))
    }
}
```

## 文件操作

### 上传文件
目前上传文件支持文件路径，文件url和文件内容三种方式进行文件上传
+ 通过文件路径进行上传
```go
func uploadWithPath(ctx context.Context,fp string) (string,error) {
    cid,err := storageCli.UploadFilesWithPath(ctx,fp,nil)
    if err != nil {
        return "",fmt.Errorf("upload with path error:%w",err)
    }

    return cid.String(),nil
}
```

+ 通过文件链接进行上传
```go
func uploadWithURL(ctx context.Context,url string) (string,error) {
    cid,_,err := storageCli.UploadFileWithURL(ctx,url,nil)
    if err != nil {
        return "",fmt.Errorf("upload with url error:%w",err)
    }

    return cid,nil
}
```

+ 通过文件内容进行上传
```go
func uploadWithBody(ctx context.Context,body io.Reader,fn string) (string,error) {
    cid,err := storageCli.UploadStreamV2(ctx,body,fn,nil)
    if err != nil {
        return "",fmt.Errorf("upload with body error:%w",err)
    }

    return cid.String(),nil
}
```

### 获取文件
目前提供两种方式获取文件，获取文件下载链接和直接获取文件内容。用户可以根据自己的需求使用不同的方式。
```go
// 获取文件下载链接
func geturlsByCID(ctx context.Context,cid string) (string,[]string,error) {
    res,err := storageCli.GetURL(ctx,cid)
    if err != nil {
        return "",nil,fmt.Errorf("get file's urlds error:%w",err)
    }

    return res.FileName,res.URLs,nil
}

// 获取文件内容
func getbodyByCID(ctx context.Context,cid string) (io.ReadCloser, string, error) {
    return storageCli.GetFileWithCid(ctx,cid)
}
```

### 删除文件
> 注意: 目前仅支持删除根目录的文件
```go
// 删除根目录下的文件
func deleteAsset(ctx context.Context,cid string) error {
    err := storageCli.Delete(ctx,cid)
    if err != nil {
        return fmt.Errorf("delete asset error:%w",err)
    }

    return nil
}
```

## 文件夹操作

### 创建文件夹
```go
func createGroup(ctx context.Context,name string,parentID int) error {
    err := storageCli.CreateGroup(ctx,name,parentID)
    if err != nil {
        return fmt.Errorf("create group error:%w",err)
    }

    return nil
}
```

### 删除文件夹
```go
func deleteGroup(ctx context.Context,gid string) error {
    err := storageCli.DeleteGroup(ctx,gid)
    if err != nil {
        return fmt.Errorf("delete group error:%w",err)
    }

    return nil
}
```

### 获取文件夹内容详情
```go
func createGroup(ctx context.Context,parentID,page,size int) error {
    err := storageCli.ListAssetGroupRsp(ctx,parentID,page,size)
    if err != nil {
        return fmt.Errorf("create group error:%w",err)
    }

    return nil
}
```

## 错误处理

在使用SDK时，请始终检查返回的错误。每个操作都可能返回特定类型的错误，您应该相应地处理这些错误。

## 最佳实践

1. 使用环境变量存储 API 密钥
2. 实现适当的错误处理和日志记录
3. 在生产环境中使用 HTTPS
4. 定期检查 SDK 更新以获取最新功能和安全修复

