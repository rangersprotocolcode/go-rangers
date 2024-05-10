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
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"fmt"
	"github.com/gogf/gf/container/gmap"
	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestAddress(t *testing.T) {
	address1 := "0x1111111111111111111111111111111111111111"
	byte1 := common.FromHex(address1)
	fmt.Println(byte1)
}
func TestSign(t *testing.T) {
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

func TestNonceQueue(t *testing.T) {
	nonceList := new(nonceQueue)
	nonceList.Push(125)
	nonceList.Push(36)
	nonceList.Push(102)
	nonceList.Push(108)
	fmt.Println(nonceList)
	for nonceList.Len() > 0 {
		i := nonceList.Pop()
		fmt.Println(i)
	}
}

func TestLogLock(t *testing.T) {
	common.Init(0, "0.ini", "dev")
	middleware.InitMiddleware()
	lockDemo := middleware.NewLoglock("log")
	fmt.Println(time.Now().String())

	//utility.GetTime() sometime cost some time in test  case
	lockDemo.RLock("aaa")
	fmt.Println(time.Now().String())

	lockDemo.RUnlock("aaa")
	fmt.Println(time.Now().String())
}

func TestSortTx(t *testing.T) {
	common.Init(0, "0.ini", "dev")
	common.SetBlockHeight(10000)
	address1 := "0x1111111111111111111111111111111111111111"
	address2 := "0x2222222222222222222222222222222222222222"
	address3 := "0x3333333333333333333333333333333333333333"
	addressList := []string{address1, address2, address3}
	var nonce uint64 = 0
	var txList = new(types.Transactions)
	for ; nonce < 10; nonce++ {
		for _, address := range addressList {
			tx := types.Transaction{Type: 188, Source: address, Nonce: nonce}
			tx.Hash = tx.GenHash()
			*txList = append(*txList, &tx)
		}
	}
	sort.Sort(txList)
	for _, tx := range *txList {
		fmt.Printf("source:%s,nonce:%d,hash:%s\n", tx.Source, tx.Nonce, tx.Hash.String())
	}
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

// empty pack
// wrong nonce
func TestTxPool_PackForCast0(t *testing.T) {
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
		Nonce:  10,
	}

	txpoolInstance.AddTransaction(tx)
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

// 2 addresses
// A has 3 tx nonce 0,1,2
// B has 6 tx nonce 0,1,1,3,4,5
// pack 6 txs
func TestTxPool_PackForCast6(t *testing.T) {
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

	tx0 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa0"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx0)

	tx1 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa1"),
		Nonce:  1,
	}
	txpoolInstance.AddTransaction(tx1)

	tx2 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa2"),
		Nonce:  2,
	}
	txpoolInstance.AddTransaction(tx2)

	tx10 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb00"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx10)

	tx11 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb11"),
		Nonce:  1,
	}
	txpoolInstance.AddTransaction(tx11)

	tx12 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb10"),
		Nonce:  1,
	}
	txpoolInstance.AddTransaction(tx12)

	tx13 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb30"),
		Nonce:  3,
	}
	txpoolInstance.AddTransaction(tx13)

	tx14 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb40"),
		Nonce:  4,
	}
	txpoolInstance.AddTransaction(tx14)

	tx15 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb50"),
		Nonce:  5,
	}
	txpoolInstance.AddTransaction(tx15)

	txList := txpoolInstance.PackForCast(10000)
	if 6 != len(txList) {
		t.Fatal("packed tx count error")
	}
	assert.Equal(t, txList[0].Hash.String(), tx10.Hash.String())
	assert.Equal(t, txList[1].Hash.String(), tx11.Hash.String())
	assert.Equal(t, txList[2].Hash.String(), tx12.Hash.String())
	assert.Equal(t, txList[3].Hash.String(), tx0.Hash.String())
	assert.Equal(t, txList[4].Hash.String(), tx1.Hash.String())
	assert.Equal(t, txList[5].Hash.String(), tx2.Hash.String())
}

