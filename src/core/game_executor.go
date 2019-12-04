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
	"time"
	"x/src/middleware/log"
	"x/src/storage/account"
	"x/src/service"
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
	chain *blockChain

	requestIds map[string]uint64
	conds      sync.Map

	debug     bool // debug 为true，则不开启requestId校验
	writeChan chan notify.Message

	logger log.Logger
}

func initGameExecutor(blockChainImpl *blockChain) {
	gameExecutor := GameExecutor{chain: blockChainImpl}
	gameExecutor.logger = log.GetLoggerByIndex(log.GameExecutorLogConfig, common.GlobalConf.GetString("instance", "index", ""))

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

	gameExecutor.writeChan = make(chan notify.Message, maxWriteSize)
	notify.BUS.Subscribe(notify.ClientTransaction, gameExecutor.Write)
	go gameExecutor.loop()

	notify.BUS.Subscribe(notify.ClientTransactionRead, gameExecutor.Read)
	notify.BUS.Subscribe(notify.BlockAddSucc, gameExecutor.onBlockAddSuccess)
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
			executor.logger.Infof("upgrade %s requestId, from %d to %d", key, executor.requestIds[key], value)
			executor.requestIds[key] = value

			executor.getCond(key).Broadcast()
			executor.getCond(key).L.Unlock()
		}
	}

}

func (executor *GameExecutor) Read(msg notify.Message) {
	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		executor.logger.Errorf("blockReqHandler:Message assert not ok!")
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
		result = service.GetCoinBalance(source, txRaw.Data)
		break

		// 查询所有主链币
	case types.TransactionTypeGetAllCoin:
		result = service.GetAllCoinInfo(source)
		break

		// 查询FT
	case types.TransactionTypeFT:
		result = service.GetFTInfo(source, txRaw.Data)
		break

		// 查询用户所有FT
	case types.TransactionTypeAllFT:
		result = service.GetAllFT(source)
		break

		//查询特定NFT
	case types.TransactionTypeNFT:
		var id types.NFTID
		err := json.Unmarshal([]byte(txRaw.Data), &id)
		if nil == err {
			result = service.GetNFTInfo(id.SetId, id.Id, gameId)
		}
		break

		// 查询账户下某个游戏的所有NFT
	case types.TransactionTypeNFTListByAddress:
		result = service.GetAllNFT(source, gameId)
		break

		// 查询NFTSet信息
	case types.TransactionTypeNFTSet:
		result = service.GetNFTSet(txRaw.Data)
		break

	case types.TransactionTypeFTSet:
		result = service.GetFTSet(txRaw.Data)
		break

	case types.TransactionTypeNFTCount:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)

		result = strconv.Itoa(service.GetNFTCount(param["address"], param["setId"], ""))
		break

	case types.TransactionTypeNFTList:
		param := make(map[string]string, 0)
		json.Unmarshal([]byte(txRaw.Data), &param)
		result = service.GetAllNFTBySetId(param["address"], param["setId"])
		break
	}

	responseId := txRaw.SocketRequestId

	// reply to the client
	go network.GetNetInstance().SendToClientReader(message.UserId, executor.makeSuccessResponse(result, responseId), message.Nonce)

	return
}

func (executor *GameExecutor) Write(msg notify.Message) {
	executor.logger.Debugf("write rcv message :%v", msg)
	if len(executor.writeChan) == maxWriteSize {
		executor.logger.Errorf("write rcv message error: %v", msg)
		return
	}
	executor.writeChan <- msg
}

