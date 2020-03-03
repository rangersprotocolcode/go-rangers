package core

import (
	"x/src/storage/account"
	"x/src/middleware/types"
	"strconv"
	"x/src/common"
	"encoding/json"
)

type minerRefundExecutor struct {
}

func (this *minerRefundExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	value, err := strconv.ParseUint(transaction.Data, 10, 64)
	if err != nil {
		logger.Errorf("fail to refund %s", transaction.Data)
		return false
	}

	minerId := common.FromHex(transaction.Source)
	logger.Debugf("before refund, addr: %s, money: %d, minerId: %v", transaction.Source, value, minerId)
	refundHeight, money, refundErr := RefundManagerImpl.GetRefundStake(header.Height, minerId, value, accountdb)
	if refundErr != nil {
		logger.Errorf("fail to refund %s, err: %s", transaction.Data, refundErr.Error())
		return false
	}

	logger.Infof("add refund, minerId: %s, height: %d, money: %d", transaction.Source, refundHeight, money)
	refundInfos := getRefundInfo(context)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	return true
}

type minerApplyExecutor struct {
}

func (this *minerApplyExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	data := transaction.Data
	var miner types.Miner
	err := json.Unmarshal([]byte(data), &miner)
	if err != nil {
		logger.Errorf("json Unmarshal error, %s", err.Error())
		return false
	}

	miner.ApplyHeight = header.Height + common.HeightAfterStake
	if isEmptyByteSlice(miner.Id) {
		miner.Id = common.FromHex(transaction.Source)
	}
	return MinerManagerImpl.AddMiner(common.HexToAddress(transaction.Source), &miner, accountdb)
}

type minerAddExecutor struct {
}

func (this *minerAddExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	data := transaction.Data
	var miner types.Miner
	err := json.Unmarshal([]byte(data), &miner)
	if err != nil {
		logger.Errorf("json Unmarshal error, %s", err.Error())
		return false
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