func initTxPool() *TxPool {
	pool := &TxPool{}
	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.evictedTxs, _ = lru.New(txCacheSize)

	executed, err := db.NewLDBDatabase(txDataBasePrefix, 256, 512)
	if err != nil {
		txPoolLogger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()

	pool.pending = make(map[string]*txList, 0)
	pool.queue = make(map[string]*txList, 0)
	pool.lock = middleware.NewLoglock("txPool")
	pool.annualRingMap = sync.Map{}
	pool.lifeCycleTicker = time.NewTicker(txCycleInterval)
	go pool.loop()
	return pool
}

var testTxPoolInstance *TxPool

//1 address
// add nonce 0
// add nonce 1
// add nonce 2
//remove nonce 1
func TestTxPool_PendingSingleAddress1(t *testing.T) {
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

	//use TxPool instead of TransactionPool
	testTxPoolInstance = initTxPool()
	preTest()
	fmt.Println("Init nonce.Addr1:0")

	address1 := "0x2f4F09b722a6e5b77bE17c9A99c785Fa70111111"
	tx1 := &types.Transaction{
		Source: address1,
		Nonce:  testTxPoolInstance.GetPendingNonce(address1),
	}
	tx1.Hash = tx1.GenHash()

	testTxPoolInstance.AddTransaction(tx1)
	pendingList := testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after add tx1(nonce0):\n")
	fmt.Printf("pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 1)
	assert.Equal(t, pendingList[0], uint64(0))

	gotTx1 := testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)

	tx2 := &types.Transaction{
		Source: address1,
		Nonce:  testTxPoolInstance.GetPendingNonce(address1),
	}
	tx2.Hash = tx2.GenHash()
	testTxPoolInstance.AddTransaction(tx2)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after add tx2(nonce1):\n")
	fmt.Printf("pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 2)
	assert.Equal(t, pendingList[0], uint64(0))
	assert.Equal(t, pendingList[1], uint64(1))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)

	gotTx2 := testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)

	tx3 := &types.Transaction{
		Source: address1,
		Nonce:  testTxPoolInstance.GetPendingNonce(address1),
	}
	tx3.Hash = tx3.GenHash()
	testTxPoolInstance.AddTransaction(tx3)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after add tx3(nonce2):\n")
	fmt.Printf("pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 3)
	assert.Equal(t, pendingList[0], uint64(0))
	assert.Equal(t, pendingList[1], uint64(1))
	assert.Equal(t, pendingList[2], uint64(2))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)

	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)

	gotTx3 := testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)

	txs := []*types.Transaction{tx2}
	testTxPoolInstance.remove(txs, nil)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after remove tx2(nonce1):\n")
	fmt.Printf("pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 1)
	assert.Equal(t, pendingList[0], uint64(2))
	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)

	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 == nil)

	gotTx3 = testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)
}

