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
	"com.tuntun.rangers/node/src/eth_tx"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/mysql"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/storage/rlp"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
)

const (
	txDataBasePrefix = "tx"
	rcvTxPoolSize    = 50000
	txCacheSize      = 1000

	txCountPerBlock = 100
)

var (
	ErrNil = errors.New("nil transaction")

	ErrChainId = errors.New("illegal chain id")

	ErrHash = errors.New("illegal transaction hash")

	ErrSign = errors.New("illegal transaction signature")

	ErrExist = errors.New("transaction already exist in pool")

	ErrEvicted = errors.New("error transaction already exist in pool")

	ErrIllegal = errors.New("illegal transaction")
)

type ExecutedReceipt struct {
	types.Receipt
	BlockHash common.Hash `json:"blockHash"`
}

type ExecutedTransaction struct {
	Receipt     ExecutedReceipt
	Transaction []byte
}

type TransactionPool interface {
	PackForCast(height uint64) []*types.Transaction

	//add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

	GetTransaction(hash common.Hash) (*types.Transaction, error)

	GetTransactionStatus(hash common.Hash) (uint, error)

	GetExecuted(hash common.Hash) *ExecutedTransaction

	GetReceived() []*types.Transaction

	TxNum() int

	MarkExecuted(header *types.BlockHeader, receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash)

	UnMarkExecuted(block *types.Block)

	Clear()

	IsExisted(hash common.Hash) bool

	VerifyTransaction(tx *types.Transaction, height uint64) error

	ProcessFee(tx types.Transaction, accountDB *account.AccountDB) error

	GetGateNonce() uint64
}

type TxPool struct {
	evictedTxs *lru.Cache
	received   *simpleContainer

	executed db.Database
	batch    db.Batch
}

var (
	txpoolInstance TransactionPool
	delta, _       = utility.StrToBigInt("0.0001")
)

func initTransactionPool() {
	if nil == txpoolInstance {
		txpoolInstance = newTransactionPool()
	}
}

func GetTransactionPool() TransactionPool {
	return txpoolInstance
}

func newTransactionPool() TransactionPool {
	pool := &TxPool{}
	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.evictedTxs, _ = lru.New(txCacheSize)

	executed, err := db.NewLDBDatabase(txDataBasePrefix, 16, 128)
	if err != nil {
		txPoolLogger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()

	return pool
}

func (pool *TxPool) refreshGateNonce(tx *types.Transaction) {
	sub := tx.SubTransactions
	if 1 == len(sub) && 0 != sub[0].Address {
		txPoolLogger.Debugf("refreshGateNonce. txhash: %s, gateNonce: %d", tx.Hash.String(), sub[0].Address)
		pool.batch.Put(utility.StrToBytes(txDataBasePrefix), utility.UInt64ToByte(sub[0].Address))
	}
}

func (pool *TxPool) GetGateNonce() uint64 {
	data, err := pool.executed.Get(utility.StrToBytes(txDataBasePrefix))
	if err != nil || 0 == len(data) {
		return 0
	}

	return utility.ByteToUInt64(data)
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	if pool.evictedTxs.Contains(tx.Hash) {
		txPoolLogger.Infof("Tx is marked evicted tx,do not add pool. Hash:%s", tx.Hash.String())
		return false, ErrEvicted
	}

	b, err := pool.add(tx)
	if nil == err {
		pool.refreshGateNonce(tx)
	}
	return b, err
}

func (pool *TxPool) MarkExecuted(header *types.BlockHeader, receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash) {
	if nil == receipts || 0 == len(receipts) {
		return
	}

	mysql.InsertLogs(header.Height, receipts, header.Hash)

	txHashList := make([]interface{}, len(receipts))
	for i, receipt := range receipts {
		hash := receipt.TxHash
		txHashList[i] = hash
		var er ExecutedReceipt
		er.BlockHash = header.Hash
		er.Height = receipt.Height
		er.TxHash = receipt.TxHash
		er.Status = receipt.Status
		er.Logs = receipt.Logs
		er.ContractAddress = receipt.ContractAddress
		er.GasUsed = receipt.GasUsed
		if 0 != len(receipt.Result) {
			er.Result = receipt.Result
		}
		executedTx := &ExecutedTransaction{
			Receipt: er,
		}

		tx := findTxInList(txs, hash, i)
		txData, _ := types.MarshalTransaction(tx)
		executedTx.Transaction = txData
		executedTxBytes, err := json.Marshal(executedTx)
		if nil != err {
			continue
		}
		pool.batch.Put(hash.Bytes(), executedTxBytes)
		if pool.batch.ValueSize() > 100*1024 {
			pool.batch.Write()
			pool.batch.Reset()
		}
		pool.refreshGateNonce(tx)
	}
	if pool.batch.ValueSize() > 0 {
		pool.batch.Write()
		pool.batch.Reset()
	}

	if evictedTxs != nil {
		for _, hash := range evictedTxs {
			pool.evictedTxs.Add(hash, 0)
		}
	}
	pool.remove(txHashList)
}

func (pool *TxPool) UnMarkExecuted(block *types.Block) {
	txs := block.Transactions
	evictedTxs := block.Header.EvictedTxs
	if nil == txs || 0 == len(txs) {
		return
	}

	mysql.DeleteLogs(block.Header.Height, block.Header.Hash)

	if evictedTxs != nil {
		for _, hash := range evictedTxs {
			pool.evictedTxs.Remove(hash)
		}
	}

	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
		var msg notify.ClientTransactionMessage
		msg.Tx = *tx
		msg.Nonce = tx.RequestId
		msg.UserId = ""
		msg.GateNonce = 0
		middleware.DataChannel.GetRcvedTx() <- &msg
	}
}

