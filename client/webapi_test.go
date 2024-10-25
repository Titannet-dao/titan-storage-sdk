package client

import (
	"context"
	"fmt"
	"testing"
)

const url = "http://120.79.221.36:10089"
const key = "M1Lv3offPmrheDHKQd+z4QThBqRMjICHhZV19cPlctgFoYe73Bfz7klQtRZrTC2Y"

func TestGetVip(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	// req := CreateAssetReq{}
	vipInfo, err := webserver.GetVipInfo(context.Background())
	if err != nil {
		fmt.Println("err ", err)
		return
	}
	fmt.Println("vip info ", *vipInfo)
}

func TestListAreaIDs(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	// req := CreateAssetReq{}
	webserver.ListAreaIDs(context.Background())
}
func TestCreateAsset(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	ap := AssetProperty{
		AssetCID:  "bafkreibcyimvlzbgwudx3oict7iufabktjherbhkopwaxzobukpc2bricq",
		AssetName: "README",
		AssetSize: 1830,
		AssetType: "file",
		GroupID:   0,
	}
	req := CreateAssetReq{AssetProperty: ap}
	rsp, err := webserver.CreateAsset(context.Background(), &req)
	if err != nil {
		fmt.Println("err ", err.Error())
		return
	}

	if rsp.IsAlreadyExist {
		fmt.Println("asset already exist")
		return
	}

	for _, ep := range rsp.Endpoints {
		fmt.Println("endpoint ", *ep)
	}
}
func TestDeleteAsset(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	webserver.DeleteAsset(context.Background(), "1052441607@qq.com", "bafkreifnpu6du62vascvvnpfxgbonagqdkxjs53v2q4g5vne6nbirmwpd")
}

func TestShareAsset(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	result, err := webserver.ShareAsset(context.Background(), "1052441607@qq.com", "", "bafkreifnpu6du62vascvvnpfxgbonagqdkxjs53v2q4g5vne6nbirmwpdu")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("result ", *result)
}

func TestCreateGroup(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	result, err := webserver.CreateGroup(context.Background(), "test", 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("result ", *result)
}

func TestListGroups(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	result, err := webserver.ListGroups(context.Background(), 0, 100, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, group := range result.AssetGroups {
		fmt.Println("group ", *group)
	}
}

func TestDeleteGroup(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	err := webserver.DeleteGroup(context.Background(), "1052441607@qq.com", 51)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("delete success")
}

func TestListAssets(t *testing.T) {
	webserver := NewWebserver(url, key, "")
	rsp, err := webserver.ListAssets(context.Background(), 0, 20, 1, "", 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("total", rsp.Total)

	for _, assetOverview := range rsp.AssetOverviews {
		fmt.Printf("assetOverview assetName %s  AssetRecord %#v\n", assetOverview.UserAssetDetail.AssetName, *assetOverview.AssetRecord)
	}
}