//2 address
// address1 add nonce 0
// address1 add nonce 1
// address2 add nonce 0
// address2 add nonce 1
// address1 add nonce 2
// address1 remove nonce 1
func TestTxPool_PendingMultiAddress(t *testing.T) {
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

	//use TxPool instead of TransactionPool
	testTxPoolInstance = initTxPool()
	preTest()
	fmt.Println("Init nonce.Addr1:0,Addr2:0")

	address1 := "0x2f4F09b722a6e5b77bE17c9A99c785Fa70111111"
	address2 := "0x2f4F09b722a6e5b77bE17c9A99c785Fa70222222"
	tx10 := &types.Transaction{
		Source: address1,
		Nonce:  testTxPoolInstance.GetPendingNonce(address1),
	}
	tx10.Hash = tx10.GenHash()
	testTxPoolInstance.AddTransaction(tx10)
	pendingList := testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after add tx10(addr1 nonce0):\n")
	fmt.Printf("addr1 pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 1)
	assert.Equal(t, pendingList[0], uint64(0))
	gotTx1 := testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx1 != nil)

	tx11 := &types.Transaction{
		Source: address1,
		Nonce:  testTxPoolInstance.GetPendingNonce(address1),
	}
	tx11.Hash = tx11.GenHash()
	testTxPoolInstance.AddTransaction(tx11)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after add tx11(addr1 nonce1):\n")
	fmt.Printf("addr1 pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 2)
	assert.Equal(t, pendingList[0], uint64(0))
	assert.Equal(t, pendingList[1], uint64(1))
	gotTx1 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 := testTxPoolInstance.received.get(tx11.Hash)
	assert.Equal(t, true, gotTx2 != nil)

	tx20 := &types.Transaction{
		Source: address2,
		Nonce:  testTxPoolInstance.GetPendingNonce(address2),
	}
	tx20.Hash = tx20.GenHash()
	testTxPoolInstance.AddTransaction(tx20)
	pendingList = testTxPoolInstance.GetPendingList(address2)
	fmt.Printf("after add tx20(addr2 nonce0):\n")
	fmt.Printf("addr2 pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 1)
	assert.Equal(t, pendingList[0], uint64(0))
	gotTx1 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx11.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 := testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx3 != nil)

	tx21 := &types.Transaction{
		Source: address2,
		Nonce:  testTxPoolInstance.GetPendingNonce(address2),
	}
	tx21.Hash = tx21.GenHash()
	testTxPoolInstance.AddTransaction(tx21)
	pendingList = testTxPoolInstance.GetPendingList(address2)
	fmt.Printf("after add tx21(addr2 nonce1):\n")
	fmt.Printf("addr2 pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 2)
	assert.Equal(t, pendingList[0], uint64(0))
	assert.Equal(t, pendingList[1], uint64(1))
	gotTx1 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx11.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 := testTxPoolInstance.received.get(tx21.Hash)
	assert.Equal(t, true, gotTx4 != nil)

	tx12 := &types.Transaction{
		Source: address1,
		Nonce:  testTxPoolInstance.GetPendingNonce(address1),
	}
	tx12.Hash = tx12.GenHash()
	testTxPoolInstance.AddTransaction(tx12)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after add tx12(addr1 nonce2):\n")
	fmt.Printf("addr1 pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 3)
	assert.Equal(t, pendingList[0], uint64(0))
	assert.Equal(t, pendingList[1], uint64(1))
	assert.Equal(t, pendingList[2], uint64(2))
	gotTx1 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx11.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx12.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 = testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx4 != nil)
	gotTx5 := testTxPoolInstance.received.get(tx21.Hash)
	assert.Equal(t, true, gotTx5 != nil)

	//txs := []*types.Transaction{tx11}
	//testTxPoolInstance.remove(txs, nil)

	evictedTxs := []common.Hash{tx11.Hash}
	testTxPoolInstance.remove(nil, evictedTxs)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	fmt.Printf("after remove tx11(addr1 nonce1):\n")
	fmt.Printf("addr1 pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 1)
	assert.Equal(t, pendingList[0], uint64(2))
	gotTx1 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx11.Hash)
	assert.Equal(t, true, gotTx2 == nil)
	gotTx3 = testTxPoolInstance.received.get(tx12.Hash)
	assert.Equal(t, true, gotTx3 != nil)

	pendingList = testTxPoolInstance.GetPendingList(address2)
	fmt.Printf("addr2 pending:%v\n", pendingList)
	assert.Equal(t, len(pendingList), 2)
	assert.Equal(t, pendingList[0], uint64(0))
	assert.Equal(t, pendingList[1], uint64(1))
	gotTx1 = testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx21.Hash)
	assert.Equal(t, true, gotTx2 != nil)

}

//1 address init nonce 103
// add nonce 106
// add nonce 118
// add nonce 108
// add nonce 103
// add nonce 104
// add nonce 105
// add nonce 188
// add nonce 300
//remove nonce 256
func TestTxPool_QueueSingleAddress(t *testing.T) {
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

	//use TxPool instead of TransactionPool
	testTxPoolInstance = initTxPool()
	preTest()
	address1 := "0x2f4F09b722a6e5b77bE17c9A99c785Fa70111111"
	stateDB := middleware.AccountDBManagerInstance.GetLatestStateDB()
	stateDB.SetNonce(common.HexToAddress(address1), 103)
	fmt.Println("Init nonce.Addr1:103")

	tx1 := &types.Transaction{
		Source: address1,
		Nonce:  106,
	}
	tx1.Hash = tx1.GenHash()

	testTxPoolInstance.AddTransaction(tx1)
	pendingList := testTxPoolInstance.GetPendingList(address1)
	queueList := testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx1(nonce106):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 0)
	assert.Equal(t, len(queueList), 1)
	assert.Equal(t, queueList[0], uint64(106))
	gotTx1 := testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)

	tx2 := &types.Transaction{
		Source: address1,
		Nonce:  118,
	}
	tx2.Hash = tx2.GenHash()
	testTxPoolInstance.AddTransaction(tx2)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx2(nonce118):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 0)
	assert.Equal(t, len(queueList), 2)
	assert.Equal(t, queueList[0], uint64(106))
	assert.Equal(t, queueList[1], uint64(118))
	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 := testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)

	tx3 := &types.Transaction{
		Source: address1,
		Nonce:  108,
	}
	tx3.Hash = tx3.GenHash()
	testTxPoolInstance.AddTransaction(tx3)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx3(nonce108):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 0)
	assert.Equal(t, len(queueList), 3)
	assert.Equal(t, queueList[0], uint64(106))
	assert.Equal(t, queueList[1], uint64(108))
	assert.Equal(t, queueList[2], uint64(118))
	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 := testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)

	tx4 := &types.Transaction{
		Source: address1,
		Nonce:  103,
	}
	tx4.Hash = tx4.GenHash()
	testTxPoolInstance.AddTransaction(tx4)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx4(nonce103):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 1)
	assert.Equal(t, pendingList[0], uint64(103))

	assert.Equal(t, len(queueList), 3)
	assert.Equal(t, queueList[0], uint64(106))
	assert.Equal(t, queueList[1], uint64(108))
	assert.Equal(t, queueList[2], uint64(118))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 := testTxPoolInstance.received.get(tx4.Hash)
	assert.Equal(t, true, gotTx4 != nil)

	tx5 := &types.Transaction{
		Source: address1,
		Nonce:  104,
	}
	tx5.Hash = tx5.GenHash()
	testTxPoolInstance.AddTransaction(tx5)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx5(nonce104):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 2)
	assert.Equal(t, pendingList[0], uint64(103))
	assert.Equal(t, pendingList[1], uint64(104))

	assert.Equal(t, len(queueList), 3)
	assert.Equal(t, queueList[0], uint64(106))
	assert.Equal(t, queueList[1], uint64(108))
	assert.Equal(t, queueList[2], uint64(118))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 = testTxPoolInstance.received.get(tx4.Hash)
	assert.Equal(t, true, gotTx4 != nil)
	gotTx5 := testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx5 != nil)

	tx6 := &types.Transaction{
		Source: address1,
		Nonce:  105,
	}
	tx6.Hash = tx6.GenHash()
	testTxPoolInstance.AddTransaction(tx6)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx6(nonce105):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 4)
	assert.Equal(t, pendingList[0], uint64(103))
	assert.Equal(t, pendingList[1], uint64(104))
	assert.Equal(t, pendingList[2], uint64(105))
	assert.Equal(t, pendingList[3], uint64(106))

	assert.Equal(t, len(queueList), 2)
	assert.Equal(t, queueList[0], uint64(108))
	assert.Equal(t, queueList[1], uint64(118))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 = testTxPoolInstance.received.get(tx4.Hash)
	assert.Equal(t, true, gotTx4 != nil)
	gotTx5 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx5 != nil)
	gotTx6 := testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx6 != nil)

	tx7 := &types.Transaction{
		Source: address1,
		Nonce:  188,
	}
	tx7.Hash = tx7.GenHash()
	testTxPoolInstance.AddTransaction(tx7)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx7(nonce188):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 4)
	assert.Equal(t, pendingList[0], uint64(103))
	assert.Equal(t, pendingList[1], uint64(104))
	assert.Equal(t, pendingList[2], uint64(105))
	assert.Equal(t, pendingList[3], uint64(106))

	assert.Equal(t, len(queueList), 3)
	assert.Equal(t, queueList[0], uint64(108))
	assert.Equal(t, queueList[1], uint64(118))
	assert.Equal(t, queueList[2], uint64(188))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 = testTxPoolInstance.received.get(tx4.Hash)
	assert.Equal(t, true, gotTx4 != nil)
	gotTx5 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx5 != nil)
	gotTx6 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx6 != nil)
	gotTx7 := testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx7 != nil)

	tx8 := &types.Transaction{
		Source: address1,
		Nonce:  300,
	}
	tx8.Hash = tx8.GenHash()
	testTxPoolInstance.AddTransaction(tx8)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx8(nonce300):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 4)
	assert.Equal(t, pendingList[0], uint64(103))
	assert.Equal(t, pendingList[1], uint64(104))
	assert.Equal(t, pendingList[2], uint64(105))
	assert.Equal(t, pendingList[3], uint64(106))

	assert.Equal(t, len(queueList), 4)
	assert.Equal(t, queueList[0], uint64(108))
	assert.Equal(t, queueList[1], uint64(118))
	assert.Equal(t, queueList[2], uint64(188))
	assert.Equal(t, queueList[3], uint64(300))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 = testTxPoolInstance.received.get(tx4.Hash)
	assert.Equal(t, true, gotTx4 != nil)
	gotTx5 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx5 != nil)
	gotTx6 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx6 != nil)
	gotTx7 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx7 != nil)
	gotTx8 := testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx8 != nil)

	tx9 := &types.Transaction{
		Source: address1,
		Nonce:  256,
	}
	txs := []*types.Transaction{tx9}
	testTxPoolInstance.remove(txs, nil)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	pendingList = testTxPoolInstance.GetPendingList(address1)
	queueList = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after remove tx9(nonce256):\n")
	fmt.Printf("pending:%v\n", pendingList)
	fmt.Printf("queue:%v\n", queueList)
	assert.Equal(t, len(pendingList), 0)
	assert.Equal(t, len(queueList), 1)
	assert.Equal(t, queueList[0], uint64(300))

	gotTx1 = testTxPoolInstance.received.get(tx1.Hash)
	assert.Equal(t, true, gotTx1 != nil)
	gotTx2 = testTxPoolInstance.received.get(tx2.Hash)
	assert.Equal(t, true, gotTx2 != nil)
	gotTx3 = testTxPoolInstance.received.get(tx3.Hash)
	assert.Equal(t, true, gotTx3 != nil)
	gotTx4 = testTxPoolInstance.received.get(tx4.Hash)
	assert.Equal(t, true, gotTx4 != nil)
	gotTx5 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx5 != nil)
	gotTx6 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx6 != nil)
	gotTx7 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx7 != nil)
	gotTx8 = testTxPoolInstance.received.get(tx5.Hash)
	assert.Equal(t, true, gotTx8 != nil)
}