func (pool *TxPool) IsExisted(hash common.Hash) bool {
	return pool.isTransactionExisted(hash)
}

const (
	missingTx1 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 17:47:34.334308011 +0800 CST m=+707184.175867286","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0x8ae1c35d909cf45ae136dd8f683a88d5cea87b1ef187f716ad90d29abd540159","RequestId":0,"chainId":"2025","sign":"0xe8fb3f557341a7c1e37c197d7dc2fdb8a9a56b50679ca12d33774010ea6e47cd3e60e383d78698132a3fc1d0152c6d74f692f53edd5bae5d6fb7cc57aa2ae8851c"}`
	missingTx2 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 17:48:00.277689243 +0800 CST m=+707210.119248518","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0xa1a391f7c438ef00d3dfbc171e9e5a71ea01bd25b817cda05370518710c7d9ec","RequestId":0,"chainId":"2025","sign":"0xa0cb5e6b24912a2ffed63011a55bc3cf7f2426dbfc092d1dc8837f642987d222461de19cae07e1c5c7dc831d43dffb38210f702e9557aab53a6fadf0bf06426a1b"}`
	missingTx3 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 17:49:06.386777448 +0800 CST m=+707276.228336748","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0x3581f3cef722a8fecfcfc56fb76f2c6d973e98db6d210e527bb0a6931edf75d8","RequestId":0,"chainId":"2025","sign":"0xaec19436329efe55aa38ace93227f7056fd7cccc9f78757b0eb9e9a50e4b86514da027764760926144cd2cd2999d42a49283eefa82870513e4acdfd7c322a04a1c"}`
	missingTx4 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 17:53:49.586530619 +0800 CST m=+707559.428089901","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0x61c8736c78c40855f3d2398adc339459353689ea45242acfc56caf25fc46b724","RequestId":0,"chainId":"2025","sign":"0x49dd59cab656c669501cfe28508f4f5ba201755f6a99a73a669e02943a55a4600e2a1da7dd92d579511a750a4801688b11488982e98ed83fa749813ab7e3b6551c"}`
	missingTx5 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 17:56:34.515097387 +0800 CST m=+707724.356656662","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0xdc72647dac1f31d29633c4fed1248c4b5b423f0dbe1693a2e5ecec694a49758c","RequestId":0,"chainId":"2025","sign":"0xde7a0a03d0bd6149784700fcdfecc9560b250d1402b0ba247942edc45d6356be4a732983bc19ffd87b83702393d47b9c5f772f5ef2bd7a527269fe7bbe04bb231c"}`
	missingTx6 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 18:00:05.174680873 +0800 CST m=+707935.016240183","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0x0814d53ebaf1494cb0675f1ec99010e62af4b9599ebf987ccf2a93f7292484a0","RequestId":0,"chainId":"2025","sign":"0x785a7ef762ec4bc35c319bf687ae2b00cd459ec78ec9acf58f906e258c04de0944b2be5d0f66c9edc312a45fab5b292f4b785cf0a4ef3adc6e78d7709e384e351c"}`
	missingTx7 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 18:01:26.048108346 +0800 CST m=+708015.889667620","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0x9e619ad9268c45e3610b34cb70b98da9f06715cdbaec85f00eb9ce4136920cd6","RequestId":0,"chainId":"2025","sign":"0x7c2576310da4d13b45fab6412027b1c4b713f2fd319670d8f28277a77bbeae391d89b901a0183e542e788d56469c6541f6ca1313298710344bfd3a8313cb0d5c1b"}`
	missingTx8 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 18:34:48.14303912 +0800 CST m=+710017.984598395","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0xe10a5a02b41146850a5c790a5d8e5ac62911b856aeefb0e60e54c225b09122dc","RequestId":0,"chainId":"2025","sign":"0x50a5b813425d3a5480eab7a363184b8ded3e68ea8505b68c032197660c3798491bc8beacdc4ef4cdc89ed1e2081ef70e766d78ce7621cdc8115b53075fbecf961b"}`
	missingTx9 = `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 18:36:11.307691143 +0800 CST m=+710101.149250432","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0x5129106d251b123e5e5ad0bf5ec3dcf17a16f0b22b6af8d428b36598431a0dd7","RequestId":0,"chainId":"2025","sign":"0x26a17ccc063d1284f0137f46da9f4f2c0d170be75acc75349f5219685eabe0b9313e8706bef005a2feb419f5dffbcec0e8cae070259e96b5a8289041050cba071b"}`
)

