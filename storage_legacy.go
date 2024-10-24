package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/Titannet-dao/titan-storage-sdk/client"
	"github.com/Titannet-dao/titan-storage-sdk/memfile"
	byterange "github.com/Titannet-dao/titan-storage-sdk/range"
	"github.com/ipfs/go-cid"
)

// uploadFileWithForm uploads a file using a multipart form
func (s *storage) uploadFileWithForm(ctx context.Context, r io.Reader, name, uploadURL, token, trace_id string, progress ProgressFunc) (*UploadFileResult, error) {
	// Create a new multipart form body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a new form field for the file
	fileField, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, err
	}

	// Copy the file data to the form field
	_, err = io.Copy(fileField, r)
	if err != nil {
		return nil, err
	}

	// Close the multipart form
	err = writer.Close()
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("new request error %s", err.Error())
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	request = request.WithContext(ctx)

	start := time.Now()

	// Create an HTTP client and send the request
	httpClient := http.DefaultClient
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("do error %s", err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("http StatusCode %d,  %s", response.StatusCode, string(b))
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	report := &client.AssetTransferReq{
		CostMs:       int64(time.Since(start).Milliseconds()),
		TotalSize:    int64(totalSize),
		TransferType: client.AssetTransferTypeUpload,
		State:        client.AssetTransferStateFailed,
		TraceID:      trace_id,
	}
	defer func(r *client.AssetTransferReq) {
		if err := s.webAPI.AssetTransferReport(context.Background(), *r); err != nil {
			log.Printf("failed to send transfer report, %s", err.Error())
		}
	}(report)

	var ret UploadFileResult
	if err := json.Unmarshal(b, &ret); err != nil {
		log.Printf("Upload file to L1 node error, url %s, name %s, error: %s \n", uploadURL, name, err.Error())
		return nil, err

	}

	if ret.Code != 0 {
		log.Printf("Upload file to L1 node error, url %s, name %s, ret: %+v \n", uploadURL, name, ret)
		return nil, fmt.Errorf(ret.Msg)
	}

	ret.totalSize = int64(totalSize)

	report.Cid = ret.Cid
	report.State = client.AssetTransferStateSuccess

	return &ret, nil
}

func (s *storage) uploadFilesWithPathAndMakeCar(ctx context.Context, filePath string, progress ProgressFunc) (cid.Cid, error) {
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

	assetProperty := client.AssetProperty{
		AssetCID:  root.String(),
		AssetName: fileName,
		AssetSize: fileInfo.Size(),
		AssetType: fileType,
		NodeID:    s.candidateID,
		GroupID:   s.groupID,
	}

	req := client.CreateAssetReq{AssetProperty: assetProperty, AreaIDs: s.areas}
	rsp, err := s.webAPI.CreateAsset(ctx, &req)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("CreateAsset error %w", err)
	}

	if rsp.IsAlreadyExist {
		return root, nil
	}

	if len(rsp.Endpoints) == 0 {
		return cid.Cid{}, fmt.Errorf("endpoints is empty")
	}

	var (
		report = &client.AssetTransferReq{
			// CostMs:       int64(time.Since(start).Milliseconds()),
			TotalSize:    assetProperty.AssetSize,
			TransferType: client.AssetTransferTypeUpload,
			Cid:          root.String(),
			State:        client.AssetTransferStateFailed,
		}
		logContent = make(map[string]string)
		// final      bool
	)

	defer func(r *client.AssetTransferReq) {
		if err := s.webAPI.AssetTransferReport(context.Background(), *r); err != nil {
			log.Printf("failed to send transfer report, %s", err.Error())
		}
	}(report)

	for _, ep := range rsp.Endpoints {

		nodeid := getNodeIdFromCandidateAddr(ep.CandidateAddr)
		report.TraceID = ep.TraceID
		// if upload succeed,
		report.NodeID = joinNodeID(report.NodeID, nodeid)

		start := time.Now()
		_, err = s.uploadFileWithForm(ctx, carFile, fileName, ep.CandidateAddr, ep.Token, ep.TraceID, progress)
		report.CostMs = int64(time.Since(start).Milliseconds())

		if err == nil {
			report.State = client.AssetTransferStateSuccess
			return root, nil
		} else {
			fmt.Printf("upload req: %+v\n", ep)
			logContent[nodeid] = err.Error()
			// go func(r *client.AssetTransferReq) {
			// 	if err := s.webAPI.AssetTransferReport(context.Background(), *r); err != nil {
			// 		log.Printf("failed to send transfer report, %s", err.Error())
			// 	}
			// }(report)

			if delErr := s.webAPI.DeleteAsset(ctx, s.userID, root.String()); delErr != nil {
				return cid.Cid{}, fmt.Errorf("uploadFileWithForm failed %s, delete error %s", err.Error(), delErr.Error())
			}
			// return cid.Cid{}, fmt.Errorf("uploadFileWithForm error %s, delete it from titan", err.Error())
		}

	}

	return cid.Cid{}, fmt.Errorf("upload file failed")
}