//2 address
//addr1 init nonce 5
//addr2 init nonce 200

// addr1 add nonce 6
// add2 add nonce 201
// add2 add nonce 188(nonce too low 进池子)
// add2 add nonce 222
// add1 add nonce 5
// add2 remove 201
func TestTxPool_QueueMultiAddress(t *testing.T) {
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

	//use TxPool instead of TransactionPool
	testTxPoolInstance = initTxPool()
	preTest()
	address1 := "0x2f4F09b722a6e5b77bE17c9A99c785Fa70111111"
	address2 := "0x2f4F09b722a6e5b77bE17c9A99c785Fa70222222"
	stateDB := middleware.AccountDBManagerInstance.GetLatestStateDB()
	stateDB.SetNonce(common.HexToAddress(address1), 5)
	stateDB.SetNonce(common.HexToAddress(address2), 200)
	fmt.Println("Init nonce.Addr1:5,addr2:200")
	tx10 := &types.Transaction{
		Source: address1,
		Nonce:  6,
	}
	tx10.Hash = tx10.GenHash()

	testTxPoolInstance.AddTransaction(tx10)
	pendingList1 := testTxPoolInstance.GetPendingList(address1)
	queueList1 := testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx10(addr1 nonce6):\n")
	fmt.Printf("addr1 pending:%v\n", pendingList1)
	fmt.Printf("addr1 queue:%v\n", queueList1)
	assert.Equal(t, len(pendingList1), 0)
	assert.Equal(t, len(queueList1), 1)
	assert.Equal(t, queueList1[0], uint64(6))
	gotTx10 := testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx10 != nil)

	pendingList2 := testTxPoolInstance.GetPendingList(address2)
	queueList2 := testTxPoolInstance.GetQueueList(address2)
	fmt.Printf("addr2 pending:%v\n", pendingList2)
	fmt.Printf("addr2 queue:%v\n", queueList2)
	assert.Equal(t, len(pendingList2), 0)
	assert.Equal(t, len(queueList2), 0)

	tx20 := &types.Transaction{
		Source: address2,
		Nonce:  201,
	}
	tx20.Hash = tx20.GenHash()
	testTxPoolInstance.AddTransaction(tx20)
	pendingList1 = testTxPoolInstance.GetPendingList(address1)
	queueList1 = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("after add tx20(addr2 nonce201):\n")
	fmt.Printf("addr1 pending:%v\n", pendingList1)
	fmt.Printf("addr1 queue:%v\n", queueList1)
	assert.Equal(t, len(pendingList1), 0)
	assert.Equal(t, len(queueList1), 1)
	assert.Equal(t, queueList1[0], uint64(6))
	pendingList2 = testTxPoolInstance.GetPendingList(address2)
	queueList2 = testTxPoolInstance.GetQueueList(address2)
	fmt.Printf("addr2 pending:%v\n", pendingList2)
	fmt.Printf("addr2 queue:%v\n", queueList2)
	assert.Equal(t, len(pendingList2), 0)
	assert.Equal(t, len(queueList2), 1)
	assert.Equal(t, queueList2[0], uint64(201))
	gotTx10 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx10 != nil)
	gotTx20 := testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx20 != nil)

	tx21 := &types.Transaction{
		Source: address2,
		Nonce:  188,
	}
	tx21.Hash = tx21.GenHash()
	testTxPoolInstance.AddTransaction(tx21)
	fmt.Printf("after add tx21(addr2 nonce188):\n")
	pendingList1 = testTxPoolInstance.GetPendingList(address1)
	queueList1 = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("addr1 pending:%v\n", pendingList1)
	fmt.Printf("addr1 queue:%v\n", queueList1)
	assert.Equal(t, len(pendingList1), 0)
	assert.Equal(t, len(queueList1), 1)
	assert.Equal(t, queueList1[0], uint64(6))
	pendingList2 = testTxPoolInstance.GetPendingList(address2)
	queueList2 = testTxPoolInstance.GetQueueList(address2)
	fmt.Printf("addr2 pending:%v\n", pendingList2)
	fmt.Printf("addr2 queue:%v\n", queueList2)
	assert.Equal(t, len(pendingList2), 0)
	assert.Equal(t, len(queueList2), 1)
	assert.Equal(t, queueList2[0], uint64(201))
	gotTx10 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx10 != nil)
	gotTx20 = testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx20 != nil)
	gotTx21 := testTxPoolInstance.received.get(tx21.Hash)
	assert.Equal(t, true, gotTx21 != nil)

	tx22 := &types.Transaction{
		Source: address2,
		Nonce:  222,
	}
	tx22.Hash = tx22.GenHash()
	testTxPoolInstance.AddTransaction(tx22)
	fmt.Printf("after add tx22(addr2 nonce222):\n")
	pendingList1 = testTxPoolInstance.GetPendingList(address1)
	queueList1 = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("addr1 pending:%v\n", pendingList1)
	fmt.Printf("addr1 queue:%v\n", queueList1)
	assert.Equal(t, len(pendingList1), 0)
	assert.Equal(t, len(queueList1), 1)
	assert.Equal(t, queueList1[0], uint64(6))
	pendingList2 = testTxPoolInstance.GetPendingList(address2)
	queueList2 = testTxPoolInstance.GetQueueList(address2)
	fmt.Printf("addr2 pending:%v\n", pendingList2)
	fmt.Printf("addr2 queue:%v\n", queueList2)
	assert.Equal(t, len(pendingList2), 0)
	assert.Equal(t, len(queueList2), 2)
	assert.Equal(t, queueList2[0], uint64(201))
	assert.Equal(t, queueList2[1], uint64(222))
	gotTx10 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx10 != nil)
	gotTx20 = testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx20 != nil)
	gotTx21 = testTxPoolInstance.received.get(tx21.Hash)
	assert.Equal(t, true, gotTx21 != nil)
	gotTx22 := testTxPoolInstance.received.get(tx22.Hash)
	assert.Equal(t, true, gotTx22 != nil)

	tx11 := &types.Transaction{
		Source: address1,
		Nonce:  5,
	}
	tx11.Hash = tx11.GenHash()
	testTxPoolInstance.AddTransaction(tx11)
	fmt.Printf("after add tx11(addr1 nonce5):\n")
	pendingList1 = testTxPoolInstance.GetPendingList(address1)
	queueList1 = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("addr1 pending:%v\n", pendingList1)
	fmt.Printf("addr1 queue:%v\n", queueList1)
	assert.Equal(t, len(pendingList1), 2)
	assert.Equal(t, len(queueList1), 0)
	assert.Equal(t, pendingList1[0], uint64(5))
	assert.Equal(t, pendingList1[1], uint64(6))
	pendingList2 = testTxPoolInstance.GetPendingList(address2)
	queueList2 = testTxPoolInstance.GetQueueList(address2)
	fmt.Printf("addr2 pending:%v\n", pendingList2)
	fmt.Printf("addr2 queue:%v\n", queueList2)
	assert.Equal(t, len(pendingList2), 0)
	assert.Equal(t, len(queueList2), 2)
	assert.Equal(t, queueList2[0], uint64(201))
	assert.Equal(t, queueList2[1], uint64(222))
	gotTx10 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx10 != nil)
	gotTx20 = testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx20 != nil)
	gotTx21 = testTxPoolInstance.received.get(tx21.Hash)
	assert.Equal(t, true, gotTx21 != nil)
	gotTx22 = testTxPoolInstance.received.get(tx22.Hash)
	assert.Equal(t, true, gotTx22 != nil)
	gotTx11 := testTxPoolInstance.received.get(tx11.Hash)
	assert.Equal(t, true, gotTx11 != nil)

	txs := []*types.Transaction{tx20}
	testTxPoolInstance.remove(txs, nil)
	fmt.Printf("after remove tx20(addr2 nonce201):\n")
	pendingList1 = testTxPoolInstance.GetPendingList(address1)
	queueList1 = testTxPoolInstance.GetQueueList(address1)
	fmt.Printf("addr1 pending:%v\n", pendingList1)
	fmt.Printf("addr1 queue:%v\n", queueList1)
	assert.Equal(t, len(pendingList1), 2)
	assert.Equal(t, len(queueList1), 0)
	assert.Equal(t, pendingList1[0], uint64(5))
	assert.Equal(t, pendingList1[1], uint64(6))
	pendingList2 = testTxPoolInstance.GetPendingList(address2)
	queueList2 = testTxPoolInstance.GetQueueList(address2)
	fmt.Printf("addr2 pending:%v\n", pendingList2)
	fmt.Printf("addr2 queue:%v\n", queueList2)
	assert.Equal(t, len(pendingList2), 0)
	assert.Equal(t, len(queueList2), 1)
	assert.Equal(t, queueList2[0], uint64(222))
	gotTx10 = testTxPoolInstance.received.get(tx10.Hash)
	assert.Equal(t, true, gotTx10 != nil)
	gotTx20 = testTxPoolInstance.received.get(tx20.Hash)
	assert.Equal(t, true, gotTx20 == nil)
	gotTx21 = testTxPoolInstance.received.get(tx21.Hash)
	assert.Equal(t, true, gotTx21 != nil)
	gotTx22 = testTxPoolInstance.received.get(tx22.Hash)
	assert.Equal(t, true, gotTx22 != nil)
	gotTx11 = testTxPoolInstance.received.get(tx11.Hash)
	assert.Equal(t, true, gotTx11 != nil)
}

