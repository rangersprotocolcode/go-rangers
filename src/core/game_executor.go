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

// 客户端web socket 请求的返回数据结构
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
	switch txRaw.Type {

	// 查询主链币
	case types.TransactionTypeGetCoin:
		result = GetCoinBalance(source, txRaw.Data)
		break

		// 查询所有主链币
	case types.TransactionTypeGetAllCoin:
		result = GetAllCoinInfo(source)
		break

		// 查询FT
	case types.TransactionTypeFT:
		result = GetFTInfo(source, txRaw.Data)
		break

		// 查询用户所有FT
	case types.TransactionTypeAllFT:
		result = GetAllFT(source)
		break

		//查询特定NFT
	case types.TransactionTypeNFT:
		var id types.NFTID
		err := json.Unmarshal([]byte(txRaw.Data), &id)
		if nil == err {
			result = GetNFTInfo(id.SetId, id.Id, gameId)
		}
		break

		// 查询账户下某个游戏的所有NFT
	case types.TransactionTypeNFTListByAddress:
		result = GetAllNFT(source, gameId)
		break

		// 查询NFTSet信息
	case types.TransactionTypeNFTSet:
		result = GetNFTSet(txRaw.Data)
		break

	case types.TransactionTypeFTSet:
		result = GetFTSet(txRaw.Data)
		break

	case types.TransactionTypeNFTCount:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)

		result = strconv.Itoa(GetNFTCount(param["address"], param["setId"], ""))
		break

	case types.TransactionTypeNFTList:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		result = GetAllNFTBySetId(param["address"], param["setId"])
		break
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
		accountDB := AccountDBManagerInstance.GetAccountDB(gameId, true)
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
		//		"nft": [{"setId":"suit1","id":"xizhuang"},
		//              {"setId":"gun","id":"rifle"}
		// 				]
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
			} else {
				response, ok := changeAssets(txRaw.Target, txRaw.Source, mm, accountDB)
				if !ok {
					result = "fail to transfer"
					logger.Debugf("change balances  failed")
				} else {
					result = response
				}
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
		gameId := txRaw.Target
		accountDB := AccountDBManagerInstance.GetAccountDB(gameId, false)

		response, ok := Withdraw(accountDB, &txRaw, false)
		if ok {
			result = response
		} else {
			result = "fail"
		}
		break
	case types.TransactionTypePublishFT:
		appId := txRaw.Source
		accountDB := AccountDBManagerInstance.GetAccountDB("", true)
		id, ok := PublishFT(accountDB, &txRaw)
		if ok {
			var ftSet map[string]string
			json.Unmarshal([]byte(txRaw.Data), &ftSet)
			ftSet["setId"] = id
			ftSet["creator"] = appId
			ftSet["owner"] = appId

			data, _ := json.Marshal(ftSet)
			result = string(data)
		} else {
			result = "fail"
		}

		break

	case types.TransactionTypePublishNFTSet:
		appId := txRaw.Source
		accountDB := AccountDBManagerInstance.GetAccountDB(appId, false)
		flag := PublishNFTSet(accountDB, &txRaw)
		if flag {
			result = txRaw.Data

		} else {
			result = "fail"
		}
		break
	case types.TransactionTypeMintFT:
		accountDB := AccountDBManagerInstance.GetAccountDB("", true)
		flag := MintFT(accountDB, &txRaw)
		if flag {
			result = "success"
		} else {
			result = "fail"
		}
		break

	case types.TransactionTypeShuttleNFT:
		accountDB := AccountDBManagerInstance.GetAccountDB("", true)
		ok := ShuttleNFT(accountDB, &txRaw)
		if ok {
			result = "success"
		} else {
			result = "fail"
		}
		break

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
			//gameId := txRaw.Target
			gameId := "fixed"

			// 校验交易类型
			//transactionType := txRaw.Type
			//if transactionType != types.TransactionTypeOperatorEvent &&
			//	transactionType != types.TransactionTypeWithdraw && transactionType != types.TransactionTypePublishFT {
			//	logger.Debugf("GameExecutor:Write transactionType: %d, not ok!", transactionType)
			//	continue
			//}

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
