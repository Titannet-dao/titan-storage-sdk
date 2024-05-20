package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/xerrors"
)

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
	UserID string
	AssetProperty
}

// CreateAssetRsp represents the response when creating an asset.
type CreateAssetRsp struct {
	UploadURL     string
	Token         string
	AlreadyExists bool
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
	Total          int              `json:"total"`
	AssetOverviews []*AssetOverview `json:"asset_infos"`
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
	AssetGroups []*AssetGroup `json:"infos"`
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
	UploadURL     string
	Token         string
	NodeID        string
	AlreadyExists bool
}

// Scheduler defines the interface for the scheduler.
type Scheduler interface {
	// AuthVerify checks whether the specified token is valid and returns the list of permissions associated with it.
	AuthVerify(ctx context.Context, token string) (*JWTPayload, error)
	// CreateAsset creates an asset with car CID, car name, and car size.
	CreateAsset(ctx context.Context, req *CreateAssetReq) (*CreateAssetRsp, error)
	// DeleteAsset deletes the asset of the user.
	DeleteAsset(ctx context.Context, userID, assetCID string) error
	// ShareAssets shares the assets of the user.
	ShareAssets(ctx context.Context, userID string, assetCID []string) (map[string]string, error)
	// GetCandidateIPs retrieves information about candidate IPs.
	GetCandidateIPs(ctx context.Context) ([]*CandidateIPInfo, error)
	// ListAssets lists the assets of the user.
	ListAssets(ctx context.Context, userID string, limit, offset, groupID int) (*ListAssetRecordRsp, error)

	// CreateAssetGroup create Asset group
	CreateAssetGroup(ctx context.Context, userID, name string, parent int) (*AssetGroup, error) //perm:user,web,admin
	// ListAssetGroup list Asset group
	ListAssetGroup(ctx context.Context, userID string, parent, limit, offset int) (*ListAssetGroupRsp, error) //perm:user,web,admin
	// ListAssetSummary list Asset and group
	ListAssetSummary(ctx context.Context, userID string, parent, limit, offset int) (*ListAssetSummaryRsp, error) //perm:user,web,admin
	// DeleteAssetGroup delete Asset group
	DeleteAssetGroup(ctx context.Context, userID string, groupID int) error //perm:user,web,admin
	// RenameAssetGroup rename group
	RenameAssetGroup(ctx context.Context, userID, newName string, groupID int) error //perm:user,web,admin
	// MoveAssetToGroup move a asset to group
	MoveAssetToGroup(ctx context.Context, userID, cid string, groupID int) error //perm:user,web,admin
	// MoveAssetGroup move a asset group
	MoveAssetGroup(ctx context.Context, userID string, groupID, targetGroupID int) error //perm:user,web,admin
	// GetAPPKeyPermissions get the permissions of user app key
	GetAPPKeyPermissions(ctx context.Context, userID, keyName string) ([]string, error) //perm:user,web,admin

	// GetNodeUploadInfo
	GetNodeUploadInfo(ctx context.Context, userID string) (*UploadInfo, error) //perm:user,web,admin
}

var _ Scheduler = (*scheduler)(nil)

// NewScheduler creates a new Scheduler instance with the specified URL, headers, and options.
func NewScheduler(url string, header http.Header, opts ...Option) Scheduler {
	options := []Option{URLOption(url), HeaderOption(header)}
	options = append(options, opts...)

	client := NewClient(options...)

	return &scheduler{client: client}
}

type scheduler struct {
	client *Client
}

