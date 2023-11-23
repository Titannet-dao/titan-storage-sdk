package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Filecoin-Titan/titan-storage-sdk/client"
	"github.com/Filecoin-Titan/titan-storage-sdk/memfile"
	"github.com/ipfs/go-cid"
)

// FileType represents the type of file or folder
type FileType string

const (
	FileTypeFile   FileType = "file"
	FileTypeFolder FileType = "folder"
	timeout                 = 30 * time.Second
	titanHostName           = ".asset.titannet.io"
)

// storageClose is a function type for closing the storage
type storageClose func()

// ProgressFunc is a function type for reporting progress during file uploads
type ProgressFunc func(doneSize int64, totalSize int64)

// Storage is an interface for interacting with titan storage
type Storage interface {
	// UploadFilesWithPath uploads files from the local file system to the titan storage.
	// specified by the given filePath. It returns the CID (Content Identifier) and any error encountered.
	UploadFilesWithPath(ctx context.Context, filePath string, progress ProgressFunc) (cid.Cid, error)
	// UploadFileWithURL uploads a file from the specified URL to the titan storage.
	// It returns the rootCID and the URL of the uploaded file, along with any error encountered.
	UploadFileWithURL(ctx context.Context, url string, progress ProgressFunc) (string, string, error)
	// UploadStream uploads data from an io.Reader stream to the titan storage.
	// if name is empty, name will be the cid
	// It returns the CID of the uploaded data and any error encountered.
	UploadStream(ctx context.Context, r io.Reader, name string, progress ProgressFunc) (cid.Cid, error)
	// ListUserAssets retrieves a list of user assets from the titan storage.
	// It takes limit and offset parameters for pagination and returns the asset list and any error encountered.
	ListUserAssets(ctx context.Context, limit, offset int) (*client.ListAssetRecordRsp, error)
	// Delete removes the data associated with the specified rootCID from the titan storage
	// It returns any error encountered during the deletion process.
	Delete(ctx context.Context, rootCID string) error
	// GetURL retrieves the URL associated with the specified rootCID from the titan storage.
	// It returns the URL and any error encountered during the retrieval process.
	GetURL(ctx context.Context, rootCID string) (string, error)
	// GetFileWithCid retrieves the file content associated with the specified rootCID from the titan storage.
	// It returns an io.ReadCloser for reading the file content and any error encountered during the retrieval process.
	GetFileWithCid(ctx context.Context, rootCID string) (io.ReadCloser, error)
}

// storage is the implementation of the Storage interface
type storage struct {
	schedulerAPI client.Scheduler
	httpClient   *http.Client
	candidateID  string
}

// NewStorage creates a new Storage instance
func NewStorage(titanURL, apiKey string) (Storage, storageClose, error) {
	udpPacketConn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, nil, fmt.Errorf("ListenPacket %w", err)
	}

	// use http3 client
	httpClient, err := client.NewHTTP3Client(udpPacketConn, true, "")
	if err != nil {
		return nil, nil, fmt.Errorf("NewHTTP3Client %w", err)
	}

	locatorAPI := client.NewLocator(titanURL, nil, client.HTTPClientOption(httpClient))
	schedulerURL, err := locatorAPI.GetSchedulerWithAPIKey(context.Background(), apiKey)
	if err != nil {
		return nil, nil, fmt.Errorf("GetSchedulerWithAPIKey %w", err)
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+apiKey)

	schedulerAPI := client.NewScheduler(schedulerURL, headers, client.HTTPClientOption(httpClient))
	close := func() {
		udpPacketConn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	candidates, err := schedulerAPI.GetCandidateIPs(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("GetCandidateIPs %w", err)
	}

	fastNodes := getFastNodes(candidates)
	if len(fastNodes) == 0 {
		return nil, nil, fmt.Errorf("can not get any candidate node")
	}

	return &storage{schedulerAPI: schedulerAPI, httpClient: httpClient, candidateID: fastNodes[0].NodeID}, close, nil
}

