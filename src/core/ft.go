package core

import (
	"x/src/common"
	"x/src/middleware/types"
	"strconv"
	"math/big"
	"encoding/json"
	"x/src/storage/account"
	"sync"
	"fmt"
	"strings"
)

func TransferFT(source string, ftId string, target string, supply string, accountDB *account.AccountDB) (string, bool) {
	if 0 == len(ftId) || 0 == len(supply) {
		return "", true
	}
	ftInfo := strings.Split(ftId, "-")
	if 2 != len(ftInfo) {
		return fmt.Sprintf("invalid ftId: %s", ftId), false
	}

	// 发行方
	if source == ftInfo[0] {
		return transferFTFromPublisher(source, ftInfo[1], target, supply, accountDB)
	}

	balance := FTManagerInstance.convert(supply)
	if !accountDB.SubFT(common.HexToAddress(source), ftId, balance) {
		return fmt.Sprintf("not enough ft. ftId: %s, supply: %s", ftId, supply), false
	}

	if accountDB.AddFT(common.HexToAddress(target), ftId, balance) {
		return "success", true
	} else {
		return "overflow", false
	}

}

// 发行方转币给玩家
func transferFTFromPublisher(appId string, symbol string, target string, supply string, accountDB *account.AccountDB) (string, bool) {
	if 0 == len(appId) || 0 == len(target) || 0 == len(symbol) || 0 == len(supply) {
		return "wrong params", false
	}

	balance := FTManagerInstance.convert(supply)
	if !FTManagerInstance.SubFTSet(appId, symbol, balance, accountDB) {
		return "not enough FT", false
	}

	targetAddress := common.HexToAddress(target)
	if accountDB.AddFT(targetAddress, FTManagerInstance.genID(appId, symbol), balance){
		return "TransferFT successful", true
	}else {
		return "overflow", false
	}


}

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
func (self *FTManager) PublishFTSet(name, symbol, appId, total string, kind byte, accountDB *account.AccountDB) (string, bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

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
		TotalSupply: self.convert(total),
		Remain:      self.convert(total),
		Type:        kind,
	}

	self.updateFTSet(id, ftSet, accountDB)
	return id, true
}

// 扣
func (self *FTManager) SubFTSet(appId, symbol string, amount *big.Int, accountDB *account.AccountDB) bool {
	self.lock.Lock()
	defer self.lock.Unlock()

	id := self.genID(appId, symbol)
	var ftSet *types.FTSet

	valueByte := accountDB.GetData(common.FTSetAddress, []byte(id))
	if nil == valueByte || 0 == len(valueByte) {
		return false
	}

	var ftSetData types.FTSet
	err := json.Unmarshal(valueByte, &ftSetData)
	if err != nil {
		logger.Error("fail to get ftSet: %s, %s", id, err.Error())
		return false
	}

	ftSet = &ftSetData

	if ftSet.Remain.Cmp(amount) == -1 {
		return false
	}

	ftSet.Remain = ftSet.Remain.Sub(ftSet.Remain, amount)
	self.updateFTSet(id, ftSet, accountDB)
	return true
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
	supply, _ := strconv.ParseFloat(value, 64)
	return big.NewInt(int64(supply * 1000000000))
}
