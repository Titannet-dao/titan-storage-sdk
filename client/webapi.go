package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const isAssetAlreadyExist = 1017

type JWTPayload struct {
	// role base access controller permission
	Allow []string
	ID    string
	// TODO remove NodeID later, any role id replace as ID
	NodeID string
	// Extend is json string
	Extend string
	// The sub permission of user
	AccessControlList []string
}

// AssetProperty represents the properties of an asset.
type AssetProperty struct {
	AssetCID  string
	AssetName string
	AssetSize int64
	AssetType string
	NodeID    string
	GroupID   int
}

type CreateAssetReq struct {
	// UserID string
	AreaID string
	AssetProperty
}

// CreateAssetRsp represents the response when creating an asset.
type Endpoint struct {
	CandidateAddr string
	Token         string
	AlreadyExists bool
}

type CreateAssetRsp struct {
	IsAlreadyExist bool
	Endpoints      []*Endpoint
}

// CandidateIPInfo represents information about a candidate IP.
type CandidateIPInfo struct {
	NodeID      string
	IP          string
	ExternalURL string
}

// ReplicaInfo represents information about a replica.
type ReplicaInfo struct {
	Hash        string
	NodeID      string
	Status      int
	IsCandidate bool
	EndTime     time.Time
	DoneSize    int64
}

// AssetRecord represents information about an asset record.
type AssetRecord struct {
	CID                   string
	Hash                  string
	NeedEdgeReplica       int64
	TotalSize             int64
	TotalBlocks           int64
	Expiration            time.Time
	CreatedTime           time.Time
	EndTime               time.Time
	NeedCandidateReplicas int64
	ServerID              string
	State                 string
	NeedBandwidth         int64

	RetryCount        int64
	ReplenishReplicas int64
	ReplicaInfos      []*ReplicaInfo

	SPCount int64
}

// UserAssetDetail represents detailed information about a user's asset.
type UserAssetDetail struct {
	UserID      string
	Hash        string
	AssetName   string
	AssetType   string
	ShareStatus int64
	Expiration  time.Time
	CreatedTime time.Time
	TotalSize   int64
}

// AssetOverview represents an overview of an asset.
type AssetOverview struct {
	AssetRecord      *AssetRecord
	UserAssetDetail  *UserAssetDetail
	VisitCount       int
	RemainVisitCount int
}

// ListAssetRecordRsp represents the response when listing asset records.
type ListAssetRecordRsp struct {
	Total          int
	AssetOverviews []*AssetOverview
}

// AssetGroup user asset group
type AssetGroup struct {
	ID          int
	UserID      string
	Name        string
	Parent      int
	AssetCount  int
	AssetSize   int64
	CreatedTime time.Time
}

// ListAssetGroupRsp list  asset group records
type ListAssetGroupRsp struct {
	Total       int           `json:"total"`
	AssetGroups []*AssetGroup `json:"list"`
}

// UserAssetSummary user asset and group
type UserAssetSummary struct {
	AssetOverview *AssetOverview
	AssetGroup    *AssetGroup
}

// ListAssetSummaryRsp list asset and group
type ListAssetSummaryRsp struct {
	Total int                 `json:"total"`
	List  []*UserAssetSummary `json:"list"`
}

type UploadInfo struct {
	List          []*NodeUploadInfo
	AlreadyExists bool
	AreaID        string
	Log           string
}

type NodeUploadInfo struct {
	UploadURL string
	Token     string
	NodeID    string
}

type VipInfo struct {
	UserID string `json:"uid"`
	VIP    bool   `json:"vip"`
}

type ShareAssetResult struct {
	AssetCID string   `json:"asset_cid"`
	Redirect bool     `json:"redirect"`
	Size     int64    `json:"size"`
	URLs     []string `json:"url"`
	FileName string
}

type Result struct {
	Code int    `json:"code"`
	Err  int    `json:"err"`
	Msg  string `json:"msg"`
	Data interface{}
}

