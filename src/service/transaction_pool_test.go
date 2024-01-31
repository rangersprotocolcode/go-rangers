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

package service

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"github.com/gogf/gf/container/gmap"
	"math/big"
	"os"
	"testing"
)

func TestRequestId(t *testing.T) {
	s := "0x41ed2348bb544cb9e54ed6405e930ac7164e57f4cc59f6fe33f0ba84452d9bc550d31be232410a890618f3b628e2ee5a6e679581c6efed3d31ad07d4dd2398e000"
	sign := common.HexStringToSign(s)
	fmt.Println(sign.Bytes())
	fmt.Println(sign.GetR())
	fmt.Println(sign.GetS())
	fmt.Println(sign.GetHexString())
}

func TestSlice(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6}
	fmt.Println(data)

	fmt.Println(data[2:])
}

func TestGMap(t *testing.T) {
	listMap := gmap.NewListMap(true)

	listMap.Set("1", "a")
	listMap.Set("3", "b")
	listMap.Set("2", "c")
	listMap.Set("5", "d")
	listMap.Set("4", "e")

	fmt.Println(listMap.Size())
	fmt.Println(listMap.Keys())
	fmt.Println(listMap.Values())
}

func preTest() {
	common.Init(0, "0.ini", "dev")
	middleware.InitMiddleware()
	InitService()

	common.SetBlockHeight(10000)
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	middleware.AccountDBManagerInstance.SetLatestStateDB(state, make(map[string]uint64), 10000)
}

// empty pack
func TestTxPool_PackForCast(t *testing.T) {
	defer func() {
		Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()

	preTest()

	txs := txpoolInstance.PackForCast(10000)
	if 0 != len(txs) {
		t.Fatal("no txs error")
	}
}

// same address for 2 txs
func TestTxPool_PackForCast1(t *testing.T) {
	defer func() {
		Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()

	preTest()

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}

	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000)
	if 1 != len(txs) {
		t.Fatal("no txs error")
	}

	tx1 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xbb"),
		Nonce:  1,
	}
	txpoolInstance.AddTransaction(tx1)
	txs1 := txpoolInstance.PackForCast(10000)
	if 2 != len(txs1) {
		t.Fatal("no txs error")
	}
}

// same address for 2 txs with same nonce
// still pack 2 txs
func TestTxPool_PackForCast2(t *testing.T) {
	defer func() {
		Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()

	preTest()

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}

	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000)
	if 1 != len(txs) {
		t.Fatal("no txs error")
	}

	tx1 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xbb"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx1)
	txs1 := txpoolInstance.PackForCast(10000)
	if 2 != len(txs1) {
		t.Fatal("no txs error")
	}
}

// same address for 2 txs with same nonce
// but different type
// still pack 2 txs
func TestTxPool_PackForCast3(t *testing.T) {
	defer func() {
		Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()

	preTest()

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}

	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000)
	if 1 != len(txs) {
		t.Fatal("no txs error")
	}

	tx1 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xbb"),
		Nonce:  0,
		Type:   1,
	}
	txpoolInstance.AddTransaction(tx1)
	txs1 := txpoolInstance.PackForCast(10000)
	if 2 != len(txs1) {
		t.Fatal("no txs error")
	}
}

// 2 addresses
// A has 2 tx
// B has 1 tx
// pack 3 txs
func TestTxPool_PackForCast4(t *testing.T) {
	defer func() {
		Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()

	preTest()

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000)
	if 1 != len(txs) {
		t.Fatal("no txs error")
	}

	tx1 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx1)
	txs1 := txpoolInstance.PackForCast(10000)
	if 2 != len(txs1) {
		t.Fatal("no txs error")
	}

	tx2 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xcc"),
		Nonce:  0,
		Type:   1,
	}
	txpoolInstance.AddTransaction(tx2)
	txs2 := txpoolInstance.PackForCast(10000)
	if 3 != len(txs2) {
		t.Fatal("no txs error")
	}
}

// different addr with 0 nonce
// pack txCountPerBlock txs
func TestTxPool_PackForCast5(t *testing.T) {
	defer func() {
		Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	// normal add
	i := int64(1)
	for ; i < txCountPerBlock+1; i++ {
		str := big.NewInt(i + 100).String()
		tx := &types.Transaction{
			Source: str,
			Hash:   common.HexToHash(str),
			Nonce:  0,
		}
		flag, _ := txpoolInstance.AddTransaction(tx)

		if !flag {
			t.Fatalf("fail to add tx, i: %s, hash: %s", str, tx.Hash.String())
		}
		txs := txpoolInstance.PackForCast(10000)
		if i != int64(len(txs)) {
			t.Fatalf("txsize errof, i: %d", i)
		}
	}

	// oversize
	str := big.NewInt(i + 100).String()
	tx := &types.Transaction{
		Source: str,
		Hash:   common.HexToHash(str),
		Nonce:  0,
	}
	flag, _ := txpoolInstance.AddTransaction(tx)

	if !flag {
		t.Fatalf("fail to add tx, i: %s, hash: %s", str, tx.Hash.String())
	}
	txs := txpoolInstance.PackForCast(10000)
	if txCountPerBlock != int64(len(txs)) {
		t.Fatal("oversize")
	}
}
