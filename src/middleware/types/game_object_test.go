// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"com.tuntun.rocket/node/src/storage/rlp"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
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
	ft := &NFT{ID: "sword1", Name: "yitai", Symbol: "yt", CreateTime: "1571134085856098"}

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

func TestNFTSet_ToJSONString(t *testing.T) {
	nftSet := &NFTSet{
		TotalSupply: 12,
	}
	fmt.Println(nftSet.ToJSONString())

	nftSet.TotalSupply++
	fmt.Println(nftSet.ToJSONString())

}
