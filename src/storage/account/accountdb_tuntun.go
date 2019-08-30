package account

import (
	"x/src/common"
	"math/big"
	"encoding/json"
)

// 读取对应gameId下的余额
func (self *AccountDB) GetBalanceByGameId(addr common.Address, gameId string) *big.Int {
	return self.GetBalance(addr)
}

// 设置对应gameId下的余额
func (self *AccountDB) SetBalanceByGameId(addr common.Address, gameId string, balance *big.Int) {
	self.SetBalance(addr, balance)
}

// 对应gameId下的余额
func (self *AccountDB) AddBalanceByGameId(addr common.Address, gameId string, balance *big.Int) {
	self.AddBalance(addr, balance)
}

// 减少余额
func (self *AccountDB) SubBalanceByGameId(addr common.Address, gameId string, balance *big.Int) bool {
	value := self.GetBalance(addr)
	if value.Cmp(balance) == -1 {
		return false
	}

	self.SubBalance(addr, balance)
	return true
}

func (self *AccountDB) GetFTByGameId(addr common.Address, gameId string, ftName string) *big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	if 0 == len(data.Ft) {
		return big.NewInt(0)
	}

	if data.Ft[ftName] == nil {
		return big.NewInt(0)
	}
	return data.Ft[ftName].Balance
}

func (self *AccountDB) GetAllFTByGameId(addr common.Address, gameId string) map[string]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	if 0 == len(data.Ft) {
		return nil
	}

	result := make(map[string]*big.Int, len(data.Ft))
	for key, value := range data.Ft {
		result[key] = value.Balance
	}
	return result
}

func (self *AccountDB) SetFTByGameId(addr common.Address, gameId string, ftName string, balance *big.Int) {
	if nil == balance {
		return
	}
	account := self.GetOrNewAccountObject(addr)
	account.SetFTByGameId(balance, gameId, ftName)

}

func (self *AccountDB) AddFTByGameId(addr common.Address, gameId string, ftName string, balance *big.Int) {
	if nil == balance {
		return
	}
	account := self.GetOrNewAccountObject(addr)
	account.AddFTByGameId(balance, gameId, ftName)
}

func (self *AccountDB) SubFTByGameId(addr common.Address, gameId string, ftName string, balance *big.Int) bool {
	if nil == balance {
		return true
	}
	account := self.GetOrNewAccountObject(addr)
	return account.SubFTByGameId(balance, gameId, ftName)

}

func (self *AccountDB) GetNFTByGameId(addr common.Address, gameId string, name string) string {
	data := self.getGameData(addr, gameId)
	if nil == data || 0 == len(data.Nft) {
		return ""
	}

	return data.Nft[name]
}

func (self *AccountDB) GetAllNFTByGameId(addr common.Address, gameId string) map[string]string {
	data := self.getGameData(addr, gameId)
	if nil == data {
		return nil
	}
	return data.Nft
}

func (self *AccountDB) SetNFTByGameId(addr common.Address, gameId string, name string, value string) {
	stateObject := self.GetOrNewAccountObject(addr)
	stateObject.SetNFTByGameId(gameId, name, value)
}

func (self *AccountDB) RemoveNFTByGameId(addr common.Address, gameId string, name string) {
	stateObject := self.GetOrNewAccountObject(addr)
	stateObject.SetNFTByGameId(gameId, name, "")
}

// 获取游戏已经发行的ft
func (self *AccountDB) GetFtList(gameAddr common.Address) map[string]string {
	data := self.GetData(gameAddr, "ft")
	var result map[string]string

	err := json.Unmarshal(data, &result)
	if nil != err {
		//todo: log
		result = make(map[string]string)
	}

	return result
}

// 对于游戏开发者账户，有发行FT的需求
// 这里先简单处理
func (self *AccountDB) UpdateFtList(gameAddr common.Address, value map[string]string) {
	bytes, _ := json.Marshal(value)
	self.SetData(gameAddr, "ft", bytes)
}
