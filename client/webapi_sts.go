package client

import "time"

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
	AreaIDs []string
	AssetProperty
}

// CreateAssetRsp represents the response when creating an asset.
type Endpoint struct {
	CandidateAddr string
	Token         string
	AlreadyExists bool
	TraceID       string
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
	TraceID       string
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
	TraceID  string   `json:"trace_id"`
	FileName string
}

type Result struct {
	Code int    `json:"code"`
	Err  int    `json:"err"`
	Msg  string `json:"msg"`
	Data interface{}
}

type AssetTransferReq struct {
	TraceID      string `json:"trace_id"`
	UserId       string `json:"user_id"`
	Cid          string `json:"cid"`
	Hash         string `json:"hash"`
	NodeID       string `json:"node_id"`
	Rate         int64  `json:"rate"`
	CostMs       int64  `json:"cost_ms"`
	TotalSize    int64  `json:"total_size"`
	State        int64  `json:"state"`
	TransferType string `json:"transfer_type"`
	Log          string `json:"log"`
}