// UploadFilesWithPath uploads files from the specified path
func (s *storage) UploadFilesWithPath(ctx context.Context, filePath string, progress ProgressFunc, makeCar bool) (cid.Cid, error) {
	if makeCar {
		return s.uploadFilesWithPathAndMakeCar(ctx, filePath, progress)
	}

	rsp, err := s.webAPI.GetNodeUploadInfo(ctx, s.userID, s.getArea(), false)
	if err != nil {
		return cid.Cid{}, err
	}

	if rsp.AlreadyExists {
		return cid.Cid{}, fmt.Errorf("file already exists")
	}

	if len(rsp.List) == 0 {
		return cid.Cid{}, fmt.Errorf("endpoints is empty")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return cid.Cid{}, err
	}
	defer f.Close()

	node := rsp.List[0]

	ret, err := s.uploadFileWithForm(ctx, f, f.Name(), node.UploadURL, node.Token, rsp.TraceID, progress)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("upload file with form failed, %s", err.Error())
	}

	if ret.Code != 0 {
		return cid.Cid{}, fmt.Errorf("upload file with form failed, %s", ret.Msg)
	}

	root, err := cid.Decode(ret.Cid)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("decode cid %s failed, %s", ret.Cid, err.Error())
	}

	fileType, err := getFileType(filePath)
	if err != nil {
		return cid.Cid{}, err
	}

	fileInfo, err := f.Stat()
	if err != nil {
		return cid.Cid{}, err
	}

	fmt.Printf("f name %s, fileInfo name %s", f.Name(), fileInfo.Name())

	assetProperty := client.AssetProperty{
		AssetCID:  ret.Cid,
		AssetName: fileInfo.Name(),
		AssetSize: fileInfo.Size(),
		AssetType: fileType,
		NodeID:    node.NodeID,
		GroupID:   s.groupID,
	}

	req := client.CreateAssetReq{AssetProperty: assetProperty, AreaIDs: s.areas}
	_, err = s.webAPI.CreateAsset(context.Background(), &req)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("CreateAsset error %w", err)
	}

	return root, nil

}

