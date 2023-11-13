package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	storage "github.com/Filecoin-Titan/titan-storage-sdk"
	"github.com/Filecoin-Titan/titan/lib/tablewriter"
	"github.com/spf13/cobra"
)

func getTitanURLAndApiKeyFromEnv() (string, string, error) {
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

var rootCmd = &cobra.Command{}

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
	Example: "upload /path/to/my/file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("Please specify the name of the file to be uploaded")
		}

		filePath := args[0]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Fatalf("File %s does not exist.", filePath)
		}

		titanURL, apiKey, err := getTitanURLAndApiKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, close, err := storage.NewStorage(titanURL, apiKey)
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}
		defer close()

		startTime := time.Now()
		progress := func(doneSize int64, totalSize int64) {
			log.Printf("total size:%d bytes, dong %d bytes\n", totalSize, doneSize)
		}

		cid, err := s.UploadFilesWithPath(context.Background(), filePath, progress)
		if err != nil {
			log.Fatal("UploadFilesWithPath ", err)
		}

		log.Printf("upload file %s cid %s success cost %dms\n", filePath, cid.String(), time.Since(startTime)/time.Millisecond)
	},
}

var listFilesCmd = &cobra.Command{
	Use:     "list",
	Short:   "list files",
	Example: "list --limit=20 --offset=0",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		offst, _ := cmd.Flags().GetInt("offset")

		if limit == 0 {
			log.Fatal("please set --limit flag")
		}

		titanURL, apiKey, err := getTitanURLAndApiKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, close, err := storage.NewStorage(titanURL, apiKey)
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}
		defer close()

		rets, err := s.ListUserAssets(context.Background(), limit, offst)
		if err != nil {
			log.Fatal("UploadFilesWithPath ", err)
		}

		tw := tablewriter.New(
			tablewriter.Col("CID"),
			tablewriter.Col("Name"),
			tablewriter.Col("Size"),
			tablewriter.Col("CreatedTime"),
			tablewriter.Col("Expiration"),
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

		titanURL, apiKey, err := getTitanURLAndApiKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, close, err := storage.NewStorage(titanURL, apiKey)
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}
		defer close()

		reader, err := s.GetFileWithCid(context.Background(), cid)
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

		log.Printf("get %s success cost %dms", cid, time.Since(startTime)/time.Millisecond)

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

		titanURL, apiKey, err := getTitanURLAndApiKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, close, err := storage.NewStorage(titanURL, apiKey)
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}
		defer close()

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

		titanURL, apiKey, err := getTitanURLAndApiKeyFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		s, close, err := storage.NewStorage(titanURL, apiKey)
		if err != nil {
			log.Fatal("NewStorage error ", err)
		}
		defer close()

		url, err := s.GetURL(context.Background(), rootCID)
		if err != nil {
			log.Fatal("UploadFilesWithPath ", err)
		}

		log.Println(url)
	},
}

func Execute() {
	listFilesCmd.Flags().IntP("limit", "l", 20, "Limit the length of the list")
	listFilesCmd.Flags().IntP("offset", "o", 0, "Limit the length of the list")
	getFileCmd.Flags().String("cid", "", "the cid of file")
	getFileCmd.Flags().String("out", "", "the path to save file")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(listFilesCmd)
	rootCmd.AddCommand(getFileCmd)
	rootCmd.AddCommand(deleteFileCmd)
	rootCmd.AddCommand(getURLCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
