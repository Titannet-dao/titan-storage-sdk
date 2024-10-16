package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/utopiosphe/titan-storage-sdk/client"
)

type Tenant interface {
	// SSOLogin login sub account user, if user not exist, will create the account automatically
	SSOLogin(ctx context.Context, req SubUserInfo) (*SSOLoginRsp, error)
	// SyncUser sync user info to titan explorer account
	SyncUser(ctx context.Context, req SubUserInfo) error
	// DeleteUser delete user from titan explorer
	DeleteUser(ctx context.Context, entryUUID string, withAssets bool) error
	// RefreshToken refresh user token from titan explorer
	RefreshToken(ctx context.Context, token string) (*SSOLoginRsp, error)
	// ValidateUploadCallback validate upload callback request from titan-explorer
	ValidateUploadCallback(ctx context.Context, apiSecret string, r *http.Request) (*AssetUploadNotifyCallback, error)
	// ValidateDeleteCallback validate delete callback request from titan-explorer
	ValidateDeleteCallback(ctx context.Context, apiSecret string, r *http.Request) (*AssetDeleteNotifyCallback, error)
}

type tenant struct {
	titanUrl  string
	tenantKey string
	client    *http.Client
}

func NewTenant(titanUrl, tenantKey string) (Tenant, error) {
	if len(titanUrl) == 0 || len(tenantKey) == 0 {
		return nil, fmt.Errorf("TitanURL or APIKey can not empty")
	}

	return &tenant{
		titanUrl:  titanUrl,
		tenantKey: tenantKey,
		client:    http.DefaultClient,
	}, nil
}

type SubUserInfo struct {
	EntryUUID string `json:"entry_uuid"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar"`
	Email     string `json:"email"`
}

type SSOLoginRsp struct {
	Token string `json:"token"`
	Exp   int64  `json:"exp"`
}

// SSOLogin login sub account user, if user not exist, will create the account automatically
func (t *tenant) SSOLogin(ctx context.Context, req SubUserInfo) (*SSOLoginRsp, error) {
	url := fmt.Sprintf("%s/api/v1/tenant/sso_login", t.titanUrl)

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("tenant-api-key", t.tenantKey)
	rsp, err := t.client.Do(request)
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

	ret := &client.Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	ssoLoginRsp := &SSOLoginRsp{}
	err = interfaceToStruct(ret.Data, ssoLoginRsp)
	if err != nil {
		return nil, err
	}

	return ssoLoginRsp, nil
}

// SyncUser sync user info to titan explorer account
func (t *tenant) SyncUser(ctx context.Context, req SubUserInfo) error {
	url := fmt.Sprintf("%s/api/v1/tenant/sync_user", t.titanUrl)

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("tenant-api-key", t.tenantKey)
	rsp, err := t.client.Do(request)
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

	ret := &client.Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	return nil
}

// DeleteUser delete user from titan explorer
func (t *tenant) DeleteUser(ctx context.Context, entryUUID string, withAssets bool) error {
	url := fmt.Sprintf("%s/api/v1/tenant/delete_user?entry_uuid=%s&with_assets=%t", t.titanUrl, entryUUID, withAssets)

	request, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("tenant-api-key", t.tenantKey)
	rsp, err := t.client.Do(request)
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

	ret := &client.Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	return nil
}

// RefreshToken refresh user token from titan explorer
func (t *tenant) RefreshToken(ctx context.Context, token string) (*SSOLoginRsp, error) {
	url := fmt.Sprintf("%s/api/v1/tenant/refresh_token?token=%s", t.titanUrl, token)

	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("tenant-api-key", t.tenantKey)
	rsp, err := t.client.Do(request)
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

	ret := &client.Result{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(fmt.Sprintf("code: %d, err: %d, msg: %s", ret.Code, ret.Err, ret.Msg))
	}

	ssoLoginRsp := &SSOLoginRsp{}
	err = interfaceToStruct(ret.Data, ssoLoginRsp)
	if err != nil {
		return nil, err
	}

	return ssoLoginRsp, nil
}

