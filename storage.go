package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/utopiosphe/titan-storage-sdk/client"
	byterange "github.com/utopiosphe/titan-storage-sdk/range"

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

type UploadFileResult struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Cid       string `json:"cid"`
	totalSize int64
}

// ProgressFunc is a function type for reporting progress during file uploads
type ProgressFunc func(doneSize int64, totalSize int64)

// Storage is an interface for interacting with titan storage
type Storage interface {

	// ListRegions Retrieve the list of area IDs from the scheduler
	// or you can use the global value TitanAreas after call Initliaze.
	ListRegions(ctx context.Context) ([]string, error)

	// CreateFolder Create directories, including root and subdirectories
	CreateFolder(ctx context.Context, name string, parentID int) error

	// ListDirectoryContents Retrieve a list of all folders and files.
	// It takes limit and offset parameters for pagination and returns the asset list and any error encountered.
	ListDirectoryContents(ctx context.Context, parent, pageSize, page int) (*client.ListAssetRecordRsp, error)

	// RenameFolder Rename a specific folder
	RenameFolder(ctx context.Context, folderID int64, newName string) error

	// RenameAsset Rename a specific file
	RenameAsset(ctx context.Context, assetCID string, newName string) error

	// DeleteFolder delete special folder
	DeleteFolder(ctx context.Context, folderID int) error

	// DeleteAsset Delete a specific file
	// It returns any error encountered during the deletion process.
	DeleteAsset(ctx context.Context, rootCID string) error

	// GetUserProfile Retrieve user-related information
	GetUserProfile(ctx context.Context) (*client.UserProfile, error)

	// GetItemDetails Get detailed information about files/folders
	GetItemDetails(ctx context.Context, assetCID string, folderID int) (*client.ListAssetRecordRsp, error)

	// CreateSharedLink Share file/folder data
	CreateSharedLink(ctx context.Context, assetCID string, folderID int) (string, error)

	// UploadAsset Upload files/folders
	UploadAsset(ctx context.Context, filePath string, reader io.Reader, progress ProgressFunc) (cid cid.Cid, err error)

	// UploadAssetWithUrl
	UploadAssetWithUrl(ctx context.Context, url string) (cid cid.Cid, fileName string, err error)

	// DownloadAsset Download files/folders
	DownloadAsset(ctx context.Context, assetCID string) (io.ReadCloser, string, error)

	// SetArea set areas before upload or download files
	SetAreas(ctx context.Context, area []string)

	// ------------------------------ Functions blow will be legacy -------------------------------------

	// UploadFilesWithPath uploads files from the local file system to the titan storage.
	// specified by the given filePath. It returns the CID (Content Identifier) and any error encountered.
	// if makeCar is true, it will make car in local, else will make car in server
	UploadFilesWithPath(ctx context.Context, filePath string, progress ProgressFunc, makeCar bool) (cid.Cid, error)
	// UploadFileWithURL uploads a file from the specified URL to the titan storage.
	// It returns the rootCID and the URL of the uploaded file, along with any error encountered.
	UploadFileWithURL(ctx context.Context, url string, progress ProgressFunc) (string, string, error)

	// UploadFileWithURLV2
	UploadFileWithURLV2(ctx context.Context, url string, progress ProgressFunc) (string, string, error)

	// UploadStream uploads data from an io.Reader stream to the titan storage.
	// if name is empty, name will be the cid
	// It returns the CID of the uploaded data and any error encountered.
	UploadStream(ctx context.Context, r io.Reader, name string, progress ProgressFunc) (cid.Cid, error)
	// UploadStreamV2 uploads data from an io.Reader stream without making car to the titan storage.
	UploadStreamV2(ctx context.Context, r io.Reader, name string, progress ProgressFunc) (cid.Cid, error)
	// ListUserAssets retrieves a list of user assets from the titan storage.
	// It takes limit and offset parameters for pagination and returns the asset list and any error encountered.
	ListUserAssets(ctx context.Context, parent, pageSize, page int) (*client.ListAssetRecordRsp, error)
	// Delete removes the data associated with the specified rootCID from the titan storage
	// It returns any error encountered during the deletion process.
	Delete(ctx context.Context, rootCID string) error
	// GetURL retrieves the URL and asset size associated with the specified rootCID from the titan storage.
	// It returns the URL and any error encountered during the retrieval process.
	GetURL(ctx context.Context, rootCID string) (*client.ShareAssetResult, error)
	// GetFileWithCid retrieves the file content associated with the specified rootCID from the titan storage.
	// parallel means multiple concurrent download tasks.
	// It returns an io.ReadCloser for reading the file content and filename and any error encountered during the retrieval process.
	GetFileWithCid(ctx context.Context, rootCID string) (io.ReadCloser, string, error)
	// CreateGroup create a group
	CreateGroup(ctx context.Context, name string, parentID int) error
	// ListGroup list groups
	ListGroups(ctx context.Context, parentID, limit, offset int) (*client.ListAssetGroupRsp, error)
	// DeleteGroup delete special group
	DeleteGroup(ctx context.Context, groupID int) error //perm:user,web,admin

}