func TestMockCastBlockNormal(t *testing.T) {
	castBlockInterval := time.Second * 2
	addTxInterval := time.Millisecond * 1
	mockFork := false
	mockBadNonceTx := false
	mockCastBlock(castBlockInterval, addTxInterval, mockFork, mockBadNonceTx)
}

func TestMockCastBlockWithBadNonce(t *testing.T) {
	castBlockInterval := time.Second * 2
	addTxInterval := time.Millisecond * 50
	mockFork := false
	mockBadNonceTx := true
	mockCastBlock(castBlockInterval, addTxInterval, mockFork, mockBadNonceTx)
}

func TestMockCastBlockWithFork(t *testing.T) {
	castBlockInterval := time.Second * 2
	addTxInterval := time.Millisecond * 50
	mockFork := true
	mockBadNonceTx := false
	mockCastBlock(castBlockInterval, addTxInterval, mockFork, mockBadNonceTx)
}

func TestMockCastBlockWithForkAndBadNonce(t *testing.T) {
	castBlockInterval := time.Second * 2
	addTxInterval := time.Millisecond * 50
	mockFork := true
	mockBadNonceTx := true
	mockCastBlock(castBlockInterval, addTxInterval, mockFork, mockBadNonceTx)
}

func mockCastBlock(castBlockInterval time.Duration, addTxInterval time.Duration, mockFork bool, mockBadNonceTx bool) {
	//defer func() {
	//	Close()
	//	middleware.Close()
	//	log.Close()
	//
	//	os.RemoveAll("0.ini")
	//	os.RemoveAll("logs")
	//
	//	err := os.RemoveAll("storage0")
	//	if nil != err {
	//		t.Fatal(err)
	//	}
	//}()

	preTest()
	address1 := "0x1111111111111111111111111111111111111111"
	address2 := "0x2222222222222222222222222222222222222222"
	address3 := "0x3333333333333333333333333333333333333333"
	addressList := []string{address1, address2, address3}
	monitorTicker := time.NewTicker(time.Second * 10)
	go mockProposer(castBlockInterval, mockFork)
	go mockClient(addressList, addTxInterval)

	if mockBadNonceTx {
		go mockBadNonceClient(addressList)
	}
	for {
		<-monitorTicker.C
		printMonitorLog(addressList)
	}
}

