package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AssetProperty represents the properties of an asset.
type AssetProperty struct {
	AssetCID  string
	AssetName string
	AssetSize int64
	AssetType string
	NodeID    string
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

// Scheduler defines the interface for the scheduler.
type Scheduler interface {
	// CreateUserAsset creates a new user asset.
	CreateUserAsset(ctx context.Context, ap *AssetProperty) (*CreateAssetRsp, error)
	// DeleteUserAsset deletes a user asset.
	DeleteUserAsset(ctx context.Context, assetCID string) error
	// ShareUserAssets shares user assets.
	ShareUserAssets(ctx context.Context, assetCID []string) (map[string]string, error)
	// GetCandidateIPs retrieves information about candidate IPs.
	GetCandidateIPs(ctx context.Context) ([]*CandidateIPInfo, error)
	// ListUserAssets retrieves a list of user assets with optional limit and offset.
	ListUserAssets(ctx context.Context, limit, offset int) (*ListAssetRecordRsp, error) //perm:user
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

// CreateUserAsset creates a new user asset.
func (s *scheduler) CreateUserAsset(ctx context.Context, ap *AssetProperty) (*CreateAssetRsp, error) {
	serializedParams := params{
		ap,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.CreateUserAsset",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	createAssetRsp := &CreateAssetRsp{}
	err = json.Unmarshal(b, &createAssetRsp)
	if err != nil {
		return nil, err
	}

	return createAssetRsp, nil
}

// DeleteUserAsset deletes a user asset.
func (s *scheduler) DeleteUserAsset(ctx context.Context, assetCID string) error {
	serializedParams := params{
		assetCID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.DeleteUserAsset",
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
func (s *scheduler) ShareUserAssets(ctx context.Context, assetCID []string) (map[string]string, error) {
	serializedParams := params{
		assetCID,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.ShareUserAssets",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
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

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	ret := make([]*CandidateIPInfo, 0)
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// ListUserAssets lists user assets.
func (s *scheduler) ListUserAssets(ctx context.Context, limit, offset int) (*ListAssetRecordRsp, error) {
	serializedParams := params{
		limit,
		offset,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.ListUserAssets",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := s.client.request(ctx, req)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, fmt.Errorf("%s code %d ", rsp.Error.Message, rsp.Error.Code)
	}

	ret := ListAssetRecordRsp{}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}
