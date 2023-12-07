// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"fmt"
	"strconv"
)

type minerRefundExecutor struct {
	baseFeeExecutor
	logger log.Logger
}
type MinerRefundData struct {
	Amount  string
	MinerId string
}

func (this *minerRefundExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if nil == header || nil == transaction || nil == transaction.Sign {
		return true, ""
	}

	var minerRefundData MinerRefundData
	jsonErr := json.Unmarshal(utility.StrToBytes(transaction.Data), &minerRefundData)
	if nil != jsonErr {
		msg := fmt.Sprintf("fail to refund %s,err: %s", transaction.Data, jsonErr.Error())
		this.logger.Errorf(msg)
		return false, msg
	}

	value, err := strconv.ParseUint(minerRefundData.Amount, 10, 64)
	if err != nil {
		msg := fmt.Sprintf("fail to refund %s", transaction.Data)
		this.logger.Errorf(msg)
		return false, msg
	}
	minerId := common.FromHex(minerRefundData.MinerId)

	this.logger.Debugf("before refund, addr: %s, money: %d, minerId: %s", transaction.Source, value, transaction.Data)

	situation := context["situation"].(string)
	refundHeight, money, addr, refundErr := service.RefundManagerImpl.GetRefundStake(header.Height, minerId, common.FromHex(transaction.Source), value, accountdb, situation)
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
		refundInfo.AddRefundInfo(addr, money)
	} else {
		refundInfo = types.RefundInfoList{}
		refundInfo.AddRefundInfo(addr, money)
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

	if common.IsMainnet() && miner.Type == common.MinerTypeProposer {
		msg := fmt.Sprintf("mainnet not support Proposer")
		this.logger.Errorf(msg)
		return false, msg
	}

	if nil != header {
		miner.ApplyHeight = header.Height + common.HeightAfterStake
	}

	miner.Status = common.MinerStatusNormal

	if utility.IsEmptyByteSlice(miner.Id) {
		pubKey, err := transaction.Sign.RecoverPubkey(transaction.Hash.Bytes())
		if nil != err {
			msg := fmt.Sprintf("fail to apply miner %s, recoverPubkey failed", transaction.Data)
			this.logger.Errorf(msg)
			return false, msg
		}

		if utility.IsEmptyByteSlice(miner.Id) {
			miner.Id = pubKey.GetID()
		}
	}

	if utility.IsEmptyByteSlice(miner.Account) {
		miner.Account = common.FromHex(transaction.Source)
	}

	sourceAddr := common.HexToAddress(transaction.Source)
	res, reason := service.MinerManagerImpl.AddMiner(sourceAddr, &miner, accountdb)
	return res, reason
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

	sourceAddr := common.HexToAddress(transaction.Source)
	return service.MinerManagerImpl.AddStake(sourceAddr, miner.Id, miner.Stake, accountdb)
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

	if 0 == bytes.Compare(current.Account, miner.Account) {
		msg := fmt.Sprintf("no need to change, %s Vs %s", common.ToHex(current.Account), common.ToHex(miner.Account))
		this.logger.Errorf(msg)
		return false, msg
	}

	// check authority
	sourceBytes := common.FromHex(transaction.Source)
	if bytes.Compare(current.Account, sourceBytes) != 0 {
		msg := fmt.Sprintf("fail to auth, %s vs %s", common.ToHex(current.Account), transaction.Source)
		this.logger.Errorf(msg)
		return false, msg
	}

	// check target account
	existed := service.MinerManagerImpl.GetMinerIdByAccount(miner.Account, accountdb)
	if nil != existed {
		msg := fmt.Sprintf("cannot use account %s occupied by minerId: %s", common.ToHex(miner.Account), common.ToHex(existed))
		this.logger.Errorf(msg)
		return false, msg
	}

	msg := fmt.Sprintf("successfully change account, from %s to %s", common.ToHex(current.Account), common.ToHex(miner.Account))
	current.Account = miner.Account
	service.MinerManagerImpl.UpdateMiner(current, accountdb, false)
	this.logger.Warnf(msg)

	return true, msg
}