// Webserver defines the interface for the scheduler.
type Webserver interface {
	// AuthVerify checks whether the specified token is valid and returns the list of permissions associated with it.
	// AuthVerify(ctx context.Context, token string) (*JWTPayload, error)
	// GetVipInfo() (string, error)
	GetVipInfo(ctx context.Context) (*VipInfo, error)
	// LisgAreaIDs list all area id
	LisgAreaIDs(ctx context.Context) ([]string, error)
	// CreateAsset creates an asset with car CID, car name, and car size.
	CreateAsset(ctx context.Context, req *CreateAssetReq) (*CreateAssetRsp, error)
	// DeleteAsset deletes the asset of the user.
	DeleteAsset(ctx context.Context, userID, assetCID string) error
	// ShareAsset shares the assets of the user.
	ShareAsset(ctx context.Context, userID, areaID, assetCID string) (*ShareAssetResult, error)
	// GetCandidateIPs retrieves information about candidate IPs.
	GetCandidateIPs(ctx context.Context) ([]*CandidateIPInfo, error)
	// ListAssets lists the assets of the user.
	ListAssets(ctx context.Context, parent, pageSize, page int) (*ListAssetRecordRsp, error)

	// CreateGroup create Asset group
	CreateGroup(ctx context.Context, name string, parent int) (*AssetGroup, error)
	// ListGroups get groups on parent group
	ListGroups(ctx context.Context, parent, pageSize, page int) (*ListAssetGroupRsp, error)
	// ListAssetSummary list Asset and group
	ListAssetSummary(ctx context.Context, userID string, parent, limit, offset int) (*ListAssetSummaryRsp, error)
	// DeleteGroup delete a group
	DeleteGroup(ctx context.Context, userID string, groupID int) error
	// RenameGroup rename group
	RenameGroup(ctx context.Context, userID, newName string, groupID int) error
	// MoveAssetToGroup move a asset to group
	MoveAssetToGroup(ctx context.Context, userID, cid string, groupID int) error
	// MoveAssetGroup move a asset group
	MoveAssetGroup(ctx context.Context, userID string, groupID, targetGroupID int) error
	// GetAPPKeyPermissions get the permissions of user app key
	GetAPPKeyPermissions(ctx context.Context, userID, keyName string) ([]string, error)
	// GetNodeUploadInfo
	GetNodeUploadInfo(ctx context.Context, userID string, urlMode bool) (*UploadInfo, error)
}

var _ Webserver = (*webserver)(nil)

// NewWebserver creates a new Scheduler instance with the specified URL, headers, and options.
func NewWebserver(url string, apiKey string, area string) Webserver {
	return &webserver{url: url, apiKey: apiKey, client: http.DefaultClient, area: area}
}

type webserver struct {
	// client *Client
	url    string
	apiKey string
	area   string
	client *http.Client
}

