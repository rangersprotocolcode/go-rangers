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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"math/big"
	"strings"
	"sync"
)

type FTManager struct {
	lock sync.RWMutex
}

var FTManagerInstance *FTManager

// 地址相关常量
var (
	name        = utility.StrToBytes("n")
	symbol      = utility.StrToBytes("s")
	appId       = utility.StrToBytes("a")
	owner       = utility.StrToBytes("o")
	createTime  = utility.StrToBytes("c")
	maxSupply   = utility.StrToBytes("m")
	totalSupply = utility.StrToBytes("t")
	kind        = utility.StrToBytes("k")
)

func initFTManager() {
	FTManagerInstance = &FTManager{}
	FTManagerInstance.lock = sync.RWMutex{}
}

// 查询
func (self *FTManager) GetFTSet(id string, accountDB *account.AccountDB) *types.FTSet {
	self.lock.RLock()
	defer self.lock.RUnlock()

	ftAddress := common.GenerateFTSetAddress(id)
	if !accountDB.Exist(ftAddress) {
		return nil
	}

	ftSet := types.FTSet{ID: id,
		Name:        utility.BytesToStr(accountDB.GetData(ftAddress, name)),
		Symbol:      utility.BytesToStr(accountDB.GetData(ftAddress, symbol)),
		AppId:       utility.BytesToStr(accountDB.GetData(ftAddress, appId)),
		Owner:       utility.BytesToStr(accountDB.GetData(ftAddress, owner)),
		CreateTime:  utility.BytesToStr(accountDB.GetData(ftAddress, createTime)),
		MaxSupply:   new(big.Int).SetBytes(accountDB.GetData(ftAddress, maxSupply)),
		TotalSupply: new(big.Int).SetBytes(accountDB.GetData(ftAddress, totalSupply)),
	}

	k := accountDB.GetData(ftAddress, kind)
	if nil != k && 1 == len(k) {
		ftSet.Type = k[0]
	}

	return &ftSet
}

func (self *FTManager) GenerateFTSet(name, symbol, appId, total, owner, createTime string, kind byte) *types.FTSet {
	// 生成id
	id := fmt.Sprintf("%s-%s", appId, symbol)

	// 生成ftSet
	ftSet := &types.FTSet{
		ID:          id,
		Name:        name,
		Symbol:      symbol,
		AppId:       appId,
		MaxSupply:   self.convert(total),
		TotalSupply: big.NewInt(0),
		Type:        kind,
		Owner:       owner,
		CreateTime:  createTime,
	}

	return ftSet
}

//
// ID     string // 代币ID，在发行时由layer2生成。生成规则时appId-symbol。例如0x12ef3-NOX。特别的，对于公链币，layer2会自动发行，例如official-ETH
// Name   string // 代币名字，例如以太坊
// Symbol string // 代币代号，例如ETH
// AppId  string // 发行方
// TotalSupply int64 // 发行总数， -1表示无限量（对于公链币，也是如此）
// Remain      int64 // 还剩下多少，-1表示无限（对于公链币，也是如此）
// Type        byte  // 类型，0代表公链币，1代表游戏发行的FT
func (self *FTManager) PublishFTSet(ftSet *types.FTSet, accountDB *account.AccountDB) (string, bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if nil == ftSet {
		return "", false
	}

	// checkId
	if 0 == len(ftSet.AppId) || 0 == len(ftSet.Symbol) || strings.Contains(ftSet.AppId, "-") || strings.Contains(ftSet.Symbol, "-") || ftSet.AppId == common.Official {
		return "appId or symbol wrong", false
	}

	// 检查id是否已存在
	ftAddress := common.GenerateFTSetAddress(ftSet.ID)
	if !accountDB.Exist(ftAddress) {
		return ftSet.ID, false
	}

	accountDB.SetData(ftAddress, name, utility.StrToBytes(ftSet.Name))
	accountDB.SetData(ftAddress, symbol, utility.StrToBytes(ftSet.Symbol))
	accountDB.SetData(ftAddress, appId, utility.StrToBytes(ftSet.AppId))
	accountDB.SetData(ftAddress, owner, utility.StrToBytes(ftSet.Owner))
	accountDB.SetData(ftAddress, createTime, utility.StrToBytes(ftSet.CreateTime))
	accountDB.SetData(ftAddress, kind, []byte{ftSet.Type})
	accountDB.SetData(ftAddress, totalSupply, []byte{})
	accountDB.SetNonce(ftAddress, 1)

	if nil == ftSet.MaxSupply {
		accountDB.SetData(ftAddress, maxSupply, []byte{})
	} else {
		accountDB.SetData(ftAddress, maxSupply, ftSet.MaxSupply.Bytes())
	}

	return ftSet.ID, true
}

