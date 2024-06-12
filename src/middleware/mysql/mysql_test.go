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

package mysql

import (
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestInitMySql(t *testing.T) {
	defer func() {
		os.RemoveAll("logs")
		os.RemoveAll("logs-0.db")
		os.RemoveAll("logs-0.db-shm")
		os.RemoveAll("logs-0.db-wal")
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()

	InitMySql()

	stmt, err := mysqlDBLog.Prepare("replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values(?,?,?,?,?,?,?,?,?,?,?)")
	result, err := stmt.Exec("1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11")
	if nil != err {
		t.Fatal(err)
	}

	fmt.Println(result.RowsAffected())
	fmt.Println(result.LastInsertId())

	stmt, err = mysqlDBLog.Prepare("replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values(?,?,?,?,?,?,?,?,?,?,?)")
	result, err = stmt.Exec("1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11")
	if nil != err {
		t.Fatal(err)
	}

	fmt.Println(result.RowsAffected())
	fmt.Println(result.LastInsertId())
}

func TestSelectLogs(t *testing.T) {
	defer func() {
		os.RemoveAll("logs")
		os.RemoveAll("logs-0.db")
		os.RemoveAll("logs-0.db-shm")
		os.RemoveAll("logs-0.db-wal")
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()

	InitMySql()
	logs := SelectLogs(0, 100000000, nil)

	for _, log := range logs {
		fmt.Printf("%d %s %s\n", log.BlockNumber, log.BlockHash.String(), log.TxHash.Hex())
	}
}

func TestGroupIndex(t *testing.T) {
	defer func() {
		os.RemoveAll("logs")
		os.RemoveAll("logs-0.db")
		os.RemoveAll("logs-0.db-shm")
		os.RemoveAll("logs-0.db-wal")
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()

	InitMySql()
	group := types.Group{}
	group.GroupHeight = 1
	group.Id = []byte{1, 2, 3, 4, 5, 6}
	group.Header = &types.GroupHeader{}
	group.Header.WorkHeight = 100
	group.Header.DismissHeight = math.MaxUint64
	InsertGroup(&group)

	group1 := types.Group{}
	group1.GroupHeight = 2
	group1.Id = []byte{10, 20, 30, 40}
	group1.Header = &types.GroupHeader{}
	group1.Header.WorkHeight = 400
	group1.Header.DismissHeight = 20000
	InsertGroup(&group1)

	fmt.Println(CountGroups())
	fmt.Println(SelectGroups(100))
	fmt.Println(SelectGroups(500))
	fmt.Println(SelectGroups(10))
	fmt.Println(SelectGroups(10001))
}