func (s *webserver) GetVipInfo(ctx context.Context) (*VipInfo, error) {
	url := fmt.Sprintf("%s/api/v1/storage/get_vip_info", s.url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return nil, fmt.Errorf("status code %d, %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	vipInfo := &VipInfo{}
	err = interfaceToStruct(ret.Data, vipInfo)
	if err != nil {
		return nil, err
	}

	return vipInfo, nil
}

func (s *webserver) LisgAreaIDs(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/storage/get_area_id", s.url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return nil, fmt.Errorf("status code %d, %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	fmt.Println("body ", string(body))
	return nil, nil
}

type webCreateAssetReq struct {
	AssetName string   `json:"asset_name"`
	AssetCID  string   `json:"asset_cid"`
	AreaID    []string `json:"area_id"`
	NodeID    string   `json:"node_id"`
	AssetType string   `json:"asset_type"`
	AssetSize int64    `json:"asset_size"`
	GroupID   int64    `json:"group_id"`
	Encrypted bool     `json:"encrypted"`
}

// CreateUserAsset creates a new user asset.
func (s *webserver) CreateAsset(ctx context.Context, caReq *CreateAssetReq) (*CreateAssetRsp, error) {
	uploadUrl := fmt.Sprintf("%s/api/v1/storage/create_asset", s.url)
	// uploadUrl := fmt.Sprintf("%s/api/v1/storage/create_asset?area_id=%s&asset_name=%s&asset_cid=%s&node_id=%s&asset_type=%s&asset_size=%d&group_id=%d",
	// 	s.url, caReq.AreaID, neturl.QueryEscape(caReq.AssetName), caReq.AssetCID, caReq.NodeID, caReq.AssetType, caReq.AssetSize, caReq.GroupID)

	postData := webCreateAssetReq{
		AssetName: caReq.AssetName,
		AssetCID:  caReq.AssetCID,
		AreaID:    []string{caReq.AreaID},
		NodeID:    caReq.NodeID,
		AssetType: caReq.AssetType,
		AssetSize: caReq.AssetSize,
		GroupID:   int64(caReq.GroupID),
	}

	jsonBytes, err := json.Marshal(postData)
	if err != nil {
		return nil, err
	}
	fmt.Println("url: ", uploadUrl, "data: ", string(jsonBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", uploadUrl, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return nil, fmt.Errorf("status code %d, %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		if ret.Err == isAssetAlreadyExist {
			return &CreateAssetRsp{IsAlreadyExist: true, Endpoints: nil}, nil
		}
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	endpoints := make([]*Endpoint, 0)
	err = interfaceToStruct(ret.Data, &endpoints)
	if err != nil {
		return nil, err
	}
	// fmt.Println("body ", string(body))
	return &CreateAssetRsp{IsAlreadyExist: len(endpoints) == 0, Endpoints: endpoints}, nil
}

// DeleteAsset deletes a user asset.
func (s *webserver) DeleteAsset(ctx context.Context, userID, assetCID string) error {
	url := fmt.Sprintf("%s/api/v1/storage/delete_asset?user_id=%s&asset_cid=%s", s.url, userID, assetCID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return fmt.Errorf("status code %d, %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	return nil
}

// ShareAsset shares user assets.
func (s *webserver) ShareAsset(ctx context.Context, userID, areaID, assetCID string) (*ShareAssetResult, error) {
	url := fmt.Sprintf("%s/api/v1/storage/share_asset?user_id=%s&area_id=%s&asset_cid=%s", s.url, userID, areaID, assetCID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return nil, fmt.Errorf("status code %d, %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	result := &ShareAssetResult{}
	err = interfaceToStruct(ret.Data, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetCandidateIPs retrieves candidate IPs.
func (s *webserver) GetCandidateIPs(ctx context.Context) ([]*CandidateIPInfo, error) {
	return nil, nil
}

// ListAssets lists user assets.
func (s *webserver) ListAssets(ctx context.Context, parent, pageSize, page int) (*ListAssetRecordRsp, error) {
	url := fmt.Sprintf("%s/api/v1/storage/get_asset_group_list?parent=%d&page_size=%d&page=%d", s.url, parent, pageSize, page)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return nil, fmt.Errorf("status code %d %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	type Object struct {
		AssetOverview *AssetOverview `json:"AssetOverview"`
	}

	data := struct {
		List  []*Object `json:"list"`
		Total int       `json:"total"`
	}{}

	err = interfaceToStruct(ret.Data, &data)
	if err != nil {
		return nil, err
	}

	assetOverviews := make([]*AssetOverview, 0)
	for _, obj := range data.List {
		if obj.AssetOverview == nil {
			continue
		}
		assetOverviews = append(assetOverviews, obj.AssetOverview)
	}
	// fmt.Println("body ", string(body))
	return &ListAssetRecordRsp{Total: data.Total, AssetOverviews: assetOverviews}, nil
}

// CreateGroup create a group
func (s *webserver) CreateGroup(ctx context.Context, name string, parent int) (*AssetGroup, error) {
	url := fmt.Sprintf("%s/api/v1/storage/create_group?&name=%s&parent=%d", s.url, name, parent)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return nil, fmt.Errorf("status code %d, %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	data := struct {
		Group *AssetGroup `json:"group"`
	}{}
	err = interfaceToStruct(ret.Data, &data)
	if err != nil {
		return nil, err
	}

	return data.Group, nil
}

// ListGroups list Asset group
func (s *webserver) ListGroups(ctx context.Context, parent, pageSize, page int) (*ListAssetGroupRsp, error) {
	url := fmt.Sprintf("%s/api/v1/storage/get_groups?parent=%d&page_size=%d&page=%d", s.url, parent, pageSize, page)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %d", rsp.StatusCode)
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	listAssetGroupRsp := &ListAssetGroupRsp{}
	err = interfaceToStruct(ret.Data, listAssetGroupRsp)
	if err != nil {
		return nil, err
	}

	return listAssetGroupRsp, nil
}

// ListAssetSummary list Asset and group
func (s *webserver) ListAssetSummary(ctx context.Context, userID string, parent, limit, offset int) (*ListAssetSummaryRsp, error) {
	return nil, nil
}

// DeleteGroup delete a group
func (s *webserver) DeleteGroup(ctx context.Context, userID string, gid int) error {
	url := fmt.Sprintf("%s/api/v1/storage/delete_group?user_id=%s&group_id=%d", s.url, userID, gid)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return fmt.Errorf("status code %d %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}
	return nil
}

// RenameGroup rename group
func (s *webserver) RenameGroup(ctx context.Context, userID, newName string, groupID int) error {
	return fmt.Errorf("not implemnet")
}

// MoveAssetToGroup move a asset to group
func (s *webserver) MoveAssetToGroup(ctx context.Context, userID, cid string, groupID int) error {
	return fmt.Errorf("not implemnet")
}

// MoveAssetGroup move a asset group
func (s *webserver) MoveAssetGroup(ctx context.Context, userID string, groupID, targetGroupID int) error {
	return fmt.Errorf("not implemnet")
}

// GetAPPKeyPermissions get the permissions of user app key
func (s *webserver) GetAPPKeyPermissions(ctx context.Context, userID, keyName string) ([]string, error) {
	return nil, nil
}

// GetNodeUploadInfo
func (s *webserver) GetNodeUploadInfo(ctx context.Context, userID string, urlMode bool) (*UploadInfo, error) {
	url := fmt.Sprintf("%s/api/v1/storage/get_upload_info?encrypted=false&area_id=%s", s.url, s.area)
	if urlMode {
		url += "&urlMode=true"
	}

	fmt.Println("GetUploadInfo url: ", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.apiKey)

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(rsp.Body)
		return nil, fmt.Errorf("status code %d %s", rsp.StatusCode, string(buf))
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	uploadNodes := &UploadInfo{}
	err = interfaceToStruct(ret.Data, uploadNodes)
	if err != nil {
		return nil, err
	}

	return uploadNodes, nil
}

func interfaceToStruct(input interface{}, output interface{}) error {
	buf, err := json.Marshal(input)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, output)
	if err != nil {
		return err
	}
	return nil
}
