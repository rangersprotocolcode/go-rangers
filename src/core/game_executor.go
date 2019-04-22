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

	// execute state machine transaction
	if txRaw.Type == types.TransactionTypeOperatorEvent {
		payload := string(txRaw.Data)
		outputMessage := statemachine.Docker.Process(txRaw.Target, "operator", strconv.FormatUint(txRaw.Nonce, 10), payload)

		result, _ = json.Marshal(outputMessage)

	}

	if err := executor.sendTransaction(&txRaw); err != nil {
		return
	}

	if txRaw.Type == types.TransactionTypeOperatorEvent {
		executor.chain.GetTransactionPool().AddExecuted(&txRaw)
		network.GetNetInstance().SendToClient(message.UserId, network.Message{Body: result}, message.Nonce)
	}

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
