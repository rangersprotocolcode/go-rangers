package core

import (
	"x/src/middleware/types"
	"x/src/statemachine"
	"strconv"
	"x/src/common"
	"encoding/json"
	"x/src/network"
	"x/src/middleware/notify"
)

type GameExecutor struct {
	chain *blockChain
}

//var gameExecutor *GameExecutor

func initGameExecutor(blockChainImpl *blockChain) {
	gameExecutor := GameExecutor{chain: blockChainImpl}

	notify.BUS.Subscribe(notify.ClientTransaction, gameExecutor.Tx)
}

func (executor *GameExecutor) Tx(msg notify.Message) {

	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}

	var result []byte
	txRaw := message.Tx
	switch txRaw.Type {

	// execute state machine transaction
	case types.TransactionTypeOperatorEvent:
		outputMessage := statemachine.Docker.Process(txRaw.Target, "operator", strconv.FormatUint(txRaw.Nonce, 10), txRaw.Data)
		result, _ = json.Marshal(outputMessage)

		if err := executor.sendTransaction(&txRaw); err != nil {
			return
		}

		executor.chain.GetTransactionPool().AddExecuted(&txRaw)

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
	case types.TransactionTypeWithdraw:
		if err := executor.sendTransaction(&txRaw); err != nil {
			return
		}
		result = []byte("success")
	case types.TransactionTypeAssetOnChain:
		if err := executor.sendTransaction(&txRaw); err != nil {
			return
		}
	}

	// reply to the client
	network.GetNetInstance().SendToClient(message.UserId, network.Message{Body: result}, message.Nonce)
	return

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
