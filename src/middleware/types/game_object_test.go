package types

import (
	"testing"
	"math/big"
	"x/src/storage/rlp"
	"fmt"
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
	ft := &NFT{ID: "sword1", Name: "yitai", Symbol: "yt"}

	data, err := rlp.EncodeToBytes(ft)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(data)

	nft := &NFT{}
	err = rlp.DecodeBytes(data, nft)
	fmt.Println(nft.Name)
}

func TestGameData_SetNFT(t *testing.T) {
	gameData := &GameData{}
	nftMap := &NFTMap{}
	nft := &NFT{ID: "sword1", Name: "yitai", Symbol: "yt"}
	nftMap.SetNFT("sword1", nft)
	gameData.SetNFT("test1", nftMap)

	fmt.Println(gameData.GetNFTMaps("test1").GetNFT("sword1").Name)

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
	fmt.Println(nftMap.GetAllNFT("test1"))
	nft = nftMap.GetNFT("sword1")
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

type Student struct {
	Name string
	Sex  string
}