// 扣
func (self *FTManager) SubFTSet(triedOwner, ftId string, amount *big.Int, accountDB *account.AccountDB) bool {
	self.lock.Lock()
	defer self.lock.Unlock()

	// check ftId
	ftAddress := common.GenerateFTSetAddress(ftId)
	if !accountDB.Exist(ftAddress) {
		return false
	}

	// check owner
	ftSetOwner := utility.BytesToStr(accountDB.GetData(ftAddress, owner))
	if 0 != strings.Compare(ftSetOwner, triedOwner) {
		return false
	}

	// change total
	existedTotalSupply := new(big.Int).SetBytes(accountDB.GetData(ftAddress, totalSupply))
	total := new(big.Int).Add(existedTotalSupply, amount)

	// check maxSupply
	existedMaxSupply := new(big.Int).SetBytes(accountDB.GetData(ftAddress, maxSupply))
	if existedMaxSupply.Sign() != 0 && total.Cmp(existedMaxSupply) > 0 {
		return false
	}

	//update totalSupply
	accountDB.SetData(ftAddress, totalSupply, total.Bytes())
	return true
}

func (self *FTManager) TransferBNT(source, bntId, target, supply string, accountDB *account.AccountDB) (string, *big.Int, bool) {
	if 0 == len(bntId) || 0 == len(supply) {
		return "", nil, true
	}

	balance := self.convert(supply)
	left, ok := accountDB.SubBNT(common.HexToAddress(source), bntId, balance)
	if !ok {
		return fmt.Sprintf("not enough bnt. ftId: %s, supply: %s", bntId, supply), nil, false
	}

	if accountDB.AddBNT(common.HexToAddress(target), bntId, balance) {
		return "success", left, true
	} else {
		return "overflow", nil, false
	}
}

func (self *FTManager) TransferFT(source, ftId, target, supply string, accountDB *account.AccountDB) (string, *big.Int, bool) {
	if 0 == len(ftId) || 0 == len(supply) {
		return "", nil, true
	}
	// 检查ftId的格式 xxx-xxxx
	ftInfo := strings.Split(ftId, "-")
	if 2 != len(ftInfo) {
		return fmt.Sprintf("invalid ftId: %s", ftId), nil, false
	}

	balance := self.convert(supply)
	left, ok := accountDB.SubFT(common.HexToAddress(source), ftId, balance)
	if !ok {
		return fmt.Sprintf("not enough ft. ftId: %s, supply: %s", ftId, supply), nil, false
	}

	if accountDB.AddFT(common.HexToAddress(target), ftId, balance) {
		return "success", left, true
	} else {
		return "overflow", nil, false
	}

}

// 发行方转币给玩家
func (self *FTManager) MintFT(owner, ftId, target, supply string, accountDB *account.AccountDB) (string, bool) {
	txLogger.Tracef("MintFT ftId %s,target:%s,supply:%s", ftId, target, supply)
	if 0 == len(target) || 0 == len(ftId) || 0 == len(supply) {
		logger.Debugf("wrong params")
		return "wrong params", false
	}

	balance := self.convert(supply)
	if !self.SubFTSet(owner, ftId, balance, accountDB) {
		txLogger.Tracef("Mint ft not enough FT!ftId %s,target:%s,supply:%s", ftId, target, supply)
		return "not enough FT", false
	}

	targetAddress := common.HexToAddress(target)
	if accountDB.AddFT(targetAddress, ftId, balance) {
		return fmt.Sprintf("mintFT successful. ftId: %s, supply: %s, target: %s", ftId, supply, target), true
	} else {
		txLogger.Tracef("Mint ft overflow!ftId %s,target:%s,supply:%s", ftId, target, supply)
		return fmt.Sprintf("mint ft overflow! ftId: %s, target: %s, supply: %s", ftId, target, supply), false
	}

}

func (self *FTManager) convert(value string) *big.Int {
	supply, err := utility.StrToBigInt(value)
	if err != nil {
		return nil
	}

	return supply
}