// Delete deletes the specified asset by rootCID
func (s *storage) Delete(ctx context.Context, rootCID string) error {
	return s.webAPI.DeleteAsset(ctx, s.userID, rootCID)
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

	assetProperty := client.AssetProperty{
		AssetCID:  root.String(),
		AssetName: name,
		AssetSize: int64(len(memFile.Bytes())),
		AssetType: string(FileTypeFile),
		NodeID:    s.candidateID,
		GroupID:   s.groupID,
	}

	req := client.CreateAssetReq{AssetProperty: assetProperty, AreaIDs: s.areas}
	rsp, err := s.webAPI.CreateAsset(ctx, &req)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("CreateAsset error %w", err)
	}
	// fmt.Printf("CreateAsset rsp %+v\n", rsp)
	// for i, v := range rsp.Endpoints {
	// 	fmt.Printf("endpoint[%d] %+v\n", i, v)
	// }
	if rsp.IsAlreadyExist {
		return root, nil
	}

	c := len(rsp.Endpoints)
	for i, ep := range rsp.Endpoints {
		_, err = s.uploadFileWithForm(ctx, memFile, root.String(), ep.CandidateAddr, ep.Token, ep.TraceID, progress)
		if err != nil {
			// fmt.Printf("upload req: %+v\n", ep)
			// return cid.Cid{}, fmt.Errorf("uploadFileWithForm error %s, delete it from titan", err.Error())
			log.Printf("uploadFileWithForm error %s, delete it from titan\n", err.Error())
		}
		if err != nil && i+1 == c {
			if delErr := s.webAPI.DeleteAsset(ctx, s.userID, root.String()); delErr != nil {
				return cid.Cid{}, fmt.Errorf("uploadFileWithForm failed %s, delete error %s", err.Error(), delErr.Error())
			}
			return cid.Cid{}, err
		}
		if err == nil {
			return root, nil
		}
	}

	return cid.Cid{}, fmt.Errorf("upload file failed")
}

// UploadStreamV2 uploads data from an io.Reader stream without making car to the titan storage.
func (s *storage) UploadStreamV2(ctx context.Context, r io.Reader, name string, progress ProgressFunc) (cid.Cid, error) {
	rsp, err := s.webAPI.GetNodeUploadInfo(ctx, s.userID, s.getArea(), false)
	if err != nil {
		return cid.Cid{}, err
	}

	if rsp.AlreadyExists {
		return cid.Cid{}, fmt.Errorf("file already exists")
	}

	if len(rsp.List) == 0 {
		return cid.Cid{}, fmt.Errorf("endpoints is empty")
	}

	// var size int64

	// if f, ok := r.(*os.File); ok {
	// 	if stat, err := f.Stat(); err != nil {
	// 		return cid.Cid{}, fmt.Errorf("read file failed")
	// 	} else {
	// 		size = stat.Size()
	// 	}
	// }

	// if seeker, ok := r.(io.Seeker); ok {
	// 	currentPos, err := seeker.Seek(0, io.SeekCurrent)
	// 	if err != nil {
	// 		return cid.Cid{}, fmt.Errorf("seek content failed")
	// 	}
	// 	size, err = seeker.Seek(0, io.SeekEnd)
	// 	if err != nil {
	// 		return cid.Cid{}, fmt.Errorf("seek content failed")
	// 	}
	// 	_, err = seeker.Seek(currentPos, io.SeekStart)
	// 	if err != nil {
	// 		return cid.Cid{}, fmt.Errorf("return position failed")
	// 	}
	// }

	// if br, ok := r.(*bytes.Reader); ok {
	// 	size = int64(br.Len())
	// }

	// if sr, ok := r.(*strings.Reader); ok {
	// 	size = int64(sr.Len())
	// }

	// f, err := os.Open(filePath)
	// if err != nil {
	// 	return cid.Cid{}, err
	// }
	// defer f.Close()

	// node := rsp.List[0]

	var (
		ret    *UploadFileResult
		root   cid.Cid
		nodeId string
	)

	body := &bytes.Buffer{}
	writer := io.MultiWriter(body)
	io.Copy(writer, r)
	cnt := body.Bytes()

	for _, node := range rsp.List {
		nodeId = node.NodeID

		nr := bytes.NewReader(cnt)

		ret, err = s.uploadFileWithForm(ctx, nr, name, node.UploadURL, node.Token, rsp.TraceID, progress)
		if err != nil {
			err = fmt.Errorf("upload file with form failed, error: %s", err.Error())
			log.Println(err)
			continue
		}

		if ret.Code != 0 {
			err = fmt.Errorf("upload file with form failed, ret: %+v", ret)
			log.Println(err)
			continue
		}

		root, err = cid.Decode(ret.Cid)
		if err != nil {
			err = fmt.Errorf("decode cid %s failed, reason: %s", ret.Cid, err.Error())
			log.Println(err)
			continue
		}

		break
	}

	if err != nil {
		return cid.Cid{}, err
	}

	// fileType, err := getFileType(name)
	// if err != nil {
	// 	return cid.Cid{}, err
	// }

	// fileInfo, err := f.Stat()
	// if err != nil {
	// 	return cid.Cid{}, err
	// }

	log.Printf("f name %s, fileType name %s \n", name, "file")

	assetProperty := client.AssetProperty{
		AssetCID:  ret.Cid,
		AssetName: name,
		AssetSize: ret.totalSize,
		AssetType: "file",
		NodeID:    nodeId,
		GroupID:   s.groupID,
	}

	req := client.CreateAssetReq{AssetProperty: assetProperty, AreaIDs: s.areas}
	_, err = s.webAPI.CreateAsset(context.Background(), &req)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("CreateAsset error %w", err)
	}

	return root, nil

}

