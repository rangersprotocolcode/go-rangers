package account

import (
	"x/src/common"
	"math/big"
	"x/src/middleware/types"
)

func (self *AccountDB) GetFT(addr common.Address, ftName string) *big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	if 0 == len(data.Ft) {
		return big.NewInt(0)
	}

	raw := accountObject.getFT(accountObject.data.Ft, ftName)
	if raw == nil {
		return big.NewInt(0)
	}
	return raw.Balance
}

func (self *AccountDB) GetAllFT(addr common.Address) map[string]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	if 0 == len(data.Ft) {
		return nil
	}

	result := make(map[string]*big.Int, len(data.Ft))
	for _, value := range data.Ft {
		result[value.ID] = value.Balance
	}
	return result
}

func (self *AccountDB) SetFT(addr common.Address, ftName string, balance *big.Int) {
	if nil == balance {
		return
	}
	account := self.getOrNewAccountObject(addr)
	account.SetFT(balance, ftName)
}

func (self *AccountDB) AddFT(addr common.Address, ftName string, balance *big.Int) {
	if nil == balance {
		return
	}
	account := self.getOrNewAccountObject(addr)
	account.AddFT(balance, ftName)
}

func (self *AccountDB) SubFT(addr common.Address, ftName string, balance *big.Int) bool {
	if nil == balance {
		return true
	}
	account := self.getOrNewAccountObject(addr)
	return account.SubFT(balance, ftName)

}

// 根据setId/id 查找NFT
func (self *AccountDB) GetNFTById(addr common.Address, setId, id string) *types.NFT {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data.GameData

	for _, nftMap := range data.NFTMaps {
		nft := nftMap.GetNFT(setId, id)
		if nil != nft {
			return nft
		}
	}

	return nil
}

// 在某个gameId下根据setId/id 查找NFT
func (self *AccountDB) GetNFTByGameId(addr common.Address, gameId, setId, id string) string {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	nftList := data.GameData.GetNFTMaps(gameId)
	if nil == nftList {
		return ""
	}

	nft := nftList.GetNFT(setId, id)
	if nil != nft {
		return nft.GetData(gameId)
	}
	return ""
}

func (self *AccountDB) GetAllNFTByGameId(addr common.Address, gameId string) []*types.NFT {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	nftList := data.GameData.GetNFTMaps(gameId)
	if nil == nftList {
		return nil
	}

	return nftList.GetAllNFT()
}

func (self *AccountDB) AddNFTByGameId(addr common.Address, appId string, nft *types.NFT) bool {
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.AddNFTByGameId(appId, nft)
}

func (self *AccountDB) SetNFTValueByGameId(addr common.Address, appId, setId, id, value string) bool{
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.SetNFTValueByGameId(appId, setId, id, value)
}

func (self *AccountDB) RemoveNFTByGameId(addr common.Address, gameId, setId, id string) bool{
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.SetNFTValueByGameId(gameId, setId, id, "")
}
