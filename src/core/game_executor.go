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
	"com.tuntun.rocket/node/src/middleware"
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
	Id      string `json:"id"`
	Status  string `json:"status"`
	Data    string `json:"data"`
	Message string `json:"message"`
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

// GameExecutor 用于处理client websocket请求
type GameExecutor struct {
	chain     *blockChain
	writeChan chan byte
	logger    log.Logger
	cleaner   *time.Ticker
}

func initGameExecutor(blockChainImpl *blockChain) {
	gameExecutor := GameExecutor{chain: blockChainImpl}
	gameExecutor.logger = log.GetLoggerByIndex(log.GameExecutorLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	gameExecutor.writeChan = make(chan byte, 1)
	gameExecutor.cleaner = time.NewTicker(time.Minute * 10)

	notify.BUS.Subscribe(notify.ClientTransactionRead, gameExecutor.read)
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

	var result string
	sourceString := txRaw.Source
	source := common.HexToAddress(sourceString)
	//gameId := txRaw.Target
	switch txRaw.Type {

	case types.TransactionTypeOperatorBalance:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		accountDB := getAccountDBByHashOrHeight(param["height"], param["hash"])
		result = service.GetRawBalance(source, accountDB)
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
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		result = service.GetReceipt(common.HexToHash(param["txHash"]))
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

	case types.TransactionTypeGetPastLogs:
		query := types.FilterCriteria{}
		err := json.Unmarshal([]byte(txRaw.Data), &query)
		if err != nil {
			executor.logger.Debugf("FilterCriteria unmarshal error", err)
			break
		}
		executor.logger.Debugf("rcv TransactionTypeGetPastLogs:%d,%d,%v,%v", query.FromBlock, query.ToBlock, query.Addresses, query.Topics)
		result = getPastLogs(query)
		break
		//call vm
	case types.TransactionTypeCallVM:
		data := callVMData{}
		err := json.Unmarshal([]byte(txRaw.Data), &data)
		if err != nil {
			executor.logger.Debugf("callVMData unmarshal error", err)
			break
		}
		executor.logger.Debugf("rcv TransactionTypeCallVM:%s,%s,%s,%s,%v,%s,%v,%v", data.Height, data.Hash, data.From, data.To, data.Value, data.Data, data.Gas, data.GasPrice)
		result = executor.callVM(data)
		break
	}

	responseId := txRaw.SocketRequestId

	//reply to the client
	go network.GetNetInstance().SendToClientReader(message.UserId, executor.makeSuccessResponse(result, responseId), message.Nonce)
}

// 通过流量控制，初步处理交易信息
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
	service.AccountDBManagerInstance.WaitingTxs.HeapPush(message)
	executor.writeChan <- 1
}

// 调度，处理队列里的交易
func (executor *GameExecutor) loop() {
	for {
		select {
		// 来新交易来
		case <-executor.writeChan:
			go executor.RunWrite()
		// 上块完成
		case <-service.AccountDBManagerInstance.NewTxs:
			go executor.RunWrite()
		}
	}
}

func (executor *GameExecutor) RunWrite() {
	service.AccountDBManagerInstance.WaitingTxs.TryPop(func(item *service.Item) {
		message := item.Value
		txRaw := message.Tx
		txRaw.RequestId = message.Nonce
		txRaw.SubTransactions = make([]types.UserData, 0)

		executor.logger.Infof("rcv tx with nonce: %d, txhash: %s", txRaw.RequestId, txRaw.Hash.String())

		//accountDB, height := service.AccountDBManagerInstance.GetAccountDBByGameExecutor(message.Nonce)
		//if nil == accountDB {
		//	return
		//}

		accountDB, height := service.AccountDBManagerInstance.LatestStateDB, service.AccountDBManagerInstance.Height
		if err := service.GetTransactionPool().VerifyTransaction(&txRaw, height); err != nil {
			//service.AccountDBManagerInstance.SetLatestStateDBWithNonce(accountDB, message.Nonce, "gameExecutor", height)

			executor.logger.Errorf("fail to verify tx, txhash: %s, err: %v", txRaw.Hash.String(), err.Error())
			if 0 != len(message.UserId) {
				response := executor.makeFailedResponse(err.Error(), txRaw.SocketRequestId)
				go network.GetNetInstance().SendToClientWriter(message.UserId, response, message.GateNonce)
			}
			return
		}

		result, execMessage := executor.runTransaction(accountDB, height, txRaw)
		//service.AccountDBManagerInstance.SetLatestStateDBWithNonce(accountDB, message.Nonce, "gameExecutor", height)
		executor.sendTransaction(&txRaw)

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

		if 0 != message.GateNonce {
			network.GetNetInstance().SendToClientWriter(message.UserId, response, message.GateNonce)
		}

	})

}

func (executor *GameExecutor) runTransaction(accountDB *account.AccountDB, height uint64, txRaw types.Transaction) (bool, string) {
	txhash := txRaw.Hash.String()
	executor.logger.Debugf("run tx. hash: %s", txhash)

	if executor.isExisted(txRaw) {
		executor.logger.Errorf("tx is existed: hash: %s", txhash)
		return false, "Tx Is Existed"
	}

	if common.IsProposal006() && !common.IsProposal007() {
		accountDB.IncreaseNonce(common.HexToAddress(txRaw.Source))
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
	} else {
		accountDB.Prepare(txRaw.Hash, common.Hash{}, 0)
		snapshot := accountDB.Snapshot()
		result, message = processor.Execute(&txRaw, &types.BlockHeader{Height: height, CurTime: utility.GetTime()}, accountDB, context)
		if !result {
			accountDB.RevertToSnapshot(snapshot)
		} else if txRaw.Source != "" {
			if !common.IsProposal006() {
				accountDB.IncreaseNonce(common.HexToAddress(txRaw.Source))
			}
		}
	}

	if common.IsProposal007() {
		if !(types.IsContractTx(txRaw.Type) && result) {
			nonce := accountDB.GetNonce(common.HexToAddress(txRaw.Source))
			accountDB.SetNonce(common.HexToAddress(txRaw.Source), nonce+1)
		}
	}
	message = adaptReturnMessage(txRaw, message)
	return result, message
}

func (executor *GameExecutor) sendTransaction(tx *types.Transaction) {
	middleware.LockBlockchain("AddTransaction_game")
	defer middleware.UnLockBlockchain("AddTransaction_game")

	if ok, err := service.GetTransactionPool().AddTransaction(tx); err != nil || !ok {
		executor.logger.Errorf("Add tx error:%s", err.Error())
		return
	}

	executor.logger.Debugf("Add tx success, tx: %s", tx.Hash.String())
}

func (executor *GameExecutor) isExisted(tx types.Transaction) bool {
	return service.GetTransactionPool().IsExisted(tx.Hash)
}

func adaptReturnMessage(tx types.Transaction, message string) string {
	if tx.Type != types.TransactionTypeContract {
		return message
	}

	type executeResultAdaptedData struct {
		ContractAddress string `json:"contractAddress,omitempty"`

		Result string `json:"result,omitempty"`

		ExecuteResult string `json:"executeResult,omitempty"`

		Logs []*types.Log `json:"logs,omitempty"`
	}

	var returnData = executeResultAdaptedData{}
	err := json.Unmarshal([]byte(message), &returnData)
	if err != nil {
		return message
	}
	returnData.ExecuteResult = returnData.Result
	returnData.Result = tx.Hash.String()
	jsonBytes, err := json.Marshal(returnData)
	if err != nil {
		return message
	}
	return string(jsonBytes)
}