// UploadFilesWithPath uploads files from the specified path
func (s *storage) UploadFilesWithPath(ctx context.Context, filePath string, progress ProgressFunc) (cid.Cid, error) {
	// delete template file if exist
	fileName := filepath.Base(filePath)
	tempFile := path.Join(os.TempDir(), fileName)
	if _, err := os.Stat(tempFile); err == nil {
		os.Remove(tempFile)
	}

	root, err := createCar(filePath, tempFile)
	if err != nil {
		return cid.Cid{}, err
	}

	carFile, err := os.Open(tempFile)
	if err != nil {
		return cid.Cid{}, err
	}

	defer func() {
		if err = carFile.Close(); err != nil {
			fmt.Println("close car file error ", err.Error())
		}

		if err = os.Remove(tempFile); err != nil {
			fmt.Println("delete temporary car file error ", err.Error())
		}
	}()

	fileInfo, err := carFile.Stat()
	if err != nil {
		return cid.Cid{}, err
	}

	fileType, err := getFileType(filePath)
	if err != nil {
		return cid.Cid{}, err
	}

	assetProperty := &client.AssetProperty{
		AssetCID:  root.String(),
		AssetName: fileName,
		AssetSize: fileInfo.Size(),
		AssetType: fileType,
		NodeID:    s.candidateID,
	}
	rsp, err := s.schedulerAPI.CreateUserAsset(ctx, assetProperty)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("CreateUserAsset error %w", err)
	}

	if rsp.AlreadyExists {
		return cid.Cid{}, fmt.Errorf("asset %s already exist", root.String())
	}

	err = s.uploadFileWithForm(ctx, carFile, fileName, rsp.UploadURL, rsp.Token, progress)
	if err != nil {
		if delErr := s.schedulerAPI.DeleteUserAsset(ctx, root.String()); delErr != nil {
			return cid.Cid{}, fmt.Errorf("uploadFileWithForm failed %s, delete error %s", err.Error(), delErr.Error())
		}
		return cid.Cid{}, fmt.Errorf("uploadFileWithForm error %s, delete it from titan", err.Error())
	}

	return root, nil
}

// uploadFileWithForm uploads a file using a multipart form
func (s *storage) uploadFileWithForm(ctx context.Context, r io.Reader, name, uploadURL, token string, progress ProgressFunc) error {
	// Create a new multipart form body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a new form field for the file
	fileField, err := writer.CreateFormFile("file", name)
	if err != nil {
		return err
	}

	// Copy the file data to the form field
	_, err = io.Copy(fileField, r)
	if err != nil {
		return err
	}

	// Close the multipart form
	err = writer.Close()
	if err != nil {
		return err
	}

	totalSize := body.Len()
	dongSize := int64(0)
	pr := &ProgressReader{body, func(r int64) {
		if r > 0 {
			dongSize += r
			if progress != nil {
				progress(dongSize, int64(totalSize))
			}
		}
	}}

	// Create a new HTTP request with the form data
	request, err := http.NewRequest("POST", uploadURL, pr)
	if err != nil {
		return fmt.Errorf("new request error %s", err.Error())
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	request = request.WithContext(ctx)

	// Create an HTTP client and send the request
	client := http.DefaultClient
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("do error %s", err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("http StatusCode %d", response.StatusCode)
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	type Result struct {
		Code int    `json:"code"`
		Err  int    `json:"err"`
		Msg  string `json:"msg"`
	}

	var ret Result
	if err := json.Unmarshal(b, &ret); err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(ret.Msg)
	}
	return nil
}

// getFileType returns the type of the file (file or folder)
func getFileType(filePath string) (string, error) {
	fileType := FileTypeFile
	if fileInfo, err := os.Stat(filePath); err != nil {
		return "", err
	} else if fileInfo.IsDir() {
		fileType = FileTypeFolder
	}

	return string(fileType), nil
}

// Delete deletes the specified asset by rootCID
func (s *storage) Delete(ctx context.Context, rootCID string) error {
	return s.schedulerAPI.DeleteUserAsset(ctx, rootCID)
}

// UploadStream uploads a stream of data
func (s *storage) UploadStream(ctx context.Context, r io.Reader, name string, progress ProgressFunc) (cid.Cid, error) {
	memFile := memfile.New([]byte{})
	root, err := createCarStream(r, memFile)
	if err != nil {
		return cid.Cid{}, err
	}
	memFile.Seek(0, 0)

	if len(name) == 0 {
		name = root.String()
	}

	assetProperty := &client.AssetProperty{AssetCID: root.String(), AssetName: name, AssetSize: int64(len(memFile.Bytes())), AssetType: string(FileTypeFile)}
	rsp, err := s.schedulerAPI.CreateUserAsset(ctx, assetProperty)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("CreateUserAsset error %w", err)
	}

	if rsp.AlreadyExists {
		return cid.Cid{}, fmt.Errorf("asset %s already exist", root.String())
	}

	err = s.uploadFileWithForm(ctx, memFile, root.String(), rsp.UploadURL, rsp.Token, progress)
	if err != nil {
		if delErr := s.schedulerAPI.DeleteUserAsset(ctx, root.String()); delErr != nil {
			return cid.Cid{}, fmt.Errorf("uploadFileWithForm failed %s, delete error %s", err.Error(), delErr.Error())
		}
		return cid.Cid{}, fmt.Errorf("uploadFileWithForm error %s, delete it from titan", err.Error())
	}

	return root, nil
}

