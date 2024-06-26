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
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"fmt"
	"github.com/gogf/gf/container/gmap"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"testing"
	"time"
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	txs := txpoolInstance.PackForCast(10000, state)
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  10,
	}

	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000, state)
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}

	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000, state)
	if 1 != len(txs) {
		t.Fatal("no txs error")
	}

	tx1 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xbb"),
		Nonce:  1,
	}
	txpoolInstance.AddTransaction(tx1)
	txs1 := txpoolInstance.PackForCast(10000, state)
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}

	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000, state)
	if 1 != len(txs) {
		t.Fatal("no txs error")
	}

	tx1 := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xbb"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx1)
	txs1 := txpoolInstance.PackForCast(10000, state)
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}

	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000, state)
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
	txs1 := txpoolInstance.PackForCast(10000, state)
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})

	tx := &types.Transaction{
		Source: "0x0001",
		Hash:   common.HexToHash("0xaa"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx)
	txs := txpoolInstance.PackForCast(10000, state)
	if 1 != len(txs) {
		t.Fatal("no txs error")
	}

	tx1 := &types.Transaction{
		Source: "0x0002",
		Hash:   common.HexToHash("0xbb"),
		Nonce:  0,
	}
	txpoolInstance.AddTransaction(tx1)
	txs1 := txpoolInstance.PackForCast(10000, state)
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
	txs2 := txpoolInstance.PackForCast(10000, state)
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})

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
		txs := txpoolInstance.PackForCast(10000, state)
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
	txs := txpoolInstance.PackForCast(10000, state)
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
	state, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})

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

	txList := txpoolInstance.PackForCast(10000, state)
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

func TestMockCastWithMixTxType(t *testing.T) {
	castBlockInterval := time.Second * 2
	addRPCInterval := time.Millisecond * 100
	addOriginInterval := time.Millisecond * 500
	mockCastBlock(castBlockInterval, addRPCInterval, addOriginInterval)
}

func mockCastBlock(castBlockInterval time.Duration, addRPCInterval time.Duration, addOriginInterval time.Duration) {
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
	middleware.AccountDBManagerInstance.SetHandler(mockOriginTxHandler)
	address1 := "0x1111111111111111111111111111111111111111"
	address2 := "0x2222222222222222222222222222222222222222"
	//address3 := "0x3333333333333333333333333333333333333333"

	go mockProposer(castBlockInterval)
	go mockRPCClient(address1, addRPCInterval)
	go mockOriginClient(address2, addOriginInterval)
	for {

	}
}

func mockProposer(castBlockInterval time.Duration) {
	var height uint64 = 10001
	for ; ; height++ {
		state := middleware.AccountDBManagerInstance.GetLatestStateDB()
		txs := txpoolInstance.PackForCast(height, state)
		fmt.Printf("cast block:%d,tx length:%d\n", height, len(txs))
		successTxs, evictedTxs, receipts := mockExecuteTxs(txs, height, state)
		header := types.BlockHeader{}
		nonces := make(map[string]uint64)
		for _, tx := range txs {
			if tx.RequestId > nonces["fixed"] {
				nonces["fixed"] = tx.RequestId
			}
		}

		txpoolInstance.MarkExecuted(&header, receipts, successTxs, evictedTxs)
		middleware.AccountDBManagerInstance.SetLatestStateDB(state, nonces, height+1)
		time.Sleep(castBlockInterval)
	}
}

func mockExecuteTxs(txs []*types.Transaction, height uint64, state *account.AccountDB) ([]*types.Transaction, []common.Hash, []*types.Receipt) {
	successTxs := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	receipts := make([]*types.Receipt, 0)

	for _, tx := range txs {
		if tx.RequestId > 0 {
			continue
		}
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

func mockRPCClient(address string, addTxInterval time.Duration) {
	var nonce uint64 = 0
	for {
		tx := types.Transaction{Type: 188, Source: address, Nonce: nonce}
		tx.Hash = tx.GenHash()
		_, err := txpoolInstance.AddTransaction(&tx)
		if err != nil {
			fmt.Printf("add normal tx error.Error:%s, nonce:%d,hash:%s\n", err.Error(), tx.Nonce, tx.Hash.String())
		}
		nonce++
		time.Sleep(addTxInterval)
	}
}

func mockOriginClient(address string, addTxInterval time.Duration) {
	var requestId uint64 = 1
	for {
		tx := types.Transaction{Type: 200, Source: address, Nonce: 100, RequestId: requestId, Time: time.Now().String()}
		tx.Hash = tx.GenHash()

		var msg notify.ClientTransactionMessage
		msg.Tx = tx
		msg.UserId = ""
		msg.GateNonce = 0
		msg.Nonce = requestId
		middleware.DataChannel.GetRcvedTx() <- &msg
		requestId++
		time.Sleep(addTxInterval)
	}
}

func mockOriginTxHandler(item *middleware.Item) {
	message := item.Value
	txRaw := message.Tx
	txRaw.RequestId = message.Nonce
	txRaw.SubTransactions = make([]types.UserData, 1)
	data := types.UserData{Address: message.GateNonce}
	txRaw.SubTransactions[0] = data

	_, err := GetTransactionPool().AddTransaction(&txRaw)
	if err != nil {
		fmt.Printf("handler origin tx err:%s\n", err.Error())
	}
}