// // AuthVerify checks whether the specified token is valid and returns the list of permissions associated with it.
func (s *scheduler) AuthVerify(ctx context.Context, token string) (*JWTPayload, error) {
	serializedParams := params{
		token,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.AuthVerify",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	payload := &JWTPayload{}
	err = json.Unmarshal(b, payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// CreateUserAsset creates a new user asset.
func (s *scheduler) CreateAsset(ctx context.Context, caReq *CreateAssetReq) (*CreateAssetRsp, error) {
	serializedParams := params{
		caReq,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.CreateAsset",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		if rsp.Error.Meta != nil {
			errServer := ErrServer{}
			if err = json.Unmarshal(rsp.Error.Meta, &errServer); err != nil {
				return nil, xerrors.Errorf("unmarshal ErrServer error %w, message:%s", err, rsp.Error.Message)
			}

			return nil, &errServer
		}

		return nil, xerrors.New(rsp.Error.Message)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	createAssetRsp := &CreateAssetRsp{}
	err = json.Unmarshal(b, &createAssetRsp)
	if err != nil {
		return nil, err
	}

	return createAssetRsp, nil
}

// DeleteUserAsset deletes a user asset.
func (s *scheduler) DeleteAsset(ctx context.Context, userID, assetCID string) error {
	serializedParams := params{
		userID,
		assetCID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.DeleteAsset",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	return nil

}

// ShareUserAssets shares user assets.
func (s *scheduler) ShareAssets(ctx context.Context, userID string, assetCID []string) (map[string]string, error) {
	serializedParams := params{
		userID,
		assetCID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.ShareAssets",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]string)
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// GetCandidateIPs retrieves candidate IPs.
func (s *scheduler) GetCandidateIPs(ctx context.Context) ([]*CandidateIPInfo, error) {
	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.GetCandidateIPs",
		Params:  nil,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := make([]*CandidateIPInfo, 0)
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// ListUserAssets lists user assets.
func (s *scheduler) ListAssets(ctx context.Context, userID string, limit, offset, groupID int) (*ListAssetRecordRsp, error) {
	serializedParams := params{
		userID,
		limit,
		offset,
		groupID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.ListAssets",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := ListAssetRecordRsp{}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

// CreateAssetGroup create Asset group
func (s *scheduler) CreateAssetGroup(ctx context.Context, userID, name string, parent int) (*AssetGroup, error) {
	serializedParams := params{
		userID,
		name,
		parent,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.CreateAssetGroup",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := AssetGroup{}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

// ListAssetGroup list Asset group
func (s *scheduler) ListAssetGroup(ctx context.Context, userID string, parent, limit, offset int) (*ListAssetGroupRsp, error) {
	serializedParams := params{
		userID,
		parent,
		limit,
		offset,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.ListAssetGroup",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := ListAssetGroupRsp{}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

// ListAssetSummary list Asset and group
func (s *scheduler) ListAssetSummary(ctx context.Context, userID string, parent, limit, offset int) (*ListAssetSummaryRsp, error) {
	serializedParams := params{
		userID,
		parent,
		limit,
		offset,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.ListAssetSummary",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := ListAssetSummaryRsp{}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

// DeleteAssetGroup delete Asset group
func (s *scheduler) DeleteAssetGroup(ctx context.Context, userID string, gid int) error {
	serializedParams := params{
		userID,
		gid,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.DeleteAssetGroup",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	return nil
}

// RenameAssetGroup rename group
func (s *scheduler) RenameAssetGroup(ctx context.Context, userID, newName string, groupID int) error {
	serializedParams := params{
		userID,
		newName,
		groupID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.RenameAssetGroup",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}
	return nil
}

// MoveAssetToGroup move a asset to group
func (s *scheduler) MoveAssetToGroup(ctx context.Context, userID, cid string, groupID int) error {
	serializedParams := params{
		userID,
		cid,
		groupID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.MoveAssetToGroup",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}
	return nil
}

// MoveAssetGroup move a asset group
func (s *scheduler) MoveAssetGroup(ctx context.Context, userID string, groupID, targetGroupID int) error {
	serializedParams := params{
		userID,
		groupID,
		targetGroupID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.MoveAssetGroup",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}
	return nil
}

// GetAPPKeyPermissions get the permissions of user app key
func (s *scheduler) GetAPPKeyPermissions(ctx context.Context, userID, keyName string) ([]string, error) {
	serializedParams := params{
		userID,
		keyName,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.GetAPPKeyPermissions",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := make([]string, 0)
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// GetNodeUploadInfo
func (s *scheduler) GetNodeUploadInfo(ctx context.Context, userID string) (*UploadInfo, error) {
	serializedParams := params{
		userID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.GetNodeUploadInfo",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	ret := UploadInfo{}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
