// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"encoding/json"
	"fmt"
	"testing"

	middleware_pb "com.tuntun.rangers/node/src/middleware/pb"
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

func TestMarshalTransactions(t *testing.T) {
	txs := make([]*Transaction, 2)
	//txs[0] = &Transaction{Source: "1234"}
	txs[1] = &Transaction{Source: "1234"}
	data, err := MarshalTransactions(txs)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(data)
}