// GetFileWithCid gets a single file by rootCID
func (s *storage) GetFileWithCid(ctx context.Context, rootCID string) (io.ReadCloser, string, error) {
	res, err := s.GetURL(ctx, rootCID)
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
		Cid:          rootCID,
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
	// taskCount := int64(len(res.URLs))
	// if !parallel {
	// 	taskCount = 1
	// }

	// type downloadReq struct {
	// 	url        string
	// 	start, end int64
	// 	index      int
	// }

	// var (
	// 	wg            sync.WaitGroup
	// 	mu            sync.Mutex
	// 	chunkSize     = res.Size / taskCount
	// 	chunks        = make([][]byte, taskCount)
	// 	firstFailChan = make(chan downloadReq, taskCount)
	// 	fastestTime   = math.MaxInt
	// 	fastestTask   downloadReq
	// )

	// downloadChunk := func(d downloadReq, fc chan downloadReq) {
	// 	defer wg.Done()
	// 	start := time.Now()

	// 	req, err := http.NewRequest("GET", d.url, nil)
	// 	if err != nil {
	// 		fc <- d
	// 		return
	// 	}
	// 	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", d.start, d.end))
	// 	resp, err := http.DefaultClient.Do(req)
	// 	if err != nil {
	// 		fc <- d
	// 		return
	// 	}

	// 	defer func() {
	// 		if resp != nil && resp.Body != nil {
	// 			resp.Body.Close()
	// 		}
	// 	}()

	// 	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
	// 		log.Printf("failed to download chunk: %d-%d, status code: %d", d.start, d.end, resp.StatusCode)
	// 		fc <- d
	// 		return
	// 	}

	// 	data, err := io.ReadAll(resp.Body)
	// 	if err != nil {
	// 		fc <- d
	// 		return
	// 	}
	// 	elapsed := time.Since(start)
	// 	log.Printf("Chunk: %fs, Link: %s", elapsed.Seconds(), d.url)

	// 	mu.Lock()
	// 	if int(elapsed.Milliseconds()) < fastestTime {
	// 		fastestTime = int(elapsed.Milliseconds())
	// 		fastestTask = d
	// 	}
	// 	chunks[d.index] = data
	// 	mu.Unlock()
	// }

	// for i := 0; i < int(taskCount); i++ {
	// 	wg.Add(1)
	// 	start := int64(i) * chunkSize
	// 	end := start + chunkSize - 1
	// 	if i == int(taskCount)-1 {
	// 		end = res.Size - 1 // Ensure last chunk covers the remainder
	// 	}
	// 	go downloadChunk(downloadReq{res.URLs[i], start, end, i}, firstFailChan)
	// }

	// // wait normal
	// wg.Wait()
	// close(firstFailChan)

	// secFailChan := make(chan downloadReq, len(firstFailChan))
	// for fail := range firstFailChan {
	// 	wg.Add(1)
	// 	fail.url = fastestTask.url
	// 	log.Printf("retry download chunk: %d-%d", fail.start, fail.end)
	// 	downloadChunk(fail, secFailChan)
	// }
	// // wait failed
	// wg.Wait()
	// close(secFailChan)

	// if len(secFailChan) > 0 {
	// 	return nil, "", fmt.Errorf("failed to download all chunks")
	// }

	// readers := make([]io.Reader, len(chunks))
	// for i, chunk := range chunks {
	// 	readers[i] = bytes.NewReader(chunk)
	// }

	// multiReader := io.MultiReader(readers...)

	// return io.NopCloser(multiReader), res.FileName, nil
}