type AssetUploadNotifyCallback struct {
	ExtraID  string // outer file id
	TenantID string //
	UserID   string //

	AssetName      string
	AssetCID       string
	AssetType      string
	AssetSize      int64
	GroupID        int64
	CreatedTime    time.Time
	AssetDirectUrl string
}

var (
	processedNonces = make(map[string]bool)
	nonceMutex      sync.Mutex
)

// ValidateUploadCallback validate upload callback request from titan-explorer
func (t *tenant) ValidateUploadCallback(ctx context.Context, apiSecret string, r *http.Request) (*AssetUploadNotifyCallback, error) {
	signature := r.Header.Get("X-Signature")
	timestamp := r.Header.Get("X-Timestamp")
	nonce := r.Header.Get("X-Nonce")

	// read callback body content
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	defer r.Body.Close()

	// validate timestamp to avoid replay attack
	requestTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil || time.Since(requestTime) > 5*time.Minute {
		return nil, fmt.Errorf("invalid or expired timestamp: %v", err)
	}

	// validate nonce to make sure same callback not received twice
	nonceMutex.Lock()
	if processedNonces[nonce] {
		nonceMutex.Unlock()
		return nil, fmt.Errorf("nonce already used: %s", nonce)
	}
	processedNonces[nonce] = true
	nonceMutex.Unlock()

	// validate signature
	path := fmt.Sprintf("%s://%s%s", r.URL.Scheme, r.Host, r.URL.String())
	expectedSignature := genCallbackSignature(apiSecret, r.Method, path, string(body), timestamp, nonce)
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		log.Printf("signature: %s, expected: %s\n", signature, expectedSignature)
		log.Printf("apiSecret: %s\n", apiSecret)
		log.Printf("r.Method: %s\n", r.Method)
		log.Printf("r.URL.Path: %s\n", path)
		log.Printf("r.Body: %s\n", string(body))
		log.Printf("timestamp: %s\n", timestamp)
		log.Printf("nonce: %s\n", nonce)
		return nil, fmt.Errorf("invalid signature: %v", err)
	}

	var payload AssetUploadNotifyCallback
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %s", err)
	}

	return &payload, nil
}

type AssetDeleteNotifyCallback struct {
	ExtraID  string // outer file id
	TenantID string //
	UserID   string //
	AssetCID string
}

// ValidateDeleteCallback validate delete callback request from titan-explorer
func (t *tenant) ValidateDeleteCallback(ctx context.Context, apiSecret string, r *http.Request) (*AssetDeleteNotifyCallback, error) {

	signature := r.Header.Get("X-Signature")
	timestamp := r.Header.Get("X-Timestamp")
	nonce := r.Header.Get("X-Nonce")

	// read callback body content
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	defer r.Body.Close()

	// validate timestamp to avoid replay attack
	requestTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil || time.Since(requestTime) > 5*time.Minute {
		return nil, fmt.Errorf("invalid or expired timestamp: %v", err)
	}

	// validate nonce to make sure same callback not received twice
	nonceMutex.Lock()
	if processedNonces[nonce] {
		nonceMutex.Unlock()
		return nil, fmt.Errorf("nonce already used: %s", nonce)
	}
	processedNonces[nonce] = true
	nonceMutex.Unlock()

	// validate signature
	path := fmt.Sprintf("%s://%s%s", r.URL.Scheme, r.Host, r.URL.String())
	expectedSignature := genCallbackSignature(apiSecret, r.Method, path, string(body), timestamp, nonce)
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		return nil, fmt.Errorf("invalid signature: %v", err)
	}

	var payload AssetDeleteNotifyCallback
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %s", err)
	}

	return &payload, nil
}

func genCallbackSignature(secret, method, path, body, timestamp, nonce string) string {
	data := method + path + body + timestamp + nonce
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
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