// GetFileWithCid gets a single file by rootCID
func (s *storage) GetFileWithCid(ctx context.Context, rootCID string) (io.ReadCloser, error) {
	url, err := s.GetURL(ctx, rootCID)
	if err != nil {
		return nil, err
	}

	// Create a new HTTP request with the form data
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request error %s", err.Error())
	}
	request = request.WithContext(ctx)

	// Create an HTTP client and send the request
	client := http.DefaultClient
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("do error %s", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// errAssetNotExist returns an error indicating that the asset does not exist
func errAssetNotExist(cid string) error {
	return fmt.Errorf("ShareAssets err:asset %s not exist", cid)
}

// GetURL gets the URL of the file
func (s *storage) GetURL(ctx context.Context, rootCID string) (string, error) {
	// 100 ms
	var interval = 100
	var startTime = time.Now()
	var timeout = time.Minute
	for {
		rets, err := s.schedulerAPI.ShareUserAssets(ctx, []string{rootCID})
		if err != nil {
			if err.Error() != errAssetNotExist(rootCID).Error() {
				return "", fmt.Errorf("ShareUserAssets %w", err)
			}
		}

		if len(rets) != 0 {
			url := rets[rootCID]
			url = replaceNodeIDToCID(url, rootCID)
			return url, nil
		}

		if time.Since(startTime) > timeout {
			return "", fmt.Errorf("time out of %ds, can not find asset exist", timeout/time.Second)
		}

		time.Sleep(time.Millisecond * time.Duration(interval))
	}
}

// UploadFileWithURL uploads a file from the specified URL
func (s *storage) UploadFileWithURL(ctx context.Context, url string, progress ProgressFunc) (string, string, error) {
	rsp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("http StatusCode %d", rsp.StatusCode)
	}

	filename, err := getFileNameFromURL(url)
	if err != nil {
		fmt.Println("getFileNameFromURL ", err.Error())
	}

	rootCid, err := s.UploadStream(ctx, rsp.Body, filename, progress)
	if err != nil {
		return "", "", err
	}

	newURL, err := s.GetURL(ctx, rootCid.String())
	if err != nil {
		return "", "", err
	}

	return rootCid.String(), newURL, nil
}

// getFastNodes returns a list of fast nodes from the given candidates
func getFastNodes(candidates []*client.CandidateIPInfo) []*client.CandidateIPInfo {
	if len(candidates) == 0 {
		return make([]*client.CandidateIPInfo, 0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	lock := &sync.Mutex{}
	fastCandidates := make([]*client.CandidateIPInfo, 0)

	var acquireFastNode = func(ctx context.Context, wg *sync.WaitGroup, candidate *client.CandidateIPInfo) error {
		defer wg.Done()

		request, err := http.NewRequest("GET", candidate.ExternalURL, nil)
		if err != nil {
			return err
		}
		request = request.WithContext(ctx)

		// Create an HTTP client and send the request
		client := http.DefaultClient
		_, err = client.Do(request)
		if err != nil {
			return fmt.Errorf("do error %s", err.Error())
		}
		cancel()

		lock.Lock()
		fastCandidates = append(fastCandidates, candidate)
		lock.Unlock()
		return nil
	}

	for _, candidate := range candidates {
		wg.Add(1)

		go acquireFastNode(ctx, wg, candidate)

	}
	wg.Wait()

	return fastCandidates
}

// getFileNameFromURL extracts the filename from the URL
func getFileNameFromURL(rawURL string) (string, error) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", err
	}

	filename := u.Query().Get("filename")
	if len(filename) > 0 {
		return filename, nil
	}

	// special for chatgpt
	rscd := u.Query().Get("rscd")
	vs := strings.Split(rscd, ";")
	if len(vs) < 1 {
		return "", fmt.Errorf("can not find filename")
	}

	filename = vs[1]
	filename = strings.TrimSpace(filename)
	filename = strings.TrimPrefix(filename, "filename=")

	return filename, nil
}

func replaceNodeIDToCID(urlString string, cid string) string {
	if strings.Contains(urlString, titanHostName) {
		u, err := url.ParseRequestURI(urlString)
		if err != nil {
			fmt.Println("ParseRequestURI error", err.Error())
			return urlString
		}

		hostName := u.Hostname()
		nodeID := strings.TrimSuffix(hostName, titanHostName)
		return strings.Replace(urlString, nodeID, cid, 1)
	}

	return urlString
}

func (s *storage) ListUserAssets(ctx context.Context, limit, offset int) (*client.ListAssetRecordRsp, error) {
	return s.schedulerAPI.ListUserAssets(ctx, limit, offset)
}
