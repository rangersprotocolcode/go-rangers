package core

import (
	"x/src/common"
	"x/src/network"
	"encoding/json"
	"sync"
	"x/src/middleware/notify"
	"x/src/middleware/types"
	"strconv"
	"x/src/statemachine"
)

/**
客户端web socket 请求的返回数据结构
 */
type response struct {
	Id      string `json:"id,omitempty"`
	Status  string `json:"status,omitempty"`
	Data    string `json:"data,omitempty"`
	Version string `json:"version,omitempty"`
}

func (executor *GameExecutor) makeSuccessResponse(data string, id string) []byte {
	res := response{
		Data:    data,
		Id:      id,
		Status:  "0",
		Version: "0.1",
	}

	result, err := json.Marshal(res)
	if err != nil {
		logger.Debugf("json make success response err:%s", err.Error())
	}
	return result
}

// 用于处理client websocket请求
type GameExecutor struct {
	chain *blockChain

	requestIds map[string]uint64
	conds      sync.Map

	debug     bool // debug 为true，则不开启requestId校验
	writeChan chan notify.Message
}

func initGameExecutor(blockChainImpl *blockChain) {
	gameExecutor := GameExecutor{chain: blockChainImpl}

	// 初始值，从已有的块中获取
	gameExecutor.requestIds = blockChainImpl.requestIds
	if nil == gameExecutor.requestIds {
		gameExecutor.requestIds = make(map[string]uint64)
	}

	gameExecutor.conds = sync.Map{}

	if nil != common.GlobalConf {
		gameExecutor.debug = common.GlobalConf.GetBool("gx", "debug", true)
	} else {
		gameExecutor.debug = true
	}
	gameExecutor.writeChan = make(chan notify.Message, 100)
	notify.BUS.Subscribe(notify.ClientTransaction, gameExecutor.Write)

	notify.BUS.Subscribe(notify.ClientTransactionRead, gameExecutor.Read)
	go gameExecutor.loop()
	//notify.BUS.Subscribe(notify.BlockAddSucc, gameExecutor.onBlockAddSuccess)
}

func (executor *GameExecutor) getCond(gameId string) *sync.Cond {
	defaultValue := sync.NewCond(new(sync.Mutex))
	value, _ := executor.conds.LoadOrStore(gameId, defaultValue)

	return value.(*sync.Cond)
}

func (executor *GameExecutor) onBlockAddSuccess(message notify.Message) {
	block := message.GetData().(types.Block)
	bh := block.Header

	for key, value := range bh.RequestIds {
		if executor.requestIds[key] < value {
			executor.getCond(key).L.Lock()

			if common.DefaultLogger != nil {
				common.DefaultLogger.Errorf("upgrade %s requestId, from %d to %d", key, executor.requestIds[key], value)
			}
			executor.requestIds[key] = value

			executor.getCond(key).Broadcast()
			executor.getCond(key).L.Unlock()
		}
	}

}

func (executor *GameExecutor) Read(msg notify.Message) {
	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}

	var result string
	txRaw := message.Tx
	sourceString := txRaw.Source
	source := common.HexToAddress(sourceString)
	gameId := txRaw.Target
	accountDB := AccountDBManagerInstance.GetAccountDB(gameId)

	switch txRaw.Type {

	// query balance
	case types.TransactionTypeGetBalance:
		balance := accountDB.GetBalanceByGameId(source, gameId)
		floatdata := float64(balance.Int64()) / 1000000000
		result = strconv.FormatFloat(floatdata, 'f', -1, 64)

	case types.TransactionTypeGetAsset:
		nftId := txRaw.Data
		result = accountDB.GetNFTByGameId(source, gameId, nftId)

	case types.TransactionTypeGetAllAssets:
		bytes, _ := json.Marshal(accountDB.GetAllNFTByGameId(source, gameId))
		result = string(bytes)

	case types.TransactionTypeStateMachineNonce:
		result = strconv.FormatUint(0, 10)

	case types.TransactionTypeGetAllAsset:

		subAccountData := make(map[string]interface{})

		ftList := accountDB.GetAllFTByGameId(source, gameId)
		ftMap := make(map[string]string)
		if 0 != len(ftList) {
			for id, value := range ftList {
				ftMap[id] = strconv.FormatFloat(float64(value.Int64())/1000000000, 'f', -1, 64)
			}
		}
		subAccountData["ft"] = ftMap

		subAccountData["nft"] = accountDB.GetAllNFTByGameId(source, gameId)

		balance := accountDB.GetBalanceByGameId(source, gameId)
		floatdata := float64(balance.Int64()) / 1000000000
		subAccountData["balance"] = strconv.FormatFloat(floatdata, 'f', -1, 64)

		bytes, _ := json.Marshal(subAccountData)
		result = string(bytes)
	}

	responseId := txRaw.SocketRequestId

	// reply to the client
	go network.GetNetInstance().SendToClientReader(message.UserId, executor.makeSuccessResponse(result, responseId), message.Nonce)

	return
}