func mockProposer(castBlockInterval time.Duration, mockFork bool) {
	var height uint64 = 10001
	var lastStateVersion int
	for ; ; height++ {
		txs := txpoolInstance.PackForCast(height)
		state := middleware.AccountDBManagerInstance.GetLatestStateDB()
		lastStateVersion = state.Snapshot()
		successTxs, evictedTxs, receipts := mockExecuteTxs(txs, height, state)
		header := types.BlockHeader{}

		txpoolInstance.MarkExecuted(&header, receipts, successTxs, evictedTxs)
		middleware.AccountDBManagerInstance.SetLatestStateDB(state, make(map[string]uint64), height+1)
		time.Sleep(castBlockInterval)

		if mockFork && height > 10005 && time.Now().UnixMilli()%3 > 1 {
			fmt.Printf("forked!!\n")
			state.RevertToSnapshot(lastStateVersion)
			middleware.AccountDBManagerInstance.SetLatestStateDB(state, make(map[string]uint64), height)
			block1 := &types.Block{Transactions: successTxs, Header: &types.BlockHeader{EvictedTxs: evictedTxs}}
			txpoolInstance.UnMarkExecuted(block1)
		}
	}
}

func mockExecuteTxs(txs []*types.Transaction, height uint64, state *account.AccountDB) ([]*types.Transaction, []common.Hash, []*types.Receipt) {
	successTxs := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	receipts := make([]*types.Receipt, 0)

	for _, tx := range txs {
		//fmt.Printf("execute souce:%s,nonce:%d,hash:%s\n", tx.Source, tx.Nonce, tx.Hash.String())
		expectedNonce := state.GetNonce(common.HexToAddress(tx.Source))
		if expectedNonce > tx.Nonce {
			evictedTxs = append(evictedTxs, tx.Hash)
			fmt.Printf("[%s]Tx nonce too low.tx:%s,source:%s,expected:%d,but:%d\n", time.Now().String(), tx.Hash.String(), tx.Source, expectedNonce, tx.Nonce)
		} else if expectedNonce < tx.Nonce {
			evictedTxs = append(evictedTxs, tx.Hash)
			fmt.Printf("Tx nonce too high.tx:%s,source:%s,expected:%d,but:%d\n", tx.Hash.String(), tx.Source, expectedNonce, tx.Nonce)
		} else {
			state.SetNonce(common.HexToAddress(tx.Source), tx.Nonce+1)
			receipt := types.Receipt{}
			receipt.TxHash = tx.Hash
			receipt.Height = height
			receipt.Status = 0
			receipts = append(receipts, &receipt)
			successTxs = append(successTxs, tx)
		}
	}
	return successTxs, evictedTxs, receipts
}

