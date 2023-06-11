package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/vm"
	"math/big"
	"strconv"
)

var (
	rewardLog     = log.GetLoggerByIndex(log.RewardLogConfig, strconv.Itoa(common.InstanceIndex))
	callerAddress = common.HexToAddress("0x1111111111111111111111111111111111111111")
)

const padding = "0000000000000000000000000000000000000000000000000000000000000060"

func (executor *VMExecutor) calcSubReward() {
	header := executor.block.Header

	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }
	vmCtx.Origin = callerAddress
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))
	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000

	vmInstance := vm.NewEVMWithNFT(vmCtx, executor.accountdb, executor.accountdb)
	caller := vm.AccountRef(vmCtx.Origin)

	code, done := executor.generateCode(header)
	if done {
		return
	}

	codeBytes := common.FromHex(code)
	_, _, _, err := vmInstance.Call(caller, common.EconomyContract, codeBytes, vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		rewardLog.Errorf("Genesis cross contract create error: %s", err.Error())
	}
}

func (executor *VMExecutor) generateCode(header *types.BlockHeader) (string, bool) {
	proposals, validators := service.MinerManagerImpl.GetAllMinerIdAndAccount(header.Height, executor.accountdb)

	//"0x7822b9ac"+出块人奖励地址+padding+common.GenerateCallDataUint((4+len(proposes))*32)
	//		+common.GenerateCallDataUint(len(proposes))+所有的提案组成员地址+common.GenerateCallDataUint(len(验证组成员))+验证组成员地址
	code := "0x7822b9ac" + common.GenerateCallDataAddress(proposals[common.ToHex(header.Castor)]) + padding + common.GenerateCallDataUint(uint64(4+len(proposals))*32)
	code += common.GenerateCallDataUint(uint64(len(proposals)))
	for _, addr := range proposals {
		code += common.GenerateCallDataAddress(addr)
	}

	// get validator group
	groupId := header.GroupId
	var group *types.Group
	if executor.situation != "fork" {
		group = groupChainImpl.GetGroupById(groupId)
	} else {
		group = SyncProcessor.GetGroupById(groupId)
	}
	if group == nil {
		rewardLog.Errorf("fail to get group. id: %v", groupId)
		return "", true
	}

	code += common.GenerateCallDataUint(uint64(len(group.Members)))
	for _, member := range group.Members {
		code += common.GenerateCallDataAddress(validators[common.ToHex(member)])
	}

	return code, false
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	if nil == amount || 0 == amount.Sign() {
		return
	}

	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
