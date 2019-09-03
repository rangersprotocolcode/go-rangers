package account

import (
	"x/src/common"
	"math/big"
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
	tuntunAddNFTChange struct {
		account *common.Address
		gameId  string
		id      string
		setId   string
	}
	createGameDataChange struct {
		account *common.Address
		gameId  string
	}
)

func (ch tuntunFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setFT(ch.prev, ch.name)
}
func (ch tuntunNFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setNFTByGameId(ch.gameId, ch.setId, ch.id, ch.prev)
}
func (ch createGameDataChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).data.GameData.Delete(ch.gameId)
}
func (ch tuntunAddNFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).data.GameData.GetNFTMaps(ch.gameId).Delete(ch.setId, ch.id)
}
