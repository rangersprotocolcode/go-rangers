package account

import (
	"math/big"
	"x/src/middleware/types"
)

func (self *accountObject) checkAndCreate(gameId string) {
	if self.empty() {
		self.touch()
	}

	if self.data.GameData.GetNFTMaps(gameId) == nil {
		self.data.GameData.SetNFT(gameId, &types.NFTMap{})
		self.db.transitions = append(self.db.transitions, createGameDataChange{
			account: &self.address,
			gameId:  gameId,
		})
	}
}

func (self *accountObject) getFT(ftList []*types.FT, name string) *types.FT {
	for _, ft := range ftList {
		if ft.ID == name {
			return ft
		}
	}

	return nil
}

func (c *accountObject) AddFT(amount *big.Int, name string) {
	if amount.Sign() == 0 {
		if c.empty() {
			c.touch()
		}

		return
	}
	raw := c.getFT(c.data.Ft, name)
	if nil != raw {
		c.SetFT(new(big.Int).Set(amount), name)
	} else {
		c.SetFT(new(big.Int).Add(raw.Balance, amount), name)
	}

}

func (c *accountObject) SubFT(amount *big.Int, name string) bool {
	if amount.Sign() == 0 {
		return true
	}

	raw := c.getFT(c.data.Ft, name)
	// 余额不足就滚粗
	if nil == raw || raw.Balance.Cmp(amount) == -1 {
		return false
	}

	c.SetFT(new(big.Int).Sub(raw.Balance, amount), name)
	return true
}

func (self *accountObject) SetFT(amount *big.Int, name string) {
	raw := self.getFT(self.data.Ft, name)
	change := tuntunFTChange{
		account: &self.address,
		name:    name,
	}
	if raw != nil {
		change.prev = new(big.Int).Set(raw.Balance)
	} else {
		change.prev = nil
	}

	self.db.transitions = append(self.db.transitions, change)
	self.setFT(amount, name)
}

func (self *accountObject) setFT(amount *big.Int, name string) {
	if nil == amount || amount.Sign() == 0 {
		index := -1
		for i, ft := range self.data.Ft {
			if ft.ID == name {
				index = i
				break
			}
		}
		if -1 != index {
			self.data.Ft = append(self.data.Ft[:index], self.data.Ft[index+1:]...)
		}
	} else {
		ftObject := self.getFT(self.data.Ft, name)
		if ftObject == nil {
			ftObject = &types.FT{}
			ftObject.ID = name

			self.data.Ft = append(self.data.Ft, ftObject)
		}
		ftObject.Balance = new(big.Int).Set(amount)
	}

	self.callback()
}

func (self *accountObject) SetNFTByGameId(gameId string, name string, value string) {
	self.checkAndCreate(gameId)
	data := self.data.GameData.GetNFTMaps(gameId)
	change := tuntunNFTChange{
		account: &self.address,
		gameId:  gameId,
		name:    name,
	}
	if data != nil && nil != data.GetNFT(name) && 0 != len(data.GetNFT(name).GetData(gameId)) {
		change.prev = data.GetNFT(name).GetData(gameId)
	}
	self.db.transitions = append(self.db.transitions, change)
	self.setNFTByGameId(gameId, name, value)
}

func (self *accountObject) setNFTByGameId(gameId string, name string, value string) {
	data := self.data.GameData.GetNFTMaps(gameId)
	if nil == data {
		return
	}

	if 0 == len(value) {
		data.Delete(name)
	} else {
		nftData := data.GetNFT(name)
		if nil != nftData {
			nftData.SetData(value,gameId)
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
