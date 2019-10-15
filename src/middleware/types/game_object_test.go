package types

import (
	"testing"
	"math/big"
	"x/src/storage/rlp"
	"fmt"
	"encoding/json"
	"strings"
)

func TestFT_EncodeRLP(t *testing.T) {
	ft := FT{
		ID:      "test1",
		Balance: big.NewInt(100),
	}

	data, err := rlp.EncodeToBytes(ft)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(data)

	ftMap := []FT{}
	ftMap = append(ftMap, ft)
	data, err = rlp.EncodeToBytes(ftMap)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(data)

}

func TestNFT_EncodeRLP(t *testing.T) {
	ft := &NFT{ID: "sword1", Name: "yitai", Symbol: "yt",CreateTime:"1571134085856098"}

	data, err := rlp.EncodeToBytes(ft)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(data)

	nft := &NFT{}
	err = rlp.DecodeBytes(data, nft)
	fmt.Println(nft.Name)
	fmt.Println(nft.CreateTime)
}

func TestGameData_SetNFT(t *testing.T) {
	gameData := &GameData{}
	nftMap := &NFTMap{}
	nft := &NFT{SetID: "g1", ID: "sword1", Name: "yitai", Symbol: "yt"}
	nftMap.SetNFT(nft)
	gameData.SetNFTMaps("test1", nftMap)

	fmt.Println(gameData.GetNFTMaps("test1").GetNFT("g1", "sword1").Name)

	data, err := rlp.EncodeToBytes(gameData)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(data)

	g := &GameData{}
	err = rlp.DecodeBytes(data, g)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	nftMap = g.GetNFTMaps("test1")
	fmt.Println(nftMap)
	fmt.Println(nftMap.GetAllNFT())
	nft = nftMap.GetNFT("g1", "sword1")
	fmt.Println(nft.Name)
}

func Test_RLP(t *testing.T) {
	s := Student{Name: "icattlecoder", Sex: "male"}

	data, err := rlp.EncodeToBytes(s)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	fmt.Println(data)
}

func Test_JSON(t *testing.T) {
	s := Student{Name: "icattlecoder", Sex: "male"}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	var st Student
	err = json.Unmarshal(data, &st)

	var stp *Student
	stp = &st
	fmt.Println(stp.Name)

	ftName := "abc-123"
	ftInfo := strings.Split(ftName, "-")
	fmt.Println(ftInfo[0])
	fmt.Println(ftInfo[1])
	fmt.Println(len(ftInfo))
}

type Student struct {
	Name string
	Sex  string
}
