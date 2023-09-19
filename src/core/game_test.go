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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"testing"
)

func TestSwtich(t *testing.T) {
	echo(1)
}

func echo(a int) {
	switch a {
	case 1:
		fmt.Println("1")
		return
	case 2:
		fmt.Println("2")
	case 3:
		fmt.Println("3")
	}
	fmt.Println("After case")
}

func TestGameExecutor_RunWrite(t *testing.T) {
	file := "tempTx" + strconv.Itoa(common.InstanceIndex)
	tempTxLDB, err := db.NewLDBDatabase(file, 10, 10)
	if err != nil {
		panic("newLDBDatabase fail, file=" + file + ", err=" + err.Error())
	}
	defer func() {
		fmt.Println("finished")
	}()
	tempTxLDB.Put([]byte("1234"), []byte("abcd"))
	tempTxLDB.Put([]byte("5678"), []byte("qwer"))
	tempTxLDB.Put([]byte("9012"), []byte("zxcv"))

	iter := tempTxLDB.NewIterator()
	for iter.Next() {
		fmt.Printf("%s, %s\n", iter.Key(), iter.Value())
		tempTxLDB.Delete([]byte("5678"))
	}

	value, _ := tempTxLDB.Get([]byte("5678"))
	fmt.Printf("%s", string(value))
	tempTxLDB.Close()

	os.RemoveAll("tempTx0")
}

func TestBase64(t *testing.T) {
	s := "OPl+7u9grA4AsQ9EVBiZRVFJeBv1e2gjxq3jSOfEDwQ="
	b, _ := base64.StdEncoding.DecodeString(s)
	fmt.Printf("%v\n", common.Bytes2Hex(b))
	fmt.Printf("%v\n", b)
}