func mockClient(addressList []string, addTxInterval time.Duration) {
	for {
		index := time.Now().UnixMilli() % 3
		address := addressList[index]
		tx := types.Transaction{Type: 188, Source: address, Nonce: txpoolInstance.GetPendingNonce(address)}
		tx.Hash = tx.GenHash()
		_, err := txpoolInstance.AddTransaction(&tx)
		if err != nil {
			fmt.Printf("add normal tx error.Error:%s, nonce:%d,hash:%s\n", err.Error(), tx.Nonce, tx.Hash.String())
		}
		time.Sleep(addTxInterval)
	}
}

func printMonitorLog(addressList []string) {
	fmt.Printf("[%s]tx pool receive size:%d\n", time.Now().String(), len(txpoolInstance.GetReceived()))
	for _, address := range addressList {
		fmt.Printf("%s\n", address)
		pendingList := txpoolInstance.GetPendingList(address)
		if pendingList == nil {
			fmt.Printf("pending list is nil\n")
		} else {
			//fmt.Printf("pending list size:%d\n", len(pendingList))
			fmt.Printf("pending list size:%d.%v\n", len(pendingList), pendingList)
		}

		queueList := txpoolInstance.GetQueueList(address)
		if queueList == nil {
			fmt.Printf("queue list is nil\n")
		} else {
			//fmt.Printf("queue list size:%d\n", len(queueList))
			fmt.Printf("queue list size:%d.%v\n", len(queueList), queueList)
		}
	}
	fmt.Printf("\n\n")
}

