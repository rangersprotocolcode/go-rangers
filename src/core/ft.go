package core

import (
	"x/src/common"
	"x/src/middleware/types"
	"math/big"
	"encoding/json"
	"x/src/storage/account"
	"sync"
	"fmt"
	"strings"
	"x/src/utility"
)

type FTManager struct {
	lock sync.RWMutex
}

var FTManagerInstance *FTManager

func initFTManager() {
	FTManagerInstance = &FTManager{}
	FTManagerInstance.lock = sync.RWMutex{}
}

// 查询
func (self *FTManager) GetFTSet(id string, accountDB *account.AccountDB) *types.FTSet {
	self.lock.RLock()
	defer self.lock.RUnlock()

	valueByte := accountDB.GetData(common.FTSetAddress, []byte(id))
	if nil == valueByte || 0 == len(valueByte) {
		return nil
	}

	var ftSet types.FTSet
	err := json.Unmarshal(valueByte, &ftSet)
	if err != nil {
		logger.Error("fail to get ftSet: %s, %s", id, err.Error())
		return nil
	}

	return &ftSet
}

//
// ID     string // 代币ID，在发行时由layer2生成。生成规则时appId-symbol。例如0x12ef3-NOX。特别的，对于公链币，layer2会自动发行，例如official-ETH
// Name   string // 代币名字，例如以太坊
// Symbol string // 代币代号，例如ETH
// AppId  string // 发行方
// TotalSupply int64 // 发行总数， -1表示无限量（对于公链币，也是如此）
// Remain      int64 // 还剩下多少，-1表示无限（对于公链币，也是如此）
// Type        byte  // 类型，0代表公链币，1代表游戏发行的FT
func (self *FTManager) PublishFTSet(name, symbol, appId, total, owner, createTime string, kind byte, accountDB *account.AccountDB) (string, bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

	// checkId
	if 0 == len(appId) || 0 == len(symbol) || strings.Contains(appId, "-") || strings.Contains(symbol, "-") || appId == "official" {
		return "appId or symbol wrong", false
	}

	// 生成id
	id := self.genID(appId, symbol)

	// 检查id是否已存在
	if self.contains(id, accountDB) {
		return id, false
	}

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

	self.updateFTSet(id, ftSet, accountDB)
	return id, true
}

// 扣
func (self *FTManager) SubFTSet(owner, ftId string, amount *big.Int, accountDB *account.AccountDB) bool {
	self.lock.Lock()
	defer self.lock.Unlock()

	var ftSet *types.FTSet
	valueByte := accountDB.GetData(common.FTSetAddress, []byte(ftId))
	if nil == valueByte || 0 == len(valueByte) {
		return false
	}

	var ftSetData types.FTSet
	err := json.Unmarshal(valueByte, &ftSetData)
	if err != nil {
		logger.Error("fail to get ftSet: %s, %s", ftId, err.Error())
		return false
	}

	ftSet = &ftSetData
	if ftSet.Owner != owner {
		return false
	}

	total := amount
	if ftSet.TotalSupply != nil {
		total = new(big.Int).Add(ftSet.TotalSupply, amount)
	}

	if ftSet.MaxSupply.Sign() != 0 && total.Cmp(ftSet.MaxSupply) > 0 {
		return false
	}

	ftSet.TotalSupply = total
	self.updateFTSet(ftId, ftSet, accountDB)
	return true
}

func (self *FTManager) TransferFT(source string, ftId string, target string, supply string, accountDB *account.AccountDB) (string, *big.Int, bool) {
	if 0 == len(ftId) || 0 == len(supply) {
		return "", nil, true
	}
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
		return "Wrong Params", false
	}

	balance := self.convert(supply)
	if !self.SubFTSet(owner, ftId, balance, accountDB) {
		txLogger.Tracef("Mint ft not enough FT!ftId %s,target:%s,supply:%s", ftId, target, supply)
		return "Not Enough FT", false
	}

	targetAddress := common.HexToAddress(target)
	if accountDB.AddFT(targetAddress, ftId, balance) {
		return "MintFT successful", true
	} else {
		txLogger.Tracef("Mint ft overflow!ftId %s,target:%s,supply:%s", ftId, target, supply)
		return "Overflow", false
	}

}

func (self *FTManager) genID(appId, symbol string) string {
	return fmt.Sprintf("%s-%s", appId, symbol)
}

func (self *FTManager) updateFTSet(id string, ftSet *types.FTSet, accountDB *account.AccountDB) {
	data, _ := json.Marshal(ftSet)
	accountDB.SetData(common.FTSetAddress, []byte(id), data)
}

func (self *FTManager) contains(id string, accountDB *account.AccountDB) bool {

	valueByte := accountDB.GetData(common.FTSetAddress, []byte(id))
	if nil == valueByte || 0 == len(valueByte) {
		return false
	}

	return true
}

func (self *FTManager) convert(value string) *big.Int {
	supply, err := utility.StrToBigInt(value)
	if err != nil {
		return nil
	}

	return supply
}