// storage is the implementation of the Storage interface
type storage struct {
	webAPI client.Webserver
	// httpClient  *http.Client
	candidateID string
	userID      string
	// Setting the directory for file uploads
	// default is 0, 0 is root directory
	groupID int
	areas   []string
}

type Config struct {
	TitanURL string

	// APIKey and Token set one of the two authentication methods.
	//
	// APIKey is used for long-lived access.
	// Token is created after you have logged in with expire time.
	APIKey string
	Token  string

	// Setting the directory for file uploads
	// default is 0, 0 is root directory
	GroupID     int
	UseFastNode bool
}

var TitanAreas []string

// Initialize creates a new Storage instance
func Initialize(cfg *Config) (Storage, error) {
	if len(cfg.TitanURL) == 0 {
		return nil, fmt.Errorf("TitanURL can not empty")
	}
	if len(cfg.APIKey) == 0 && len(cfg.Token) == 0 {
		return nil, fmt.Errorf("APIKey or Token can not empty")
	}
	// tlsConfig := tls.Config{InsecureSkipVerify: true}
	// httpClient := &http.Client{
	// 	Transport: &http3.RoundTripper{TLSClientConfig: &tlsConfig},
	// }

	// locatorAPI := client.NewLocator(cfg.TitanURL, nil, client.HTTPClientOption(httpClient))
	// schedulerURL, err := locatorAPI.GetSchedulerWithAPIKey(context.Background(), cfg.APIKey)
	// if err != nil {
	// 	return nil, fmt.Errorf("GetSchedulerWithAPIKey %w, api key %s", err, cfg.APIKey)
	// }

	// headers := http.Header{}
	// headers.Add("Authorization", "Bearer "+cfg.APIKey)

	webAPI := client.NewWebserver(cfg.TitanURL, cfg.APIKey, cfg.Token)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	vipInfo, err := webAPI.GetVipInfo(ctx)
	if err != nil {
		return nil, err
	}

	fastNodeID := ""
	if cfg.UseFastNode {
		candidates, err := webAPI.GetCandidateIPs(ctx)
		if err != nil {
			return nil, fmt.Errorf("GetCandidateIPs %w", err)
		}

		fastNodes := getFastNodes(candidates)
		if len(fastNodes) > 0 {
			fastNodeID = fastNodes[0].NodeID
			fmt.Println("use fastest node ", fastNodeID)
		} else {
			fmt.Println("can not get any candidate node")
		}
	}

	TitanAreas, err = webAPI.ListAreaIDs(ctx)
	if err != nil {
		return nil, err
	}

	return &storage{webAPI: webAPI, candidateID: fastNodeID, userID: vipInfo.UserID, groupID: cfg.GroupID}, nil
}

// or you can use the global value TitanAreas after call Initliaze.
func (s *storage) ListRegions(ctx context.Context) ([]string, error) {
	return s.webAPI.ListAreaIDs(ctx)
}

// CreateFolder Create directories, including root and subdirectories
func (s *storage) CreateFolder(ctx context.Context, name string, parent int) error {
	_, err := s.webAPI.CreateGroup(ctx, name, parent)
	return err
}

// ListDirectoryContents Retrieve a list of all folders and files.
// It takes limit and offset parameters for pagination and returns the asset list and any error encountered.
func (s *storage) ListDirectoryContents(ctx context.Context, parent, pageSize, page int) (*client.ListAssetRecordRsp, error) {
	return s.webAPI.ListAssets(ctx, parent, pageSize, page, "", 0)
}

// RenameFolder Rename a specific folder
func (s *storage) RenameFolder(ctx context.Context, folderID int64, newName string) error {
	return s.webAPI.RenameGroup(ctx, s.userID, newName, int(folderID))
}

// RenameAsset Rename a specific file
func (s *storage) RenameAsset(ctx context.Context, assetCID string, newName string) error {
	return s.webAPI.RenameAsset(ctx, assetCID, newName)
}

// DeleteFolder delete special group
func (s *storage) DeleteFolder(ctx context.Context, folderID int) error {
	return s.webAPI.DeleteGroup(ctx, s.userID, folderID)
}

