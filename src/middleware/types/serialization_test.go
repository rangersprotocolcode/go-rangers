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
	"encoding/json"
	"fmt"
	"testing"

	middleware_pb "com.tuntun.rocket/node/src/middleware/pb"
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

	nft.SetData("key1", "data1")
	nft.SetData("key2", "data2")

	fmt.Printf("%s\n", nft.ToJSONString())

}
