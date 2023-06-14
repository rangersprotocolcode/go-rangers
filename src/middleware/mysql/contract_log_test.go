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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package mysql

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"encoding/json"
	"os"
	"testing"
)

func TestInsertLogs(t *testing.T) {
	defer func() {
		os.RemoveAll("logs-0.db")
		os.RemoveAll("logs-0.db-shm")
		os.RemoveAll("logs-0.db-wal")
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()

	InitMySql()

	data := `{"blockHash":"0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5","blockNumber":"0x2604220","cumulativeGasUsed":0,"from":"0x41af145b13e76920799252a4affcfb63a7bebd11","gasUsed":"0x0","logs":[{"address":"0x021f5327280b68f382171056aa34dce310dc6c1d","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x00000000000000000000000041af145b13e76920799252a4affcfb63a7bebd11","0x0000000000000000000000009c1cbfe5328dfb1733d59a7652d0a49228c7e12c"],"data":"0x00000000000000000000000000000000000000000000000011d181b8c139349d","blockNumber":"0x2604220","transactionHash":"0x97feedc228bc4d8774da0fbda37ebf4437019b9f115c624f7a6d3b1903709e2e","transactionIndex":"0x1","blockHash":"0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5","logIndex":"0x99","removed":false},{"address":"0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c","topics":["0xad43c264a23265278792c8a50133a738a0762c733d8ee602fb643eead18c2a85"],"data":"0x000000000000000000000000000000000000000000000000000046c2cabae9b5000000000000000000000000021f5327280b68f382171056aa34dce310dc6c1d0000000000000000000000001ca05267f79ad1e496956922f257c2ff6c8e189200000000000000000000000000000000000000000000000000000000000000e000000000000000000000000041af145b13e76920799252a4affcfb63a7bebd1100000000000000000000000041af145b13e76920799252a4affcfb63a7bebd1100000000000000000000000000000000000000000000000011d181b8c139349d0000000000000000000000000000000000000000000000000000000000000008486f6c7952656b69000000000000000000000000000000000000000000000000","blockNumber":"0x2604220","transactionHash":"0x97feedc228bc4d8774da0fbda37ebf4437019b9f115c624f7a6d3b1903709e2e","transactionIndex":"0x1","blockHash":"0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5","logIndex":"0x9a","removed":false}],"logsBloom":"0x00000002000000000000000000000000008000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008001000000000000000200000000000000000000010000000000000000000000000000000000000000200000000000010000000000000000000000080000008000000000000000000000000000000000000000200000000000000000001000000000040000000000000000000000000000000000040000002000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000001000000000000000","status":1,"to":"0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c","transactionHash":"0x97feedc228bc4d8774da0fbda37ebf4437019b9f115c624f7a6d3b1903709e2e","transactionIndex":0}`
	var rec types.Receipt
	if err := json.Unmarshal([]byte(data), &rec); err != nil {
		t.Fatal(err)
	}

	//var list types.Receipts
	list := make([]*types.Receipt, 1)
	list[0] = &rec

	InsertLogs(100000, list, common.HexToHash("0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5"))

	contract := common.HexToAddress("0x021f5327280b68f382171056aa34dce310dc6c1d")
	contractAddresses := make([]common.Address, 1)
	contractAddresses[0] = contract
	logs := SelectLogsByHash(common.HexToHash("0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5"), contractAddresses)
	if 1 != len(logs) {
		t.Fatal("fail to select 0x021f5327280b68f382171056aa34dce310dc6c1d")
	}

	contract2 := common.HexToAddress("0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c")
	contractAddresses2 := make([]common.Address, 1)
	contractAddresses2[0] = contract2
	logs2 := SelectLogsByHash(common.HexToHash("0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5"), contractAddresses2)
	if 1 != len(logs2) {
		t.Fatal("fail to select 0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c")
	}

	contract3 := common.HexToAddress("0xff")
	contractAddresses3 := make([]common.Address, 1)
	contractAddresses3[0] = contract3
	logs3 := SelectLogsByHash(common.HexToHash("0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5"), contractAddresses3)
	if 0 != len(logs3) {
		t.Fatal("fail to select 0xff")
	}
}

func TestInsertLogs2(t *testing.T) {
	defer func() {
		os.RemoveAll("logs-0.db")
		os.RemoveAll("logs-0.db-shm")
		os.RemoveAll("logs-0.db-wal")
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()

	InitMySql()

	//var list types.Receipts
	list := make([]*types.Receipt, 0)

	InsertLogs(100000, list, common.HexToHash("0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5"))

	contract3 := common.HexToAddress("0xff")
	contractAddresses3 := make([]common.Address, 1)
	contractAddresses3[0] = contract3
	logs3 := SelectLogsByHash(common.HexToHash("0x1ec9d7fb2bbfcae65110c3cc3e7a9da2a1c8c90683897efc755787ccea638ff5"), contractAddresses3)
	if 0 != len(logs3) {
		t.Fatal("fail to select 0xff")
	}
}
