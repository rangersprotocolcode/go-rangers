// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"strconv"
)

type minerRefundExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

func (this *minerRefundExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if nil == header || nil == transaction || nil == transaction.Sign {
		return true, ""
	}

	value, err := strconv.ParseUint(transaction.Data, 10, 64)
	if err != nil {
		msg := fmt.Sprintf("fail to refund %s", transaction.Data)
		this.logger.Errorf(msg)
		return false, msg
	}

	pubKey, err := transaction.Sign.RecoverPubkey(transaction.Hash.Bytes())
	if nil != err {
		msg := fmt.Sprintf("fail to refund %s, recoverPubkey failed", transaction.Data)
		this.logger.Errorf(msg)
		return false, msg
	}
	minerId := pubKey.GetID()
	this.logger.Debugf("before refund, addr: %s, money: %d, minerId: %v", transaction.Source, value, minerId)

	situation := context["situation"].(string)
	refundHeight, money, refundErr := service.RefundManagerImpl.GetRefundStake(header.Height, minerId, value, accountdb, situation)
	if refundErr != nil {
		msg := fmt.Sprintf("fail to refund %s, err: %s", transaction.Data, refundErr.Error())
		this.logger.Errorf(msg)
		return false, msg
	}

	msg := fmt.Sprintf("refund, minerId: %s, height: %d, money: %d", transaction.Source, refundHeight, money)
	this.logger.Infof(msg)
	refundInfos := types.GetRefundInfo(context)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = types.RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	return true, msg
}

type minerApplyExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

func (this *minerApplyExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	data := transaction.Data
	var miner types.Miner
	err := json.Unmarshal([]byte(data), &miner)
	if err != nil {
		msg := fmt.Sprintf("json Unmarshal error, %s", err.Error())
		this.logger.Errorf(msg)
		return false, msg
	}

	if nil != header {
		miner.ApplyHeight = header.Height + common.HeightAfterStake
	}

	miner.Status = common.MinerStatusNormal

	if utility.IsEmptyByteSlice(miner.Id) || utility.IsEmptyByteSlice(miner.Account) {
		pubKey, err := transaction.Sign.RecoverPubkey(transaction.Hash.Bytes())
		if nil != err {
			msg := fmt.Sprintf("fail to apply miner %s, recoverPubkey failed", transaction.Data)
			this.logger.Errorf(msg)
			return false, msg
		}

		if utility.IsEmptyByteSlice(miner.Id) {
			miner.Id = pubKey.GetID()
		}

		if utility.IsEmptyByteSlice(miner.Account) {
			miner.Account = pubKey.GetAddress().Bytes()
		}
	}

	return service.MinerManagerImpl.AddMiner(common.HexToAddress(transaction.Source), &miner, accountdb)
}

type minerAddExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

func (this *minerAddExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	data := transaction.Data
	var miner types.Miner
	err := json.Unmarshal([]byte(data), &miner)
	if err != nil {
		msg := fmt.Sprintf("json Unmarshal error, %s", err.Error())
		this.logger.Errorf(msg)
		return false, msg
	}

	if utility.IsEmptyByteSlice(miner.Id) {
		pubKey, err := transaction.Sign.RecoverPubkey(transaction.Hash.Bytes())
		if nil != err {
			msg := fmt.Sprintf("fail to refund %s, recoverPubkey failed", transaction.Data)
			this.logger.Errorf(msg)
			return false, msg
		}
		miner.Id = pubKey.GetID()
	}

	return service.MinerManagerImpl.AddStake(common.HexToAddress(transaction.Source), miner.Id, miner.Stake, accountdb)
}

type minerChangeAccountExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

func (this *minerChangeAccountExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	data := transaction.Data
	var miner types.Miner
	err := json.Unmarshal([]byte(data), &miner)
	if err != nil {
		msg := fmt.Sprintf("json Unmarshal error, %s", err.Error())
		this.logger.Errorf(msg)
		return false, msg
	}

	current := service.MinerManagerImpl.GetMiner(miner.Id, accountdb)
	if nil == current {
		msg := fmt.Sprintf("fail to getMiner, %s", common.ToHex(miner.Id))
		this.logger.Errorf(msg)
		return false, msg
	}

	// check authority
	if bytes.Compare(current.Account, common.Hex2Bytes(transaction.Source)) != 0 {
		msg := fmt.Sprintf("fail to auth, %s vs %s", common.ToHex(current.Account), transaction.Source)
		this.logger.Errorf(msg)
		return false, msg
	}

	msg := fmt.Sprintf("successfully change account, from %s to %s", common.ToHex(current.Account), common.ToHex(miner.Account))
	current.Account = miner.Account
	service.MinerManagerImpl.UpdateMiner(current, accountdb, false)
	this.logger.Warnf(msg)
	return true, msg
}
