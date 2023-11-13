package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AssetProperty struct {
	AssetCID  string
	AssetName string
	AssetSize int64
	AssetType string
	NodeID    string
}

type CreateAssetRsp struct {
	UploadURL     string
	Token         string
	AlreadyExists bool
}

// NodeIPInfo
type CandidateIPInfo struct {
	NodeID      string
	IP          string
	ExternalURL string
}

type ReplicaInfo struct {
	Hash        string
	NodeID      string
	Status      int
	IsCandidate bool
	EndTime     time.Time
	DoneSize    int64
}

// AssetRecord represents information about an asset record
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

type AssetOverview struct {
	AssetRecord      *AssetRecord
	UserAssetDetail  *UserAssetDetail
	VisitCount       int
	RemainVisitCount int
}

// ListAssetRecordRsp list asset records
type ListAssetRecordRsp struct {
	Total          int              `json:"total"`
	AssetOverviews []*AssetOverview `json:"asset_infos"`
}

type Scheduler interface {
	CreateUserAsset(ctx context.Context, ap *AssetProperty) (*CreateAssetRsp, error)
	DeleteUserAsset(ctx context.Context, assetCID string) error
	ShareUserAssets(ctx context.Context, assetCID []string) (map[string]string, error)
	GetCandidateIPs(ctx context.Context) ([]*CandidateIPInfo, error)
	ListUserAssets(ctx context.Context, limit, offset int) (*ListAssetRecordRsp, error) //perm:user
}

var _ Scheduler = (*scheduler)(nil)

func NewScheduler(url string, header http.Header, opts ...Option) Scheduler {
	options := []Option{URLOption(url), HeaderOption(header)}
	options = append(options, opts...)

	client := NewClient(options...)

	return &scheduler{client: client}
}

type scheduler struct {
	client *Client
}

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
