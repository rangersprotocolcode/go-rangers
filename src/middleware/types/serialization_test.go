package types

import (
	"testing"
	"x/src/middleware/pb"
	"encoding/json"
	"fmt"
)

func TestPbToBlockHeader(t *testing.T) {
	header := BlockHeader{}
	header.RequestIds = make(map[string]uint64)
	header.RequestIds["1"] = 1024

	pb := middleware_pb.BlockHeader{}
	pb.RequestIds, _ = json.Marshal(header.RequestIds)

	header2 := BlockHeader{}
	json.Unmarshal(pb.RequestIds, &header2.RequestIds)

	fmt.Println(header2.RequestIds["1"])

	fmt.Println(header2.RequestIds["2"] < 2)

}

func TestNFT(t *testing.T) {
	nft := NFT{SetID: "111", ID: "fdd", Name: "nftName", Symbol: "nftSymbol", Creator: "testman", CreateTime: "4644646546464", Owner: "abc",
		Renter: "dbd", Status: 0, Condition: 0, AppId: "0xdafawe"}

	dataValue := []string{"data1", "data2"}
	dataKey := []string{"key1", "key2"}
	nft.DataKey = dataKey
	nft.DataValue = dataValue

	fmt.Printf("%s\n", nft.ToJSONString())

}
