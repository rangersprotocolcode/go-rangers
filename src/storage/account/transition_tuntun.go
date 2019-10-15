package account

import (
	"x/src/common"
	"math/big"
	"x/src/middleware/types"
)

type (
	tuntunFTChange struct {
		account *common.Address
		prev    *big.Int
		name    string
	}
	tuntunNFTChange struct {
		account *common.Address
		prev    string
		gameId  string
		setId   string
		id      string
	}
	tuntunNFTApproveChange struct {
		account *common.Address
		appId   string
		setId   string
		id      string
		prev    string
	}
	tuntunNFTStatusChange struct {
		account *common.Address
		prev    byte
		appId   string
		setId   string
		id      string
	}
	tuntunAddNFTChange struct {
		account *common.Address
		gameId  string
		id      string
		setId   string
	}
	tuntunRemoveNFTChange struct {
		account *common.Address
		gameId  string
		nft     *types.NFT
	}
	createGameDataChange struct {
		account *common.Address
		gameId  string
	}
)

func (ch tuntunFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setFT(ch.prev, ch.name)
}
func (ch tuntunNFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setNFTByGameId(ch.gameId, ch.setId, ch.id, ch.prev)
}
func (ch createGameDataChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).data.GameData.Delete(ch.gameId)
}

func (ch tuntunAddNFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).data.GameData.GetNFTMaps(ch.gameId).Delete(ch.setId, ch.id)
}

func (ch tuntunRemoveNFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).data.GameData.GetNFTMaps(ch.gameId).SetNFT(ch.nft)
}

func (ch tuntunNFTApproveChange) undo(s *AccountDB) {
	object := s.getAccountObject(*ch.account, false)
	nft := object.getNFT(ch.appId, ch.setId, ch.id)
	nft.Renter = ch.prev
}

func (ch tuntunNFTStatusChange) undo(s *AccountDB) {
	object := s.getAccountObject(*ch.account, false)
	nft := object.getNFT(ch.appId, ch.setId, ch.id)
	nft.Status = ch.prev
}
