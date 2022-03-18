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
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
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
	cleaner   *time.Ticker
}

func initGameExecutor(blockChainImpl *blockChain) {
	gameExecutor := GameExecutor{chain: blockChainImpl}
	gameExecutor.logger = log.GetLoggerByIndex(log.GameExecutorLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	gameExecutor.writeChan = make(chan notify.ClientTransactionMessage, maxWriteSize)
	gameExecutor.cleaner = time.NewTicker(time.Minute * 10)

	//notify.BUS.Subscribe(notify.ClientTransactionRead, gameExecutor.read)
	notify.BUS.Subscribe(notify.ClientTransaction, gameExecutor.write)

	go gameExecutor.loop()
}

func (executor *GameExecutor) read(msg notify.Message) {
	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		executor.logger.Errorf("blockReqHandler:Message assert not ok!")
		return
	}
	executor.logger.Debugf("rcv message: %v", message)
	txRaw := message.Tx
	//if err := service.GetTransactionPool().VerifyTransactionHash(&txRaw); err != nil {
	//	txLogger.Errorf("Verify tx hash error!Hash:%s,error:%s", txRaw.Hash.String(), err.Error())
	//	response := executor.makeFailedResponse(err.Error(), txRaw.SocketRequestId)
	//	go network.GetNetInstance().SendToClientWriter(message.UserId, response, message.Nonce)
	//	return
	//}

	var result string
	sourceString := txRaw.Source
	source := common.HexToAddress(sourceString)
	//gameId := txRaw.Target
	switch txRaw.Type {

	// 查询账户余额
	case types.TransactionTypeOperatorBalance:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
		result = service.GetBalance(source, accountDB)
		break
		//查询network ID
	case types.TransactionTypeGetNetworkId:
		result = service.GetNetWorkId()
		break
		//查询CHAIN ID
	case types.TransactionTypeGetChainId:
		result = service.GetChainId()
		break
		//查询最新块
	case types.TransactionTypeGetBlockNumber:
		result = getBlockNumber()
		break
		//根据高度或者HASH查询block
	case types.TransactionTypeGetBlock:
		query := queryBlockData{}
		json.Unmarshal([]byte(txRaw.Data), &query)
		result = getBlock(query.Height, query.Hash, query.ReturnTransactionObjects)
		break
		//查询NONCE
	case types.TransactionTypeGetNonce:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
		result = service.GetNonce(source, accountDB)
		break
		//查询交易
	case types.TransactionTypeGetTx:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		result = getTransaction(common.HexToHash(param["txHash"]))
		break
		//查询收据
	case types.TransactionTypeGetReceipt:
		//param := make(map[string]string, 0)
		//json.Unmarshal([]byte(txRaw.Data), &param)
		//result = service.GetReceipt(common.HexToHash(param["txHash"]))
		break
		//查询交易数量
	case types.TransactionTypeGetTxCount:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		result = getTransactionCount(param["height"], param["hash"])
		break
		//根据索引查询块中交易
	case types.TransactionTypeGetTxFromBlock:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		result = getTransactionFromBlock(param["height"], param["hash"], param["index"])
		break
		//查询存储信息
	case types.TransactionTypeGetContractStorage:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
		result = service.GetContractStorageAt(param["address"], param["key"], accountDB)
		break
		//查询CODE
	case types.TransactionTypeGetCode:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
		result = service.GetCode(param["address"], accountDB)
		break

	}

	responseId := txRaw.SocketRequestId

	//reply to the client
	if txRaw.Type != types.TransactionTypeGetReceipt {
		go network.GetNetInstance().SendToClientReader(message.UserId, executor.makeSuccessResponse(result, responseId), message.Nonce)
	}
	return
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

func (executor *GameExecutor) loop() {
	for {
		select {
		case msg := <-executor.writeChan:
			go executor.RunWrite(msg)
		}
	}
}

func (executor *GameExecutor) RunWrite(message notify.ClientTransactionMessage) {
	txRaw := message.Tx
	txRaw.RequestId = message.Nonce
	txRaw.SubTransactions = make([]types.UserData, 0)

	executor.logger.Infof("rcv tx with nonce: %d, txhash: %s", txRaw.RequestId, txRaw.Hash.String())

	accountDB, height := service.AccountDBManagerInstance.GetAccountDBByGameExecutor(message.Nonce)
	if nil == accountDB {
		return
	}
	defer service.AccountDBManagerInstance.SetLatestStateDBWithNonce(accountDB, message.Nonce, "gameExecutor", height)

	if err := service.GetTransactionPool().VerifyTransaction(&txRaw, height); err != nil {
		executor.logger.Errorf("fail to verify tx, txhash: %s, err: %v", txRaw.Hash.String(), err.Error())
		if 0 != len(message.UserId) {
			response := executor.makeFailedResponse(err.Error(), txRaw.SocketRequestId)
			go network.GetNetInstance().SendToClientWriter(message.UserId, response, message.GateNonce)
		}
		return
	}

	result, execMessage := executor.runTransaction(accountDB, height, txRaw)
	if 0 == len(message.UserId) {
		return
	}
	executor.logger.Debugf("txhash: %s, send to user: %s, msg: %s, gatenonce: %d", txRaw.Hash.String(), message.UserId, execMessage, message.GateNonce)

	// reply to the client
	var response []byte
	if result {
		response = executor.makeSuccessResponse(execMessage, txRaw.SocketRequestId)
	} else {
		response = executor.makeFailedResponse(execMessage, txRaw.SocketRequestId)
	}
	network.GetNetInstance().SendToClientWriter(message.UserId, response, message.GateNonce)
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
	context["situation"] = "gameExecutor"
	context["refund"] = make(map[uint64]types.RefundInfoList)

	message := ""
	result := true

	start := time.Now()
	defer func() {
		executor.logger.Debugf("finish tx. result: %t, message: %s, cost time : %v, txhash: %s, requestId: %d", result, message, time.Since(start), txhash, txRaw.RequestId)
	}()

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

	return result, message
}

func (executor *GameExecutor) sendTransaction(tx *types.Transaction) {
	if ok, err := service.GetTransactionPool().AddTransaction(tx); err != nil || !ok {
		executor.logger.Errorf("Add tx error:%s", err.Error())
		return
	}

	executor.logger.Debugf("Add tx success, tx: %s", tx.Hash.String())
}

func (executor *GameExecutor) isExisted(tx types.Transaction) bool {
	return service.GetTransactionPool().IsExisted(tx.Hash)
}