func (pool *TxPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	switch hash.String() {
	case "0x8ae1c35d909cf45ae136dd8f683a88d5cea87b1ef187f716ad90d29abd540159":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx1), &tx)
		return &tx, nil
	case "0xa1a391f7c438ef00d3dfbc171e9e5a71ea01bd25b817cda05370518710c7d9ec":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx2), &tx)
		return &tx, nil
	case "0x3581f3cef722a8fecfcfc56fb76f2c6d973e98db6d210e527bb0a6931edf75d8":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx3), &tx)
		return &tx, nil
	case "0x61c8736c78c40855f3d2398adc339459353689ea45242acfc56caf25fc46b724":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx4), &tx)
		return &tx, nil
	case "0xdc72647dac1f31d29633c4fed1248c4b5b423f0dbe1693a2e5ecec694a49758c":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx5), &tx)
		return &tx, nil
	case "0x0814d53ebaf1494cb0675f1ec99010e62af4b9599ebf987ccf2a93f7292484a0":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx6), &tx)
		return &tx, nil
	case "0x9e619ad9268c45e3610b34cb70b98da9f06715cdbaec85f00eb9ce4136920cd6":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx7), &tx)
		return &tx, nil
	case "0xe10a5a02b41146850a5c790a5d8e5ac62911b856aeefb0e60e54c225b09122dc":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx8), &tx)
		return &tx, nil
	case "0x5129106d251b123e5e5ad0bf5ec3dcf17a16f0b22b6af8d428b36598431a0dd7":
		var tx types.Transaction
		json.Unmarshal([]byte(missingTx9), &tx)
		return &tx, nil
	}
	receivedTx := pool.received.get(hash)
	if nil != receivedTx {
		return receivedTx, nil
	}

	executedTx := pool.GetExecuted(hash)
	if nil != executedTx {
		tx, _ := types.UnMarshalTransaction(executedTx.Transaction)
		return &tx, nil
	}
	return nil, ErrNil
}