// GetURL gets the URL of the file
func (s *storage) GetURL(ctx context.Context, rootCID string) (*client.ShareAssetResult, error) {
	// 100 ms
	var interval = 1000
	var startTime = time.Now()
	var timeout = 15 * time.Second
	for {
		time.Sleep(time.Millisecond * time.Duration(interval))

		if time.Since(startTime) > timeout {
			return nil, fmt.Errorf("time out of %ds, can not find asset exist", timeout/time.Second)
		}

		result, err := s.webAPI.ShareAsset(ctx, s.userID, "", rootCID)
		if err != nil {
			log.Printf("ShareUserAsset %v, cid: %s \n", err.Error(), rootCID)
			// if err.Error() != errAssetNotExist(rootCID).Error() {
			// 	return nil, fmt.Errorf("ShareUserAssets %w", err)
			// }
			continue
		}

		if len(result.URLs) > 0 {
			for i := range result.URLs {
				result.URLs[i] = replaceNodeIDToCID(result.URLs[i], rootCID)
			}
			u, _ := url.ParseRequestURI(result.URLs[0])
			if err != nil {
				return result, nil
			}
			if u != nil && u.Query().Get("filename") != "" {
				result.FileName = u.Query().Get("filename")
			}
			return result, nil
		}

	}
}

// UploadFileWithURL uploads a file from the specified URL
func (s *storage) UploadFileWithURL(ctx context.Context, url string, progress ProgressFunc) (string, string, error) {
	log.Println("UploadFileWithURL link:", url)
	rsp, err := http.Get(url)
	if err != nil {
		log.Printf("http.Get(%s) error: %s \n", url, err)
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

	rootCid, err := s.UploadStreamV2(ctx, rsp.Body, filename, progress)
	if err != nil {
		return "", "", err
	}

	// try 5 times to fetch url , otherwise return error
	var res *client.ShareAssetResult
	for i := 0; i < 1; i++ {
		time.Sleep(time.Second * 1)
		res, err = s.GetURL(ctx, rootCid.String())
		if err != nil {
			log.Printf("get url err: %s\n", err.Error())
			continue
		}
		break
	}

	if res == nil {
		return "", "", err
	}

	return rootCid.String(), res.URLs[0], nil
}

// UploadFileWithURLV2 uploads a url and let L1 to download it, returns
func (s *storage) UploadFileWithURLV2(ctx context.Context, url string, progress ProgressFunc) (string, string, error) {

	rsp, err := s.webAPI.GetNodeUploadInfo(ctx, s.userID, s.getArea(), true)
	if err != nil {
		return "", "", err
	}

	nodes := rsp.List
	if len(nodes) == 0 {
		return "", "", fmt.Errorf("endpoints is empty")
	}

	return "", "", nil

}

func (s *storage) ListUserAssets(ctx context.Context, parent, pageSize, page int) (*client.ListAssetRecordRsp, error) {
	return s.webAPI.ListAssets(ctx, parent, pageSize, page, "", 0)
}

// CreateGroup create a group
func (s *storage) CreateGroup(ctx context.Context, name string, parent int) error {
	_, err := s.webAPI.CreateGroup(ctx, name, parent)
	return err
}

// ListGroup list groups
func (s *storage) ListGroups(ctx context.Context, parent, pageSize, page int) (*client.ListAssetGroupRsp, error) {
	return s.webAPI.ListGroups(ctx, parent, pageSize, page)
}

// DeleteGroup delete special group
func (s *storage) DeleteGroup(ctx context.Context, groupID int) error {
	return s.webAPI.DeleteGroup(ctx, s.userID, groupID)
}

func (s *storage) getArea() string {
	if len(s.areas) > 0 {
		return s.areas[0]
	}
	return ""
}
