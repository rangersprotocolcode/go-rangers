package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
)

type minerRefundExecutor struct {
}

func (this *minerRefundExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if err := service.GetTransactionPool().ProcessFee(*transaction, accountdb); err != nil {
		return false, "not enough fee"
	}

	value, err := strconv.ParseUint(transaction.Data, 10, 64)
	if err != nil {
		msg := fmt.Sprintf("fail to refund %s", transaction.Data)
		logger.Errorf(msg)
		return false, msg
	}

	minerId := common.FromHex(transaction.Source)
	logger.Debugf("before refund, addr: %s, money: %d, minerId: %v", transaction.Source, value, minerId)
	refundHeight, money, refundErr := RefundManagerImpl.GetRefundStake(header.Height, minerId, value, accountdb)
	if refundErr != nil {
		msg := fmt.Sprintf("fail to refund %s, err: %s", transaction.Data, refundErr.Error())
		logger.Errorf(msg)
		return false, msg
	}

	msg := fmt.Sprintf("refund, minerId: %s, height: %d, money: %d", transaction.Source, refundHeight, money)
	logger.Infof(msg)
	refundInfos := getRefundInfo(context)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	return true, msg
}

type minerApplyExecutor struct {
}

func (this *minerApplyExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if err := service.GetTransactionPool().ProcessFee(*transaction, accountdb); err != nil {
		return false, ""
	}

	data := transaction.Data
	var miner types.Miner
	err := json.Unmarshal([]byte(data), &miner)
	if err != nil {
		msg := fmt.Sprintf("json Unmarshal error, %s", err.Error())
		logger.Errorf(msg)
		return false, msg
	}

	miner.ApplyHeight = header.Height + common.HeightAfterStake
	if isEmptyByteSlice(miner.Id) {
		miner.Id = common.FromHex(transaction.Source)
	}
	return MinerManagerImpl.AddMiner(common.HexToAddress(transaction.Source), &miner, accountdb)
}

type minerAddExecutor struct {
}

func (this *minerAddExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if err := service.GetTransactionPool().ProcessFee(*transaction, accountdb); err != nil {
		return false, "not enough fee"
	}

	data := transaction.Data
	var miner types.Miner
	err := json.Unmarshal([]byte(data), &miner)
	if err != nil {
		msg := fmt.Sprintf("json Unmarshal error, %s", err.Error())
		logger.Errorf(msg)
		return false, msg
	}

	if isEmptyByteSlice(miner.Id) {
		miner.Id = common.FromHex(transaction.Source)
	}
	return MinerManagerImpl.AddStake(common.HexToAddress(transaction.Source), miner.Id, miner.Stake, accountdb)
}

func isEmptyByteSlice(data []byte) bool {
	if nil == data || 0 == len(data) {
		return true
	}

	for _, item := range data {
		if 0 != item {
			return false
		}
	}

	return true
}