func (pool *TxPool) GetTransactionStatus(hash common.Hash) (uint, error) {
	executedTx := pool.GetExecuted(hash)
	if executedTx == nil {
		return 0, ErrNil
	}
	return executedTx.Receipt.Status, nil
}

func (pool *TxPool) Clear() {
	middleware.LockBlockchain("Clear")
	defer middleware.UnLockBlockchain("Clear")

	executed, _ := db.NewDatabase(txDataBasePrefix)
	pool.executed = executed
	pool.batch.Reset()

	pool.received = newSimpleContainer(rcvTxPoolSize)
}

func (pool *TxPool) GetReceived() []*types.Transaction {
	items := pool.received.asSlice()
	result := make([]*types.Transaction, len(items))

	for i, item := range items {
		result[i] = item.(*types.Transaction)
	}
	return result
}

func (pool *TxPool) GetExecuted(hash common.Hash) *ExecutedTransaction {
	txBytes, _ := pool.executed.Get(hash.Bytes())
	if txBytes == nil {
		return nil
	}

	var executedTx *ExecutedTransaction
	err := json.Unmarshal(txBytes, &executedTx)
	if err != nil || executedTx == nil {
		return nil
	}
	//complete log's txHash and blockHash filed
	if len(executedTx.Receipt.Logs) > 0 {
		for _, log := range executedTx.Receipt.Logs {
			log.BlockHash = executedTx.Receipt.BlockHash
			log.TxHash = executedTx.Receipt.TxHash
		}
	}
	return executedTx
}

func (p *TxPool) TxNum() int {
	return p.received.Len()
}

func (pool *TxPool) PackForCast(height uint64) []*types.Transaction {
	packedTxs := make([]*types.Transaction, 0)

	txs := pool.received.asSlice()
	if 0 == len(txs) {
		txPoolLogger.Debugf("packed no tx. height: %d", height)
		return packedTxs
	}

	for i, tx := range txs {
		packedTxs = append(packedTxs, tx.(*types.Transaction))
		if i >= txCountPerBlock {
			break
		}
	}

	txPoolLogger.Debugf("packed tx. height: %d. nonce from %d to %d. size: %d", height, packedTxs[0].RequestId, packedTxs[len(packedTxs)-1].RequestId, len(packedTxs))
	return packedTxs
}

func (pool *TxPool) VerifyTransaction(tx *types.Transaction, height uint64) error {
	if tx.Type == types.TransactionTypeETHTX {
		return verifyETHTx(tx, height)
	}
	err := verifyTxChainId(tx, height)
	if nil != err {
		return err
	}
	err = verifyTransactionHash(tx)
	if nil != err {
		return err
	}
	err = verifyTransactionSign(tx)
	if nil != err {
		return err
	}

	txPoolLogger.Debugf("Verify tx success. hash: %s", tx.Hash.String())
	return nil
}

func (pool *TxPool) ProcessFee(tx types.Transaction, accountDB *account.AccountDB) error {
	addr := common.HexStringToAddress(tx.Source)
	balance := accountDB.GetBalance(addr)

	if balance.Cmp(delta) < 0 {
		msg := fmt.Sprintf("not enough max, addr: %s, balance: %s", tx.Source, balance)
		return fmt.Errorf(msg)
	}
	accountDB.SubBalance(addr, delta)
	accountDB.AddBalance(common.FeeAccount, delta)
	return nil
}

func (pool *TxPool) add(tx *types.Transaction) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}

	hash := tx.Hash
	if pool.isTransactionExisted(hash) {
		return false, ErrExist
	}
	pool.received.push(tx)
	txPoolLogger.Debugf("[pool]Add tx:%s. nonce: %d, After add,received size:%d", tx.Hash.String(), tx.RequestId, pool.received.Len())
	return true, nil
}

func (pool *TxPool) remove(txHashList []interface{}) {
	pool.received.remove(txHashList)
	txPoolLogger.Debugf("[pool]removed tx, %d After remove,received size:%d", len(txHashList), pool.received.Len())

}