func mockBadNonceClient(addressList []string) {
	for {
		address := addressList[0]
		pendingNonce := txpoolInstance.GetPendingNonce(address)
		index := time.Now().UnixMilli() % 3
		//index = 1
		//index = 2
		var nonce uint64
		switch index {
		case 0:
			nonce = pendingNonce
		case 1:
			//nonce = pendingNonce + 1000000 //test queue tx expire
			nonce = pendingNonce + 2
		case 2:
			if pendingNonce > 2 {
				nonce = pendingNonce - 2
			} else {
				nonce = 0
			}
		}
		tx := types.Transaction{Type: 188, Source: address, Nonce: nonce, Data: "123dahfaoldnflajeoifaejfdaefa"}
		tx.Hash = tx.GenHash()
		_, err := txpoolInstance.AddTransaction(&tx)
		if err != nil {
			fmt.Printf("Add bad nonce tx failed! real nocne:%d,pending nonce:%d,txHash:%s,err:%s\n", nonce, pendingNonce, tx.Hash.String(), err.Error())
		} else {
			fmt.Printf("Add bad nonce tx success! real nonce:%d,pending nonce:%d,txHash:%s\n", nonce, pendingNonce, tx.Hash.String())
		}
		//if index == 1 {
		//	printMonitorLog(addressList)
		//}
		time.Sleep(time.Second * 1)
		//if index == 1 {
		//	printMonitorLog(addressList)
		//}
	}
}
