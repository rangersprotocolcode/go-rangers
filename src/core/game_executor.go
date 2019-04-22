package core

import (
	"x/src/middleware/types"
	"x/src/statemachine"
	"strconv"
	"x/src/common"
	"encoding/json"
)

type GameExecutor struct {
}

func (executor *GameExecutor) Tx(txRaw types.Transaction) ([]byte, error) {
	var message []byte
	// execute state machine transaction
	if txRaw.Type == types.TransactionTypeOperatorEvent {
		payload := string(txRaw.Data)
		outputMessage := statemachine.Docker.Process(txRaw.Target, "operator", strconv.FormatUint(txRaw.Nonce, 10), payload)

		message, _ = json.Marshal(outputMessage)

	}

	if err := executor.sendTransaction(&txRaw); err != nil {
		return nil, err
	}

	if txRaw.Type == types.TransactionTypeOperatorEvent {
		GetBlockChain().GetTransactionPool().AddExecuted(&txRaw)
	}

	return message, nil

}

func (executor *GameExecutor) sendTransaction(trans *types.Transaction) error {
	if ok, err := GetBlockChain().GetTransactionPool().AddTransaction(trans); err != nil || !ok {
		common.DefaultLogger.Errorf("AddTransaction not ok or error:%s", err.Error())
		return err
	}

	return nil
}