// DeleteAsset Delete removes the data associated with the specified rootCID from the titan storage
// It returns any error encountered during the deletion process.
func (s *storage) DeleteAsset(ctx context.Context, rootCID string) error {
	return s.webAPI.DeleteAsset(ctx, s.userID, rootCID)
}

// GetUserProfile Retrieve user-related information
func (s *storage) GetUserProfile(ctx context.Context) (*client.UserProfile, error) {

	userStorage, err := s.webAPI.GetUserStorage(ctx)
	if err != nil {
		log.Printf("Failed to get user storage, %v", err)
	}

	vipInfo, err := s.webAPI.GetVipInfo(ctx)
	if err != nil {
		log.Printf("Failed to get vip info, %v", err)
	}

	assetCount, err := s.webAPI.GetAssetCount(ctx)
	if err != nil {
		log.Printf("Failed to get asset count, %v", err)
	}

	return &client.UserProfile{
		UserStorage: userStorage,
		Vip:         vipInfo,
		AssetCount:  assetCount,
	}, nil
}

// GetItemDetails Get detailed information about files/folders
func (s *storage) GetItemDetails(ctx context.Context, assetCID string, folderID int) (*client.ListAssetRecordRsp, error) {
	return s.webAPI.ListAssets(ctx, 0, 0, 0, assetCID, folderID)
}

// CreateSharedLink Share file/folder data
func (s *storage) CreateSharedLink(ctx context.Context, assetCID string, folderID int) (string, error) {
	// if folderID > 0 {
	// 	return "", errors.New("not implemented yet")
	// }
	return "", errors.New("not implemented yet")
}

// UploadAsset Upload files/folders
func (s *storage) UploadAsset(ctx context.Context, filePath string, reader io.Reader, progress ProgressFunc) (cid.Cid, error) {
	if filePath != "" {
		fileType, err := getFileType(filePath)
		if err != nil {
			return cid.Cid{}, err
		}

		if fileType == string(FileTypeFolder) {
			return s.uploadFilesWithPathAndMakeCar(ctx, filePath, progress)
		}

		if fileType == string(FileTypeFile) {
			return s.UploadFilesWithPath(ctx, filePath, progress, false)
		}
	}

	if reader != nil {
		return s.UploadStreamV2(ctx, reader, "", progress)
	}

	return cid.Cid{}, errors.New("FilePath or Reader must be non empty")
}

// UploadAssetWithUrl
func (s *storage) UploadAssetWithUrl(ctx context.Context, url string) (cid.Cid, string, error) {
	return cid.Cid{}, "", errors.New("not implemented yet")
}

// DownloadAsset Download files/folders
func (s *storage) DownloadAsset(ctx context.Context, assetCID string) (io.ReadCloser, string, error) {
	res, err := s.GetURL(ctx, assetCID)
	if err != nil {
		return nil, "", err
	}

	start := time.Now()

	r := byterange.New(1 << 20)

	reader, size, err := r.GetFile(ctx, res)

	report := &client.AssetTransferReq{
		CostMs:       int64(time.Since(start).Milliseconds()),
		TotalSize:    size,
		TransferType: client.AssetTransferTypeDownload,
		Cid:          assetCID,
		State:        client.AssetTransferStateFailed,
		TraceID:      res.TraceID,
	}

	if err == nil {
		report.State = client.AssetTransferStateSuccess
	}

	go func() {
		if err := s.webAPI.AssetTransferReport(context.Background(), *report); err != nil {
			log.Printf("failed to send transfer report, %s", err.Error())
		}
	}()

	return reader, res.FileName, err
}

func joinNodeID(str string, nodeID string) string {
	if str == "" {
		return nodeID
	}

	return fmt.Sprintf("%s,%s", str, nodeID)
}

func getNodeIdFromCandidateAddr(addr string) string {
	u, err := url.Parse(addr)
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`([a-f0-9\-]+)\.`)
	matches := re.FindStringSubmatch(u.Host)

	if len(matches) > 1 {
		return matches[1]
	}
	return ""
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

// errAssetNotExist returns an error indicating that the asset does not exist
func errAssetNotExist(cid string) error {
	return fmt.Errorf("ShareAssets err:asset %s not exist", cid)
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
	if len(rscd) > 0 {
		re := regexp.MustCompile(`filename="([^"]+)"`)
		matches := re.FindStringSubmatch(rscd)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	// vs := strings.Split(rscd, ";")
	// if len(vs) < 1 {
	// 	return "", fmt.Errorf("can not find filename")
	// }

	// filename = vs[1]
	// filename = strings.TrimSpace(filename)
	// filename = strings.TrimPrefix(filename, "filename=")

	return path.Base(u.Path), nil
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

func (s *storage) SetAreas(ctx context.Context, areas []string) {
	s.areas = areas
}
