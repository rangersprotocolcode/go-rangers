package account

import (
	"math/big"
	"x/src/middleware/types"
)

func (self *accountObject) checkAndCreate(gameId string) {
	if self.empty() {
		self.touch()
	}

	if self.data.GameData[gameId] == nil {
		self.data.GameData[gameId] = make(map[string]*types.NFT, 0)
		self.db.transitions = append(self.db.transitions, createGameDataChange{
			account: &self.address,
			gameId:  gameId,
		})
	}
}

func (c *accountObject) AddFT(amount *big.Int, name string) {
	if amount.Sign() == 0 {
		if c.empty() {
			c.touch()
		}

		return
	}
	raw := c.data.Ft[name]
	if raw == nil {
		c.SetFT(new(big.Int).Set(amount), name)
	} else {
		c.SetFT(new(big.Int).Add(raw.Balance, amount), name)
	}

}

func (c *accountObject) SubFT(amount *big.Int, name string) bool {
	if amount.Sign() == 0 {
		return true
	}

	raw := c.data.Ft[name]
	// 余额不足就滚粗
	if nil == raw || raw.Balance.Cmp(amount) == -1 {
		return false
	}

	c.SetFT(new(big.Int).Sub(raw.Balance, amount), name)
	return true
}

func (self *accountObject) SetFT(amount *big.Int, name string) {
	raw := self.data.Ft[name]
	change := tuntunFTChange{
		account: &self.address,
		name:    name,
	}
	if nil != raw {
		change.prev = new(big.Int).Set(raw.Balance)
	} else {
		change.prev = nil
	}

	self.db.transitions = append(self.db.transitions, change)
	self.setFT(amount, name)
}

func (self *accountObject) setFT(amount *big.Int, name string) {
	if nil == amount || amount.Sign() == 0 {
		delete(self.data.Ft, name)
	} else {
		ftObject := self.data.Ft[name]
		if nil == ftObject {
			ftObject = &types.FT{}
			ftObject.ID = name

			self.data.Ft[name] = ftObject
		}
		ftObject.Balance = new(big.Int).Set(amount)
	}

	self.callback()
}

func (self *accountObject) SetNFTByGameId(gameId string, name string, value string) {
	self.checkAndCreate(gameId)
	data := self.data.GameData[gameId]
	change := tuntunNFTChange{
		account: &self.address,
		gameId:  gameId,
		name:    name,
	}
	if data != nil && nil != data[name] && 0 != len(data[name].Data[gameId]) {
		change.prev = data[name].Data[gameId]
	}
	self.db.transitions = append(self.db.transitions, change)
	self.setNFTByGameId(gameId, name, value)
}

func (self *accountObject) setNFTByGameId(gameId string, name string, value string) {
	data := self.data.GameData[gameId]
	if nil == data {
		return
	}

	if 0 == len(value) {
		delete(data, name)
	} else {
		nftData := data[name]
		if nil != nftData {
			nftData.Data[gameId] = value
		}

	}

	self.callback()
}

func (self *accountObject) callback() {
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}
