package core

import (
	"x/src/middleware/types"
	"x/src/statemachine"
	"strconv"
	"x/src/common"
	"encoding/json"
	"x/src/network"
	"x/src/middleware/notify"
	"sync"
)

// 用于处理client websocket请求
type GameExecutor struct {
	chain *blockChain

	requestId uint64
	cond      *sync.Cond
}

//var gameExecutor *GameExecutor

func initGameExecutor(blockChainImpl *blockChain) {
	gameExecutor := GameExecutor{chain: blockChainImpl}

	// 初始值，从已有的块中获取
	gameExecutor.requestId = blockChainImpl.requestId
	gameExecutor.cond = sync.NewCond(new(sync.Mutex))

	notify.BUS.Subscribe(notify.ClientTransaction, gameExecutor.Write)

	notify.BUS.Subscribe(notify.ClientTransactionRead, gameExecutor.Read)
}

func (executor *GameExecutor) Read(msg notify.Message) {
	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}

	var result []byte
	txRaw := message.Tx
	switch txRaw.Type {

	// query balance
	case types.GetBalance:
		sub := GetSubAccount(txRaw.Source, txRaw.Target, GetBlockChain().GetAccountDB())
		if nil == sub {
			result = []byte{}
		} else {
			floatdata := float64(sub.Balance.Int64()) / 1000000000
			result = []byte(strconv.FormatFloat(floatdata, 'f', -1, 64))
		}

	case types.GetAsset:
		sub := GetSubAccount(txRaw.Source, txRaw.Target, GetBlockChain().GetAccountDB())
		if nil == sub {
			result = []byte{}
		}

		assets := sub.Assets
		if nil == assets || 0 == len(assets) {
			result = []byte{}
		} else {
			for _, asset := range assets {
				if asset.Id == string(txRaw.Data) {
					result = []byte(asset.Value)
				}
			}
		}

	case types.GetAllAssets:
		sub := GetSubAccount(txRaw.Source, txRaw.Target, GetBlockChain().GetAccountDB())
		if nil == sub {
			result = []byte{}
		}

		assets := sub.Assets
		result, _ = json.Marshal(assets)

	case types.StateMachineNonce:
		result = []byte(strconv.Itoa(statemachine.Docker.Nonce(txRaw.Target)))
	}

	// reply to the client
	network.GetNetInstance().SendToClientReader(message.UserId, network.Message{Body: result}, message.Nonce)
	return
}

func (executor *GameExecutor) Write(msg notify.Message) {

	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}

	// 校验 requestId
	requestId := message.Nonce
	if requestId <= executor.requestId {
		// 已经执行过的消息，忽略
		if common.DefaultLogger != nil {
			common.DefaultLogger.Errorf("requestId :%d skipped, current requestId: %d", requestId, executor.requestId)
		}
		return
	}

	// requestId 按序执行
	executor.cond.L.Lock()
	for ; requestId != (executor.requestId + 1); {

		// waiting until the right requestId
		if common.DefaultLogger != nil {
			common.DefaultLogger.Errorf("requestId :%d is waiting, current requestId: %d", requestId, executor.requestId)
		}
		executor.cond.Wait()
	}

	executor.runTransaction(msg)

	executor.cond.Broadcast()
	executor.cond.L.Unlock()

}
func (executor *GameExecutor) runTransaction(msg notify.Message) {
	message, _ := msg.(*notify.ClientTransactionMessage)
	txRaw := message.Tx
	txRaw.RequestId = message.Nonce

	if err := executor.sendTransaction(&txRaw); err != nil {
		return
	}

	var result []byte
	switch txRaw.Type {

	// execute state machine transaction
	case types.TransactionTypeOperatorEvent:
		outputMessage := statemachine.Docker.Process(txRaw.Target, "operator", strconv.FormatUint(txRaw.Nonce, 10), txRaw.Data)
		result, _ = json.Marshal(outputMessage)

	case types.TransactionTypeWithdraw:
		result = []byte("success")
	}

	// reply to the client
	go network.GetNetInstance().SendToClientWriter(message.UserId, network.Message{Body: result}, message.Nonce)

	// bingo
	executor.requestId++
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
