package TitanStorageMobile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	storage "github.com/Titannet-dao/titan-storage-sdk"
)

type ProgressHandler interface {
	OnProgress(doneSize int64, totalSize int64)
}

var storage_api storage.Storage

func StorageInit(titanURL string, apiKey string, groupid int) (err error) {
	storage_api, err = storage.Initialize(&storage.Config{TitanURL: titanURL, APIKey: apiKey, GroupID: groupid})
	return
}

func CreateGroup(name string, parentID int) error {
	return storage_api.CreateGroup(context.Background(), name, parentID)
}

func ListGroups(parentID, limit, offset int) (resp string, err error) {
	originResp, err1 := storage_api.ListGroups(context.Background(), parentID, limit, offset)
	if err1 != nil {
		err = err1
		return
	}
	jsonData, err2 := json.Marshal(originResp)
	if err2 != nil {
		err = err2
		return
	}
	resp = string(jsonData)
	return
}

func DeleteGroup(groupID int) error {
	return storage_api.DeleteGroup(context.Background(), groupID)
}

func ListUserAssets(parent, pageSize, page int) (resp string, err error) {
	originResp, err1 := storage_api.ListUserAssets(context.Background(), parent, pageSize, page)
	if err1 != nil {
		err = err1
		return
	}
	jsonData, err2 := json.Marshal(originResp)
	if err2 != nil {
		err = err2
		return
	}
	resp = string(jsonData)
	return
}

func UploadFilesWithPath(filePath string, handler ProgressHandler) (string, error) {
	progress := func(doneSize int64, totalSize int64) {
		if handler != nil {
			handler.OnProgress(doneSize, totalSize)
		}
	}
	originCid, err := storage_api.UploadFilesWithPath(context.Background(), filePath, progress, true)
	return originCid.String(), err
}

func UploadFileWithURL(url string, handler ProgressHandler) (string, error) {
	progress := func(doneSize int64, totalSize int64) {
		if handler != nil {
			handler.OnProgress(doneSize, totalSize)
		}
	}
	cid, url, err := storage_api.UploadFileWithURL(context.Background(), url, progress)

	return fmt.Sprintf("{\"rootCID\":\"%s\", \"url\":\"%s\"}", cid, url), err
}

func UploadFileWithData(data []byte, name string, handler ProgressHandler) (string, error) {
	progress := func(doneSize int64, totalSize int64) {
		if handler != nil {
			handler.OnProgress(doneSize, totalSize)
		}
	}
	reader := bytes.NewReader(data)
	originCid, err := storage_api.UploadStream(context.Background(), reader, name, progress)

	return originCid.String(), err
}

func GetURL(rootCID string) (string, error) {
	res, err := storage_api.GetURL(context.Background(), rootCID)
	if err != nil {
		return "", err
	}
	return res.URLs[0], nil
}

func GetFileWithCid(rootCID string) (data []byte, err error) {
	readerCloser, _, err1 := storage_api.GetFileWithCid(context.Background(), rootCID)
	if err1 != nil {
		err = err1
		return
	}
	data, err2 := io.ReadAll(readerCloser)
	err = err2
	return
}

func Delete(rootCID string) error {
	return storage_api.Delete(context.Background(), rootCID)
}
