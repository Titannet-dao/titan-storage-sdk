package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	storage "github.com/utopiosphe/titan-storage-sdk"
)

const (
	titanStorageURL = "https://api-test1.container1.titannet.io"
)

var (
	ts  storage.Storage
	ctx context.Context
)

func TestMain(m *testing.M) {
	var err error

	ts, err = storage.Initialize(&storage.Config{
		TitanURL: titanStorageURL,
		APIKey:   os.Getenv("apikey"),
	})
	if err != nil {
		panic(fmt.Errorf("new client of titan storage error:%w", err))
	}

	m.Run()
}

// ExampleStorage_ListRegions Retrieve the list of area IDs from the scheduler
func ExampleStorage_ListRegions() {
	areaIds, err := ts.ListRegions(ctx)
	if err != nil {
		fmt.Printf("ListRegions Error:%v", err)
		return
	}

	fmt.Println(areaIds)
}

// ExampleStorage_CreateFolder Create directories, including root and subdirectories
func ExampleStorage_CreateFolder() {
	err := ts.CreateFolder(ctx, "test", 0)
	if err != nil {
		fmt.Printf("CreateFolder Error:%v", err)
		return
	}
}

// ExampleStorage_ListDirectoryContents  Retrieve a list of all folders and files.
func ExampleStorage_ListDirectoryContents() {
	resp, err := ts.ListDirectoryContents(ctx, 0, 10, 1)
	if err != nil {
		fmt.Printf("ListDirectoryContents Error:%v", err)
		return
	}

	fmt.Println(resp.Total)
}

// ExampleStorage_RenameFolder Rename a specific folder
func ExampleStorage_RenameFolder() {
	err := ts.RenameFolder(ctx, 123, "test")
	if err != nil {
		fmt.Printf("RenameFolder Error:%v", err)
		return
	}
}

// ExampleStorage_RenameAsset Rename a specific file
func ExampleStorage_RenameAsset() {
	err := ts.RenameAsset(ctx, "bafkreib5arnexhnsn6etb4xs7ywm52iey3i7xkxgjxm4dhw5njxmz2dn4i", "456")
	if err != nil {
		fmt.Printf("RenameAsset Error:%v", err)
		return
	}
}

// ExampleStorage_DeleteFolder delete special folder
func ExampleStorage_DeleteFolder() {
	err := ts.DeleteFolder(ctx, 123)
	if err != nil {
		fmt.Printf("DeleteFolder Error:%v", err)
		return
	}
}

// ExampleStorage_DeleteAsset Delete a specific file
func ExampleStorage_DeleteAsset() {
	err := ts.DeleteAsset(ctx, "bafkreib5arnexhnsn6etb4xs7ywm52iey3i7xkxgjxm4dhw5njxmz2dn4i")
	if err != nil {
		fmt.Printf("DeleteFolder Error:%v", err)
		return
	}
}

// ExampleStorage_GetItemDetails Get detailed information about files/folders
func ExampleStorage_GetItemDetails() {
	resp, err := ts.GetItemDetails(ctx, "bafkreib5arnexhnsn6etb4xs7ywm52iey3i7xkxgjxm4dhw5njxmz2dn4i", 123)
	if err != nil {
		fmt.Printf("GetItemDetails Error:%v", err)
		return
	}

	fmt.Println(resp.Total)
}

// ExampleStorage_CreateSharedLink Share file/folder data
func ExampleStorage_CreateSharedLink() {
	shareLink, err := ts.CreateSharedLink(ctx, "bafkreib5arnexhnsn6etb4xs7ywm52iey3i7xkxgjxm4dhw5njxmz2dn4i", 123)
	if err != nil {
		fmt.Printf("GetItemDetails Error:%v", err)
		return
	}

	fmt.Println(shareLink)
}

// ExampleStorage_UploadAsset Upload files/folders
func ExampleStorage_UploadAsset() {
	fp := "test.png"

	body, err := os.ReadFile(fp)
	if err != nil {
		fmt.Printf("read file error:%v", err)
		return
	}

	// Choose either file path or file I/O; if both are provided, filepath takes precedence
	cid, err := ts.UploadAsset(ctx, fp, bytes.NewReader(body), func(doneSize, totalSize int64) {
		fmt.Println(totalSize, doneSize)
	})
	if err != nil {
		fmt.Printf("UploadAsset error:%v", err)
		return
	}

	fmt.Println(cid.String())
}

// ExampleStorage_DownloadAsset Download files/folders
func ExampleStorage_DownloadAsset() {
	body, name, err := ts.DownloadAsset(ctx, "bafkreib5arnexhnsn6etb4xs7ywm52iey3i7xkxgjxm4dhw5njxmz2dn4i")
	if err != nil {
		fmt.Printf("DownloadAsset error:%v", err)
		return
	}

	b, err := io.ReadAll(body)
	if err != nil {
		fmt.Printf("ReadAll from body error:%v", err)
		return
	}
	f, err := os.Create(name)
	if err != nil {
		fmt.Printf("Create file error:%v", err)
		return
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		fmt.Printf("Write file error:%v", err)
	}
}

// ExampleStorage_GetUserProfile Retrieve user-related information
func ExampleStorage_GetUserProfile() {
	userProfile, err := ts.GetUserProfile(ctx)
	if err != nil {
		fmt.Printf("GetUserProfile error:%v", err)
		return
	}
	fmt.Println(userProfile)
}
