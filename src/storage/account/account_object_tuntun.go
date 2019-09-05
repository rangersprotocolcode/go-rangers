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
		self.data.GameData.SetNFTMaps(gameId, &types.NFTMap{})
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
	if nil == raw {
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

// 新增一个nft实例
func (self *accountObject) AddNFTByGameId(gameId string, nft *types.NFT) bool {
	if nil == nft {
		return false
	}
	self.checkAndCreate(gameId)
	nftMaps := self.data.GameData.GetNFTMaps(gameId)
	if nftMaps.GetNFT(nft.SetID, nft.ID) != nil {
		return false
	}

	change := tuntunAddNFTChange{
		account: &self.address,
		gameId:  gameId,
		id:      nft.ID,
		setId:   nft.SetID,
	}
	self.db.transitions = append(self.db.transitions, change)
	return nftMaps.SetNFT(nft)
}

func (self *accountObject) ApproveNFT(gameId, setId, id, renter string) bool {
	data := self.data.GameData.GetNFTMaps(gameId)
	if nil == data {
		return false
	}
	nft := data.GetNFT(setId, id)
	if nft == nil {
		return false
	}

	change := tuntunNFTApproveChange{
		nft:  nft,
		prev: nft.Renter,
	}
	self.db.transitions = append(self.db.transitions, change)
	nft.Renter = renter
	return true
}

// 更新nft属性值
func (self *accountObject) SetNFTValueByGameId(gameId, setId, id, value string) bool {
	data := self.data.GameData.GetNFTMaps(gameId)
	if nil == data {
		return false
	}

	self.checkAndCreate(gameId)

	change := tuntunNFTChange{
		account: &self.address,
		gameId:  gameId,
		setId:   setId,
		id:      id,
	}
	nft := data.GetNFT(setId, id)
	if nil != nft {
		nftValue := nft.GetData(gameId)
		if 0 != len(nftValue) {
			change.prev = nftValue
		}
	}
	self.db.transitions = append(self.db.transitions, change)

	return self.setNFTByGameId(gameId, setId, id, value)
}

func (self *accountObject) setNFTByGameId(gameId, setId, id, value string) bool {
	data := self.data.GameData.GetNFTMaps(gameId)
	if nil == data {
		return false
	}

	if 0 == len(value) {
		data.Delete(setId, id)
	} else {
		nftData := data.GetNFT(setId, id)
		if nil != nftData {
			nftData.SetData(value, gameId)
		}

	}

	self.callback()
	return true
}

func (self *accountObject) callback() {
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}
