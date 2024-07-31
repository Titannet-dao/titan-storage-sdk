package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	storage "github.com/Filecoin-Titan/titan-storage-sdk"
	"github.com/spf13/cobra"
)

func getTitanURLAndAPIKeyFromEnv() (string, string, error) {
	titanURL := os.Getenv("TITAN_URL")
	apiKey := os.Getenv("API_KEY")
	if len(titanURL) == 0 {
		return "", "", fmt.Errorf("please set environment variable TITAN_URL, example: export TITAN_URL=Your_titan_url")
	}

	if len(apiKey) == 0 {
		return "", "", fmt.Errorf("please set environment variable API_KEY, example: export API_KEY=Your_API_KEY")
	}

	return titanURL, apiKey, nil
}

func getAreaIDFromEnv() string {
	return os.Getenv("AREA_ID")
}

var rootCmd = &cobra.Command{}
var currentWorkingGroup = 0

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("0.0.1")
	},
}

var uploadCmd = &cobra.Command{
	Use:     "upload",
	Short:   "upload file",
	Example: "upload --make-car=true /path/to/my/file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("Please specify the name of the file to be uploaded")
		}

		filePath := args[0]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Fatalf("File %s does not exist.", filePath)
		}

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey, AreaID: getAreaIDFromEnv()})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		startTime := time.Now()
		fileSize := int64(0)
		progress := func(doneSize int64, totalSize int64) {
			fileSize = totalSize
			log.Printf("total size:%d bytes, done %d bytes\n", totalSize, doneSize)
		}

		makeCar, _ := cmd.Flags().GetBool("make-car")
		cid, err := s.UploadFilesWithPath(context.Background(), filePath, progress, makeCar)
		if err != nil {
			log.Fatal("UploadFilesWithPath ", err)
		}

		costTime := time.Since(startTime) / time.Millisecond
		log.Printf("upload file %s cid %s success %d bytes cost %dms, speed %d B/s\n", filePath, cid.String(), fileSize, costTime, fileSize/int64(costTime)*1000)
	},
}

var listFilesCmd = &cobra.Command{
	Use:     "list",
	Short:   "list files",
	Example: "list --group-id=0 --page-size=20 --page=1",
	Run: func(cmd *cobra.Command, args []string) {
		groupID, _ := cmd.Flags().GetInt("group-id")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		page, _ := cmd.Flags().GetInt("page")

		if pageSize == 0 {
			log.Fatal("please set --page-size")
		}

		if page <= 0 {
			log.Fatal("page-size > 0")
		}

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		rets, err := s.ListUserAssets(context.Background(), groupID, pageSize, page)
		if err != nil {
			log.Fatal("UploadFilesWithPath ", err)
		}

		tw := NewTableWriter(
			Col("CID"),
			Col("Name"),
			Col("Size"),
			Col("CreatedTime"),
			Col("Expiration"),
		)

		for w := 0; w < len(rets.AssetOverviews); w++ {
			asset := rets.AssetOverviews[w]
			m := map[string]interface{}{
				"CID":         asset.AssetRecord.CID,
				"Name":        asset.UserAssetDetail.AssetName,
				"Size":        asset.AssetRecord.TotalSize,
				"CreatedTime": asset.AssetRecord.CreatedTime,
				"Expiration":  asset.AssetRecord.Expiration,
			}

			tw.Write(m)
		}

		tw.Flush(os.Stdout)
	},
}

var getFileCmd = &cobra.Command{
	Use:     "get",
	Short:   "get file",
	Example: "get --cid=you-cid --out=your-file-name",
	Run: func(cmd *cobra.Command, args []string) {
		cid, _ := cmd.Flags().GetString("cid")
		outFileName, _ := cmd.Flags().GetString("out")

		if len(cid) == 0 {
			log.Fatal("Please specify the cid of the file to be get")
		}

		if len(outFileName) == 0 {
			outFileName = cid
		}

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		reader, _, err := s.GetFileWithCid(context.Background(), cid, false)
		if err != nil {
			log.Fatal("UploadFilesWithPath ", err)
		}
		defer reader.Close()

		newFile, err := os.Create(outFileName)
		if err != nil {
			log.Fatal("Create file", err)
		}
		defer newFile.Close()

		startTime := time.Now()
		lastTime := time.Now()
		downloadCount := int64(0)
		size := int64(0)
		progress := func(r int64) {
			size += r
			downloadCount += r
			costTime := time.Since(lastTime)
			if costTime > 10*time.Millisecond {
				log.Printf("downloading %d bytes, speed %d B/s", size, int64(float64(downloadCount)/float64(costTime)*float64(time.Second)))
				downloadCount = 0
				lastTime = time.Now()
			}
		}
		progressReader := &storage.ProgressReader{Reader: reader, Reporter: progress}
		if _, err := io.Copy(newFile, progressReader); err != nil {
			log.Fatal(err)
		}

		costTime := time.Since(startTime) / time.Millisecond
		speed := int64(0)
		if costTime > 0 {
			speed = size / int64(costTime) * 1000
		}
		log.Printf("get %s success cost %d ms, size %d bytes, speed %d B/s", cid, costTime, size, speed)

	},
}

