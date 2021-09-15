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

package core

import (
	"com.tuntun.rocket/node/src/common"
	executors "com.tuntun.rocket/node/src/executor"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

// 客户端web socket 请求的返回数据结构
type response struct {
	Id      string `json:"id,omitempty"`
	Status  string `json:"status,omitempty"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func (executor *GameExecutor) makeSuccessResponse(data string, id string) []byte {
	res := response{
		Data:   data,
		Id:     id,
		Status: "0",
	}

	result, err := json.Marshal(res)
	if err != nil {
		executor.logger.Debugf("json make success response err:%s", err.Error())
	}
	return result
}

func (executor *GameExecutor) makeFailedResponse(message string, id string) []byte {
	res := response{
		Message: message,
		Id:      id,
		Status:  "1",
	}

	result, err := json.Marshal(res)
	if err != nil {
		executor.logger.Debugf("json make failed response err:%s", err.Error())
	}
	return result
}

const maxWriteSize = 100000

// 用于处理client websocket请求
type GameExecutor struct {
	chain     *blockChain
	writeChan chan notify.ClientTransactionMessage
	logger    log.Logger
	tempTx    *db.LDBDatabase
	cleaner   *time.Ticker
}

func initGameExecutor(blockChainImpl *blockChain) {
	gameExecutor := GameExecutor{chain: blockChainImpl}
	gameExecutor.logger = log.GetLoggerByIndex(log.GameExecutorLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	gameExecutor.writeChan = make(chan notify.ClientTransactionMessage, maxWriteSize)
	gameExecutor.cleaner = time.NewTicker(time.Minute * 10)

	file := "tempTx"
	tempTxLDB, err := db.NewLDBDatabase(file, 10, 10)
	if err != nil {
		panic("newLDBDatabase fail, file=" + file + ", err=" + err.Error())
	}
	gameExecutor.tempTx = tempTxLDB
	gameExecutor.recover()

	//notify.BUS.Subscribe(notify.ClientTransactionRead, gameExecutor.read)
	notify.BUS.Subscribe(notify.ClientTransaction, gameExecutor.write)
	notify.BUS.Subscribe(notify.CoinProxyNotify, gameExecutor.coinProxyHandler)
	notify.BUS.Subscribe(notify.WrongTxNonce, gameExecutor.wrongTxNonceHandler)
	notify.BUS.Subscribe(notify.BlockAddSucc, gameExecutor.onBlockAddSuccess)

	go gameExecutor.loop()
	go gameExecutor.cleanLoop()
}

func (executor *GameExecutor) recover() {
	executor.logger.Warnf("start recover")
	iterator := executor.tempTx.NewIterator()
	for iterator.Next() {
		txBytes := iterator.Value()
		tx, err := types.UnMarshalTransaction(txBytes)
		if err != nil {
			continue
		}

		msg := notify.ClientTransactionMessage{
			Tx:     tx,
			UserId: "",
			Nonce:  binary.BigEndian.Uint64(iterator.Key()),
		}
		go executor.RunWrite(msg)
		executor.logger.Warnf("recover tx, nonce: %d, hash: %s", msg.Nonce, tx.Hash.ShortS())
	}
	executor.logger.Warnf("end recover")
}

func (executor *GameExecutor) write(msg notify.Message) {
	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		executor.logger.Errorf("GameExecutor: Write assert not ok!")
		return
	}

	if len(executor.writeChan) == maxWriteSize {
		executor.logger.Errorf("write rcv message error: %v", msg)
		return
	}

	executor.logger.Debugf("write rcv message: %s", message.TOJSONString())
	executor.writeChan <- *message
}

func (executor *GameExecutor) wrongTxNonceHandler(msg notify.Message) {
	cpn, ok := msg.(*notify.NonceNotifyMessage)
	if !ok {
		logger.Debugf("wrongTxNonceHandler: Message assert not ok!")
		return
	}

	executor.logger.Warnf("process wrong nonce: %d, reason: %s", cpn.Nonce, cpn.Msg)
	tx := types.Transaction{Type: types.TransactionTypeWrongTxNonce, RequestId: cpn.Nonce}
	tx.Hash = common.BytesToHash(common.Sha256(utility.UInt64ToByte(tx.RequestId)))
	writeMessage := notify.ClientTransactionMessage{Nonce: cpn.Nonce, Tx: tx}
	executor.writeChan <- writeMessage
}

func (executor *GameExecutor) onBlockAddSuccess(message notify.Message) {
	block := message.GetData().(types.Block)
	msg := fmt.Sprintf("height: %d, hash: %s", block.Header.Height, block.Header.Hash.ShortS())
	defer executor.logger.Info("onBlockAddSuccess, " + msg)

	transactions := block.Transactions
	if nil == transactions || 0 == len(transactions) {
		msg += ", without txs"
		return
	}

	count := 0
	for _, tx := range transactions {
		count++
		executor.tempTx.Delete(common.Uint64ToByte(tx.RequestId))
	}
	msg += fmt.Sprintf(", deleted %d txs", count)
}

func (executor *GameExecutor) coinProxyHandler(msg notify.Message) {
	cpn, ok := msg.(*notify.CoinProxyNotifyMessage)
	if !ok {
		logger.Debugf("coinProxyHandler: Message assert not ok!")
		return
	}

	message := notify.ClientTransactionMessage{
		Tx:    cpn.Tx,
		Nonce: cpn.Tx.RequestId,
	}
	executor.logger.Debugf("coinProxyHandler rcv message: %s", message.TOJSONString())
	executor.writeChan <- message
}

func (executor *GameExecutor) loop() {
	for {
		select {
		case msg := <-executor.writeChan:
			go executor.RunWrite(msg)
		}
	}
}

func (executor *GameExecutor) cleanLoop() {
	for {
		select {
		case <-executor.cleaner.C:
			nonce := service.AccountDBManagerInstance.GetLatestNonce()
			iter := executor.tempTx.NewIterator()
			for iter.Next() {
				key := iter.Key()
				currentNonce := binary.BigEndian.Uint64(key)
				if currentNonce <= nonce {
					executor.tempTx.Delete(key)
					executor.logger.Infof("clean tx: %d", currentNonce)
				}
			}
		}
	}
}

func (executor *GameExecutor) RunWrite(message notify.ClientTransactionMessage) {
	txRaw := message.Tx
	txRaw.RequestId = message.Nonce
	txRaw.SubTransactions = make([]types.UserData, 0)

	executor.logger.Infof("rcv tx with nonce: %d, txhash: %s", txRaw.RequestId, txRaw.Hash.String())
	go executor.saveTempTx(txRaw)

	accountDB, height := service.AccountDBManagerInstance.GetAccountDBByGameExecutor(message.Nonce)
	if nil == accountDB {
		return
	}
	defer service.AccountDBManagerInstance.SetLatestStateDBWithNonce(accountDB, message.Nonce, "gameExecutor", height)

	if types.TransactionTypeWrongTxNonce == txRaw.Type {
		executor.sendTransaction(&txRaw)
		return
	}

	if err := service.GetTransactionPool().VerifyTransaction(&txRaw); err != nil {
		return
	}

	executor.runTransaction(accountDB, height, txRaw)
}

func (executor *GameExecutor) saveTempTx(txRaw types.Transaction) {
	txBytes, _ := types.MarshalTransaction(&txRaw)
	executor.tempTx.Put(common.Uint64ToByte(txRaw.RequestId), txBytes)
}

func (executor *GameExecutor) runTransaction(accountDB *account.AccountDB, height uint64, txRaw types.Transaction) (bool, string) {
	txhash := txRaw.Hash.String()
	executor.logger.Debugf("run tx. hash: %s", txhash)
	defer executor.sendTransaction(&txRaw)

	if executor.isExisted(txRaw) {
		executor.logger.Errorf("tx is existed: hash: %s", txhash)
		return false, "Tx Is Existed"
	}

	processor := executors.GetTxExecutor(txRaw.Type)
	if nil == processor {
		return false, fmt.Sprintf("finish tx. wrong tx type: %d, hash: %s", txRaw.Type, txhash)
	}
	context := make(map[string]interface{})
	context["gameExecutor"] = 1
	context["chain"] = blockChainImpl

	message := ""
	result := true

	start := time.Now()
	defer executor.logger.Debugf("finish tx. result: %t, message: %s, cost time : %v, txhash: %s, requestId: %d", result, message, time.Since(start), txhash, txRaw.RequestId)

	result, message = processor.BeforeExecute(&txRaw, nil, accountDB, context)
	if !result {
		executor.logger.Errorf("finish tx. hash: %s, failed. not enough max", txhash)
		return result, message
	}
	accountDB.Prepare(txRaw.Hash, common.Hash{}, 0)
	snapshot := accountDB.Snapshot()
	result, message = processor.Execute(&txRaw, &types.BlockHeader{Height: height, CurTime: utility.GetTime()}, accountDB, context)
	if !result {
		accountDB.RevertToSnapshot(snapshot)
	} else if txRaw.Source != "" {
		accountDB.IncreaseNonce(common.HexToAddress(txRaw.Source))
	}

	// 特殊处理返回
	switch txRaw.Type {
	case types.TransactionTypePublishFT:
		appId := txRaw.Source
		if result {
			var ftSet map[string]string
			json.Unmarshal([]byte(txRaw.Data), &ftSet)
			ftSet["setId"] = message
			ftSet["creator"] = appId
			ftSet["owner"] = appId

			data, _ := json.Marshal(ftSet)
			message = string(data)
		}

		break
	case types.TransactionTypePublishNFTSet:
		if result {
			message = txRaw.Data
		}
		break
	}

	return result, message
}

func (executor *GameExecutor) sendTransaction(tx *types.Transaction) {
	if ok, err := service.GetTransactionPool().AddTransaction(tx); err != nil || !ok {
		executor.logger.Errorf("Add tx error:%s", err.Error())

		transaction := types.Transaction{Type: types.TransactionTypeWrongTxNonce, Data: tx.Hash.Hex(), RequestId: tx.RequestId}
		transaction.Hash = common.BytesToHash(common.Sha256(tx.Hash.Bytes()))
		service.GetTransactionPool().AddTransaction(&transaction)
		return
	}

	executor.logger.Debugf("Add tx success, tx: %s", tx.Hash.String())
}

func (executor *GameExecutor) isExisted(tx types.Transaction) bool {
	return service.GetTransactionPool().IsExisted(tx.Hash)
}

// read method in mimer do not return
//func (executor *GameExecutor) read(msg notify.Message) {
//	message, ok := msg.(*notify.ClientTransactionMessage)
//	if !ok {
//		executor.logger.Errorf("blockReqHandler:Message assert not ok!")
//		return
//	}
//	executor.logger.Debugf("rcv message: %v", message)
//	txRaw := message.Tx
//	//if err := service.GetTransactionPool().VerifyTransactionHash(&txRaw); err != nil {
//	//	txLogger.Errorf("Verify tx hash error!Hash:%s,error:%s", txRaw.Hash.String(), err.Error())
//	//	response := executor.makeFailedResponse(err.Error(), txRaw.SocketRequestId)
//	//	go network.GetNetInstance().SendToClientWriter(message.UserId, response, message.Nonce)
//	//	return
//	//}
//
//	var result string
//	sourceString := txRaw.Source
//	source := common.HexToAddress(sourceString)
//	gameId := txRaw.Target
//	switch txRaw.Type {
//
//	// 查询账户余额
//	case types.TransactionTypeOperatorBalance:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
//		result = service.GetBalance(source, accountDB)
//		break
//
//		// 查询主链币
//	case types.TransactionTypeGetCoin:
//		result = service.GetCoinBalance(source, txRaw.Data)
//		break
//
//		// 查询所有主链币
//	case types.TransactionTypeGetAllCoin:
//		result = service.GetAllCoinInfo(source)
//		break
//
//		// 查询FT
//	case types.TransactionTypeFT:
//		result = service.GetFTInfo(source, txRaw.Data)
//		break
//
//		// 查询用户所有FT
//	case types.TransactionTypeAllFT:
//		result = service.GetAllFT(source)
//		break
//
//		//查询特定NFT
//	case types.TransactionTypeNFT:
//		var id types.NFTID
//		err := json.Unmarshal([]byte(txRaw.Data), &id)
//		if nil == err {
//			result = service.GetNFTInfo(id.SetId, id.Id, gameId)
//		}
//		break
//
//		// 查询账户下某个游戏的所有NFT
//	case types.TransactionTypeNFTListByAddress:
//		result = service.GetAllNFT(source, gameId)
//		break
//
//		// 查询NFTSet信息
//	case types.TransactionTypeNFTSet:
//		result = service.GetNFTSet(txRaw.Data)
//		break
//
//	case types.TransactionTypeFTSet:
//		result = service.GetFTSet(txRaw.Data)
//		break
//
//	case types.TransactionTypeNFTCount:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//
//		result = strconv.Itoa(service.GetNFTCount(param["address"], param["setId"], ""))
//		break
//
//	case types.TransactionTypeNFTList:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		result = service.GetAllNFTBySetId(param["address"], param["setId"])
//		break
//
//	case types.TransactionTypeNFTGtZero:
//		accountDB := service.AccountDBManagerInstance.GetAccountDB("", true)
//		nftList := service.NFTManagerInstance.GetNFTListByAddress(source, "", accountDB)
//		resultMap := make(map[string]int, 0)
//		for _, nft := range nftList {
//			value, ok := resultMap[nft.SetID]
//			if ok {
//				value++
//			} else {
//				value = 1
//			}
//			resultMap[nft.SetID] = value
//		}
//
//		bytes, _ := json.Marshal(resultMap)
//		result = string(bytes)
//		break
//		//查询CHAIN ID
//	case types.TransactionTypeGetChainId:
//		result = service.GetChainId()
//		break
//		//查询最新块
//	case types.TransactionTypeGetBlockNumber:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		result = getBlockNumber(param["height"], param["hash"])
//		break
//		//根据高度或者HASH查询block
//	case types.TransactionTypeGetBlock:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		result = getBlock(param["height"], param["hash"])
//		break
//		//查询NONCE
//	case types.TransactionTypeGetNonce:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
//		result = service.GetNonce(source, accountDB)
//		break
//		//查询交易
//	case types.TransactionTypeGetTx:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		result = getTransaction(common.HexToHash(param["txHash"]))
//		break
//		//查询收据
//	case types.TransactionTypeGetReceipt:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		result = service.GetReceipt(common.HexToHash(param["txHash"]))
//		break
//		//查询交易数量
//	case types.TransactionTypeGetTxCount:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		result = getTransactionCount(param["height"], param["hash"])
//		break
//		//根据索引查询块中交易
//	case types.TransactionTypeGetTxFromBlock:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		result = getTransactionFromBlock(param["height"], param["hash"], param["index"])
//		break
//		//查询存储信息
//	case types.TransactionTypeGetStorage:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
//		result = service.GetStorageAt(param["address"], param["key"], accountDB)
//		break
//		//查询CODE
//	case types.TransactionTypeGetCode:
//		param := make(map[string]string, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
//		result = service.GetCode(param["address"], accountDB)
//		break
//
//	case types.TransactionTypeGetPastLogs:
//		param := make(map[string]interface{}, 0)
//		json.Unmarshal([]byte(txRaw.Data), &param)
//		from := param["fromBlock"].(uint64)
//		to := param["toBlock"].(uint64)
//		var addressList []common.Address
//		for _, address := range param["address"].([]string) {
//			addressList = append(addressList, common.HexToAddress(address))
//		}
//		topics := param["topics"].([][]string)
//		result = getPastLogs(from, to, addressList, topics)
//		break
//	}
//
//	responseId := txRaw.SocketRequestId
//
//	// reply to the client
//	go network.GetNetInstance().SendToClientReader(message.UserId, executor.makeSuccessResponse(result, responseId), message.Nonce)
//
//	return
//}