func (pool *TxPool) isTransactionExisted(hash common.Hash) bool {
	existInReceivedTxs := pool.received.contains(hash)
	if existInReceivedTxs {
		return true
	}

	isExecutedTx, _ := pool.executed.Has(hash.Bytes())
	return isExecutedTx
}

func findTxInList(txs []*types.Transaction, txHash common.Hash, receiptIndex int) *types.Transaction {
	if nil == txs || 0 == len(txs) {
		return nil
	}

	if receiptIndex < len(txs) && txs[receiptIndex].Hash == txHash {
		return txs[receiptIndex]
	}

	for _, tx := range txs {
		if tx.Hash == txHash {
			return tx
		}
	}
	return nil
}

func verifyTxChainId(tx *types.Transaction, height uint64) error {
	expectedChainId := common.ChainId(height)
	if tx.ChainId != expectedChainId {
		txPoolLogger.Errorf("Verify chain id error!Hash:%s,chainId:%s,expect chainId:%s", tx.Hash.String(), tx.ChainId, expectedChainId)
		return ErrChainId
	}
	return nil
}

func verifyTransactionHash(tx *types.Transaction) error {
	expectHash := tx.GenHash()
	if tx.Hash != expectHash {
		txPoolLogger.Errorf("Verify tx hash error!Hash:%s,expect hash:%s", tx.Hash.String(), expectHash.String())
		return ErrHash
	}
	return nil
}

func verifyTransactionSign(tx *types.Transaction) error {
	if tx.Sign == nil {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s,error:nil sign!", tx.Hash.String())
		return ErrSign
	}

	hashByte := tx.Hash.Bytes()
	pk, err := tx.Sign.RecoverPubkey(hashByte)
	if err != nil {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s,error:%s", tx.Hash.String(), err.Error())
		return ErrSign
	}
	if !pk.Verify(hashByte, tx.Sign) {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s, error: verify sign fail", tx.Hash.String())
		return ErrSign
	}
	expectAddr := pk.GetAddress().GetHexString()
	if tx.Source != expectAddr {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s,error:illegal signer! source:%s,expect source:%s", tx.Hash.String(), tx.Source, expectAddr)
		return ErrSign
	}
	return nil
}

func verifyETHTx(tx *types.Transaction, height uint64) error {
	if tx == nil {
		return ErrNil
	}
	ethTx := new(eth_tx.Transaction)
	var encodedTx utility.Bytes
	encodedTx = common.FromHex(tx.ExtraData)
	if err := rlp.DecodeBytes(encodedTx, ethTx); err != nil {
		txPoolLogger.Errorf("Verify eth tx rlp error!error:%v", err)
		return ErrIllegal
	}

	signer := eth_tx.NewEIP155Signer(common.GetChainId(height))
	sender, err := eth_tx.Sender(signer, ethTx)
	if err != nil {
		txPoolLogger.Errorf("Verify eth tx error!tx:%s,error:%v", ethTx.Hash().String(), err)
		return ErrIllegal
	}

	expectedTx := eth_tx.ConvertTx(ethTx, sender, encodedTx)
	if !compareTx(tx, expectedTx) {
		txPoolLogger.Errorf("Verify eth tx error:tx diff!tx:%s,expected tx:%s", tx.ToTxJson().ToString(), expectedTx.ToTxJson().ToString())
		return ErrIllegal
	}
	txPoolLogger.Debugf("Verify eth tx success. hash: %s", tx.Hash.String())
	return nil
}

func compareTx(tx *types.Transaction, expectedTx *types.Transaction) bool {
	if tx == nil || expectedTx == nil {
		return false
	}
	if tx.Source != expectedTx.Source || tx.Target != expectedTx.Target || tx.Type != expectedTx.Type || tx.ExtraData != expectedTx.ExtraData {
		return false
	}
	if tx.Nonce != expectedTx.Nonce || tx.ChainId != expectedTx.ChainId || tx.Data != expectedTx.Data || tx.Hash != expectedTx.Hash {
		return false
	}
	return true
}