var deleteFileCmd = &cobra.Command{
	Use:     "delete",
	Short:   "delete file",
	Example: "delete your-file-cid",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("Please specify the cid of the file to be delete")
		}

		rootCID := args[0]

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		err = s.Delete(context.Background(), rootCID)
		if err != nil {
			log.Fatal("UploadFilesWithPath ", err)
		}

		log.Printf("delete %s success", rootCID)
	},
}

var getURLCmd = &cobra.Command{
	Use:     "url",
	Short:   "get file ur",
	Example: "url your-file-cid",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("Please specify the cid of the file to be delete")
		}

		rootCID := args[0]

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		url, err := s.GetURL(context.Background(), rootCID)
		if err != nil {
			log.Fatal("GetURL ", err)
		}

		log.Println(url)
	},
}

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage groups",
}

var createGroupCmd = &cobra.Command{
	Use:   "create",
	Short: "create --name abc --pid 0",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		parentID, _ := cmd.Flags().GetInt("parentID")

		fmt.Printf("Adding group %s to %d\n", name, parentID)

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		err = s.CreateGroup(cmd.Context(), name, parentID)
		if err != nil {
			log.Fatal("CreateGroup ", err)
		}

	},
}

var listGroupCmd = &cobra.Command{
	Use:   "list",
	Short: "list --parentID 0 -s 0 -e 20",
	Run: func(cmd *cobra.Command, args []string) {
		parentID, _ := cmd.Flags().GetInt("parentID")
		start, _ := cmd.Flags().GetInt("start")
		end, _ := cmd.Flags().GetInt("end")

		count := end - start
		if count <= 0 {
			log.Fatal("can not special the start and end")
		}

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		rsp, err := s.ListGroups(cmd.Context(), parentID, count, start)
		if err != nil {
			log.Fatal("CreateGroup ", err)
		}

		tw := NewTableWriter(
			Col("ID"),
			Col("Name"),
			Col("UserID"),
			Col("Parent"),
			Col("AssetCount"),
			Col("AssetSize"),
			Col("CreatedTime"),
		)

		for _, group := range rsp.AssetGroups {
			// asset := rets.AssetOverviews[w]
			m := map[string]interface{}{
				"ID":          group.ID,
				"Name":        group.Name,
				"UserID":      group.UserID,
				"Parent":      group.Parent,
				"AssetCount":  group.AssetCount,
				"AssetSize":   group.AssetSize,
				"CreatedTime": group.CreatedTime,
			}

			tw.Write(m)
		}

		tw.Flush(os.Stdout)
		fmt.Println("Total ", rsp.Total)

	},
}

var deleteGroupCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a group",
	Run: func(cmd *cobra.Command, args []string) {
		parentID, _ := cmd.Flags().GetInt("groupID")

		titanURL, apiKey, err := getTitanURLAndAPIKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, err := storage.NewStorage(&storage.Config{TitanURL: titanURL, APIKey: apiKey})
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}

		err = s.DeleteGroup(cmd.Context(), parentID)
		if err != nil {
			log.Fatal("DeleteGroup ", err)
		}
	},
}

func init() {
	uploadCmd.Flags().Bool("make-car", true, "make car")

	listFilesCmd.Flags().Int("group-id", 0, "the group id")
	listFilesCmd.Flags().Int("page-size", 20, "Limit the page size")
	listFilesCmd.Flags().Int("page", 1, "the page")

	getFileCmd.Flags().String("cid", "", "the cid of file")
	getFileCmd.Flags().String("out", "", "the path to save file")

	createGroupCmd.Flags().StringP("name", "n", "", "special the name for group")
	createGroupCmd.Flags().Int("parentID", 0, "special the parent for group")

	listGroupCmd.Flags().Int("parentID", 0, "special the parent for group")
	listGroupCmd.Flags().IntP("start", "s", 0, "special the start for list")
	listGroupCmd.Flags().IntP("end", "e", 20, "special the end for list")

	deleteGroupCmd.Flags().Int("groupID", 0, "special the group id")
}

func Execute() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(listFilesCmd)
	rootCmd.AddCommand(getFileCmd)
	rootCmd.AddCommand(deleteFileCmd)
	rootCmd.AddCommand(getURLCmd)

	groupCmd.AddCommand(createGroupCmd)
	groupCmd.AddCommand(listGroupCmd)
	groupCmd.AddCommand(deleteGroupCmd)
	rootCmd.AddCommand(groupCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