func (executor *GameExecutor) runTransaction(txRaw types.Transaction, requestId uint64) (bool, string) {
	txhash := txRaw.Hash.String()
	executor.logger.Debugf("run tx. hash: %s", txhash)

	if executor.isExisted(txRaw) {
		executor.logger.Errorf("tx is existed:%v", txRaw)
		return false, "Tx Is Existed"
	}

	message := ""
	result := true

	start := time.Now()

	switch txRaw.Type {
	// execute state machine transaction
	case types.TransactionTypeOperatorEvent:
		executor.logger.Debugf("begin TransactionTypeOperatorEvent. txhash: %s, appId: %s, transferInfo: %s, payload: %s", txhash, txRaw.Target, txRaw.ExtraData, txRaw.Data)

		gameId := txRaw.Target
		accountDB := service.AccountDBManagerInstance.GetAccountDB(gameId, true)
		isTransferOnly := 0 == len(gameId)

		// 只是转账
		if isTransferOnly {
			// 交易已经执行过了
			if executor.isExisted(txRaw) {
				return true, ""
			}

			snapshot := accountDB.Snapshot()
			result, ok := executor.doTransfer(txRaw, accountDB)
			if !ok {
				accountDB.RevertToSnapshot(snapshot)
			}

			return ok, result
		}

		// 已经执行过了（入块时），则不用再执行了
		if nil != service.TxManagerInstance.BeginTransaction(gameId, accountDB, &txRaw) {
			// bingo
			executor.logger.Infof("Tx is executed!")
			executor.requestIds[txRaw.Target] = executor.requestIds[txRaw.Target] + 1
			return false, "Tx Is Executed"
		}

		// 调用状态机
		// 1. 先转账
		message, result = executor.doTransfer(txRaw, accountDB)

		// 2. 转账成功，调用状态机
		var outputMessage *types.OutputMessage
		if result && len(txRaw.Data) != 0 {
			txRaw.SubTransactions = make([]types.UserData, 0)
			outputMessage = statemachine.STMManger.Process(txRaw.Target, "operator", requestId, txRaw.Data, &txRaw)
			executor.logger.Debugf("txhash %s invoke state machine. result:%v", txhash, outputMessage)
			if outputMessage != nil {
				message = outputMessage.Payload
			}

			service.GetTransactionPool().PutGameData(txRaw.Hash)
		}

		// 没有结果返回，默认出错，回滚
		if !result || len(txRaw.Data) != 0 && (outputMessage == nil || outputMessage.Status == 1) {
			executor.logger.Infof("Roll back tx.")
			service.TxManagerInstance.RollBack(gameId)

			// 加入到已执行过的交易池，打包入块不会再执行这笔交易
			executor.markExecuted(&txRaw)
			if 0 == len(message) {
				message = "Tx Execute Failed"
			}
		} else {
			service.TxManagerInstance.Commit(gameId)
		}

		executor.logger.Debugf("end TransactionTypeOperatorEvent. txhash: %s", txhash)

	case types.TransactionTypeWithdraw:
		gameId := txRaw.Target
		accountDB := service.AccountDBManagerInstance.GetAccountDB(gameId, true)

		response, ok := Withdraw(accountDB, &txRaw, false)
		if ok {
			message = response
			result = true
		} else {
			message = response
			result = false
		}
		break

	case types.TransactionTypePublishFT:
		appId := txRaw.Source
		accountDB := service.AccountDBManagerInstance.GetAccountDB("", true)
		id, ok := service.PublishFT(accountDB, &txRaw)
		if ok {
			var ftSet map[string]string
			json.Unmarshal([]byte(txRaw.Data), &ftSet)
			ftSet["setId"] = id
			ftSet["creator"] = appId
			ftSet["owner"] = appId

			data, _ := json.Marshal(ftSet)
			message = string(data)
			result = true
		} else {
			message = id
			result = false
		}
		break

	case types.TransactionTypePublishNFTSet:
		appId := txRaw.Source
		accountDB := service.AccountDBManagerInstance.GetAccountDB(appId, true)
		flag, response := service.PublishNFTSet(accountDB, &txRaw)
		if flag {
			message = txRaw.Data
			result = true
		} else {
			message = response
			result = false
		}
		break

	case types.TransactionTypeMintFT:
		accountDB := service.AccountDBManagerInstance.GetAccountDB("", true)
		flag, response := service.MintFT(accountDB, &txRaw)

		result = flag
		message = response
		break

	case types.TransactionTypeMintNFT:
		accountDB := service.AccountDBManagerInstance.GetAccountDB("", true)
		flag, response := service.MintNFT(accountDB, &txRaw)
		message = response
		result = flag
		break

	case types.TransactionTypeShuttleNFT:
		accountDB := service.AccountDBManagerInstance.GetAccountDB("", true)
		ok, response := service.ShuttleNFT(accountDB, &txRaw)
		message = response
		result = ok
		break
	}

	// 打包入块
	// 入块时不再需要调用状态机，因为已经执行过了
	executor.sendTransaction(&txRaw)

	// bingo
	executor.requestIds[txRaw.Target] = executor.requestIds[txRaw.Target] + 1

	executor.logger.Debugf("finish tx. result: %t, message: %s, cost time : %v, txhash: %s, requestId: %d", result, message, time.Since(start), txhash, executor.requestIds[txRaw.Target])
	return result, message
}

