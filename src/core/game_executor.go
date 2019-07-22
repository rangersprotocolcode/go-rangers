package core

import (
	"x/src/middleware/types"
	"strconv"
	"x/src/common"
	"encoding/json"
	"x/src/network"
	"x/src/middleware/notify"
	"sync"
	"x/src/statemachine"
)

// 用于处理client websocket请求
type GameExecutor struct {
	chain *blockChain

	requestIds map[string]uint64
	conds      sync.Map

	debug     bool // debug 为true，则不开启requestId校验
	writeChan chan notify.Message
}

type response struct {
	Data    string
	Hash    string
	Status  byte
	Version string
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
	gameId := txRaw.Target
	sub := GetSubAccount(txRaw.Source, gameId, AccountDBManagerInstance.GetAccountDB(gameId))
	if nil == sub {
		result = ""
	} else {
		switch txRaw.Type {

		// query balance
		case types.GetBalance:
			floatdata := float64(sub.Balance.Int64()) / 1000000000
			result = strconv.FormatFloat(floatdata, 'f', -1, 64)

		case types.GetAsset:
			assets := sub.Assets
			if nil == assets || 0 == len(assets) {
				result = ""
			} else {
				result = assets[txRaw.Data]
			}

		case types.GetAllAssets:
			assets := sub.Assets
			bytes, _ := json.Marshal(assets)
			result = string(bytes)

		case types.StateMachineNonce:
			result = strconv.FormatUint(sub.Nonce, 10)

		}
	}

	responseId := txRaw.SocketRequestId

	// reply to the client
	go network.GetNetInstance().SendToClientReader(message.UserId, network.Message{Body: executor.makeSuccessResponse(result, responseId)}, message.Nonce)

	return
}

func (executor *GameExecutor) makeSuccessResponse(bytes string, hash string) []byte {
	res := response{
		Data:    bytes,
		Hash:    hash,
		Status:  0,
		Version: "0.1",
	}

	data, err := json.Marshal(res)
	if err != nil {
		logger.Debugf("json make success response err:%s", err.Error())
	}
	return data
}

func (executor *GameExecutor) Write(msg notify.Message) {
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

		// 处理转账
		// 支持多人转账{"address1":"value1", "address2":"value2"}
		gameId := txRaw.Target
		accountDB := AccountDBManagerInstance.GetAccountDB(gameId)

		// 已经执行过了（入块时），则不用再执行了
		if nil != TxManagerInstance.BeginTransaction(gameId, accountDB, &txRaw) || GetBlockChain().GetTransactionPool().IsGameData(txRaw.Hash) {
			// bingo
			executor.requestIds[txRaw.Target] = executor.requestIds[txRaw.Target] + 1
			return ""
		}

		if 0 != len(txRaw.ExtraData) {
			mm := make(map[string]string, 0)
			if err := json.Unmarshal([]byte(txRaw.ExtraData), &mm); nil != err {
				result = "fail to transfer"
				break
			}

			if !changeBalances(txRaw.Target, txRaw.Source, mm, accountDB) {
				result = "fail to transfer"
				break
			}

		}

		var outputMessage *types.OutputMessage
		// 转账成功，调用状态机
		if result != "fail to transfer" {
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
		if 0 == len(result) || result == "fail to transfer" || outputMessage == nil || outputMessage.Status == 1 {
			TxManagerInstance.RollBack(gameId)

			// 加入到已执行过的交易池，打包入块不会再执行这笔交易
			executor.markExecuted(&txRaw)
			if 0 == len(result) {
				result = "fail to executed"
			}
		} else {
			TxManagerInstance.Commit(gameId)
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
			message, ok := msg.(*notify.ClientTransactionMessage)
			if !ok {
				logger.Debugf("GameExecutor:Write assert not ok!")
				return
			}

			txRaw := message.Tx
			txRaw.RequestId = message.Nonce
			gameId := txRaw.Target

			// 校验交易类型
			transactionType := txRaw.Type
			if transactionType != types.TransactionTypeOperatorEvent &&
				transactionType != types.TransactionTypeWithdraw &&
				transactionType != types.TransactionTypeAssetOnChain {
				logger.Debugf("GameExecutor:Write transactionType: %d, not ok!", transactionType)
				return
			}

			// 校验 requestId
			if !executor.debug {
				requestId := message.Nonce
				if requestId <= executor.requestIds[gameId] {
					// 已经执行过的消息，忽略
					if common.DefaultLogger != nil {
						common.DefaultLogger.Errorf("%s requestId :%d skipped, current requestId: %d", gameId, requestId, executor.requestIds[gameId])
					}
					return
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
			if 0 != len(result) {
				response := executor.makeSuccessResponse(result, txRaw.SocketRequestId)
				go network.GetNetInstance().SendToClientWriter(message.UserId, network.Message{Body: response}, message.Nonce)

			}

			if !executor.debug {
				executor.getCond(gameId).Broadcast()
				executor.getCond(gameId).L.Unlock()
			}

		}
	}
}