func (executor *GameExecutor) Write(msg notify.Message) {
	logger.Infof("Game executor write rcv message :%v", msg)
	executor.writeChan <- msg
}
func (executor *GameExecutor) runTransaction(txRaw types.Transaction) string {
	logger.Infof("Game executor run tx:%v", txRaw)
	if executor.isExisted(txRaw) {
		logger.Infof("Game executor is existed:%v", txRaw)
		return ""
	}

	result := ""
	switch txRaw.Type {

	// execute state machine transaction
	case types.TransactionTypeOperatorEvent:

		gameId := txRaw.Target
		accountDB := AccountDBManagerInstance.GetAccountDB(gameId)
		logger.Infof("After get account db!")

		// 已经执行过了（入块时），则不用再执行了
		if nil != TxManagerInstance.BeginTransaction(gameId, accountDB, &txRaw) || GetBlockChain().GetTransactionPool().IsGameData(txRaw.Hash) {
			// bingo
			logger.Infof("Tx is executed!")
			executor.requestIds[txRaw.Target] = executor.requestIds[txRaw.Target] + 1
			return "Tx is executed"
		}

		// 处理转账
		// 支持source地址给多人转账，包含余额，ft，nft
		// 数据格式{"address1":{"balance":"127","ft":{"name1":"189","name2":"1"},"nft":["id1","sword2"]}, "address2":{"balance":"1"}}

		//	{
		//	"address1": {
		//		"balance": "127",
		//		"ft": {
		//			"name1": "189",
		//			"name2": "1"
		//		},
		//		"nft": ["id1", "sword2"]
		//	},
		//	"address2": {
		//		"balance": "1"
		//	}
		//}

		logger.Debugf("Rcv operator event tx:%v", txRaw)
		if 0 != len(txRaw.ExtraData) {
			mm := make(map[string]types.TransferData, 0)
			if err := json.Unmarshal([]byte(txRaw.ExtraData), &mm); nil != err {
				result = "fail to transfer"
				logger.Debugf("Transfer data unmarshal error:%s", err.Error())
			} else if !changeBalances(txRaw.Target, txRaw.Source, mm, accountDB) {
				result = "fail to transfer"
				logger.Debugf("change balances  failed")
			}

		}

		var outputMessage *types.OutputMessage
		// 转账成功，调用状态机
		if result != "fail to transfer" && len(txRaw.Data) != 0 {
			// 调用状态机
			txRaw.SubTransactions = make([]string, 0)
			outputMessage = statemachine.Docker.Process(txRaw.Target, "operator", strconv.FormatUint(txRaw.Nonce, 10), txRaw.Data)
			logger.Infof("invoke state machine result:%v", outputMessage)
			if outputMessage != nil {
				result = outputMessage.Payload
			}

			GetBlockChain().GetTransactionPool().PutGameData(txRaw.Hash)
		}

		// 没有结果返回，默认出错，回滚
		if result == "fail to transfer" || len(txRaw.Data) != 0 && (outputMessage == nil || outputMessage.Status == 1) {
			logger.Infof("Roll back tx.")
			TxManagerInstance.RollBack(gameId)

			// 加入到已执行过的交易池，打包入块不会再执行这笔交易
			executor.markExecuted(&txRaw)
			if 0 == len(result) {
				result = "fail to executed"
			}
		} else {
			TxManagerInstance.Commit(gameId)
			if 0 == len(result) {
				result = "success"
			}
		}

	case types.TransactionTypeWithdraw:
		result = "success"
	}

	// 打包入块
	// 入块时不再需要调用状态机，因为已经执行过了
	executor.sendTransaction(&txRaw)

	// bingo
	executor.requestIds[txRaw.Target] = executor.requestIds[txRaw.Target] + 1

	return result
}

func (executor *GameExecutor) sendTransaction(trans *types.Transaction) error {
	//trans.SubHash = trans.GenSubHash()
	if ok, err := executor.chain.GetTransactionPool().AddTransaction(trans); err != nil || !ok {
		if nil != common.DefaultLogger {
			common.DefaultLogger.Errorf("AddTransaction not ok or error:%s", err.Error())
		}
		return err
	}

	return nil
}

func (executor *GameExecutor) markExecuted(trans *types.Transaction) {
	executor.chain.GetTransactionPool().AddExecuted(trans)
}

func (executor *GameExecutor) isExisted(tx types.Transaction) bool {
	return executor.chain.GetTransactionPool().IsExisted(tx.Hash)
}

func (executor *GameExecutor) loop() {
	for {
		select {
		case msg := <-executor.writeChan:
			logger.Infof("Game executor write chan rcv message :%v", msg)
			message, ok := msg.(*notify.ClientTransactionMessage)
			if !ok {
				logger.Debugf("GameExecutor:Write assert not ok!")
				continue
			}

			txRaw := message.Tx
			txRaw.RequestId = message.Nonce
			gameId := txRaw.Target

			// 校验交易类型
			transactionType := txRaw.Type
			if transactionType != types.TransactionTypeOperatorEvent &&
				transactionType != types.TransactionTypeWithdraw {
				logger.Debugf("GameExecutor:Write transactionType: %d, not ok!", transactionType)
				continue
			}

			// 校验 requestId
			if !executor.debug {
				requestId := message.Nonce
				if requestId <= executor.requestIds[gameId] {
					// 已经执行过的消息，忽略
					if common.DefaultLogger != nil {
						common.DefaultLogger.Errorf("%s requestId :%d skipped, current requestId: %d", gameId, requestId, executor.requestIds[gameId])
					}
					continue
				}

				// requestId 按序执行
				executor.getCond(gameId).L.Lock()
				for ; requestId != (executor.requestIds[gameId] + 1); {

					// waiting until the right requestId
					if common.DefaultLogger != nil {
						common.DefaultLogger.Errorf("%s requestId :%d is waiting, current requestId: %d", gameId, requestId, executor.requestIds[gameId])
					}
					// todo 超时放弃
					executor.getCond(gameId).Wait()
				}
			}

			result := executor.runTransaction(txRaw)
			logger.Infof("run tx result:%s,tx:%v", result, txRaw)
			// reply to the client
			response := executor.makeSuccessResponse(result, txRaw.SocketRequestId)
			go network.GetNetInstance().SendToClientWriter(message.UserId, response, message.Nonce)

			if !executor.debug {
				executor.getCond(gameId).Broadcast()
				executor.getCond(gameId).L.Unlock()
			}

		}
	}
}