// 处理转账
// 支持source地址给多人转账，包含余额，ft，nft
// 数据格式{"address1":{"balance":"127","ft":{"name1":"189","name2":"1"},"nft":["id1","sword2"]}, "address2":{"balance":"1"}}
//{
//	"address1": {
//		"bnt": {
//          "ETH.ETH":"0.008",
//          "NEO.CGAS":"100"
//      },
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
func (executor *GameExecutor) doTransfer(txRaw types.Transaction, accountDB *account.AccountDB) (string, bool) {
	if 0 == len(txRaw.ExtraData) {
		return "", true
	}

	mm := make(map[string]types.TransferData, 0)
	if err := json.Unmarshal([]byte(txRaw.ExtraData), &mm); nil != err {
		executor.logger.Errorf("Transfer data unmarshal error:%s", err.Error())
		return "Transfer Bad Format", false
	}

	// todo: mm条目数校验：调用状态机的，mm条目数只能有一个；纯转账，允许多个
	response, ok := service.ChangeAssets(txRaw.Source, mm, accountDB)
	if !ok {
		executor.logger.Errorf("change balances failed")
	}

	return response, ok

}

func (executor *GameExecutor) sendTransaction(tx *types.Transaction) {
	if ok, err := service.GetTransactionPool().AddTransaction(tx); err != nil || !ok {
		executor.logger.Errorf("Add tx error:%s", err.Error())
		return
	}

	executor.logger.Debugf("Add tx success, tx: %s", tx.Hash.String())
}

func (executor *GameExecutor) markExecuted(trans *types.Transaction) {
	service.GetTransactionPool().AddExecuted(trans)
}

func (executor *GameExecutor) isExisted(tx types.Transaction) bool {
	return service.GetTransactionPool().IsExisted(tx.Hash)
}

func (executor *GameExecutor) loop() {
	for {
		select {
		case msg := <-executor.writeChan:
			executor.RunNotify(msg)
		}
	}
}

func (executor *GameExecutor) RunNotify(msg notify.Message) {
	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		executor.logger.Errorf("GameExecutor:Write assert not ok!")
		return
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
			executor.logger.Errorf("%s requestId :%d skipped, current requestId: %d", gameId, requestId, executor.requestIds[gameId])
			return
		}

		// requestId 按序执行
		executor.getCond(gameId).L.Lock()
		for ; requestId != (executor.requestIds[gameId] + 1); {

			// waiting until the right requestId
			executor.logger.Infof("%s requestId :%d is waiting, current requestId: %d", gameId, requestId, executor.requestIds[gameId])

			// todo 超时放弃
			executor.getCond(gameId).Wait()
		}
	}

	result, execMessage := executor.runTransaction(txRaw, message.Nonce)

	// reply to the client
	var response []byte
	if result {
		response = executor.makeSuccessResponse(execMessage, txRaw.SocketRequestId)
	} else {
		response = executor.makeFailedResponse(execMessage, txRaw.SocketRequestId)
	}
	go network.GetNetInstance().SendToClientWriter(message.UserId, response, message.Nonce)

	if !executor.debug {
		executor.getCond(gameId).Broadcast()
		executor.getCond(gameId).L.Unlock()
	}
}
