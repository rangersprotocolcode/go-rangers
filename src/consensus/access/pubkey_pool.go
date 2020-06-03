package access

import (
	"x/src/consensus/groupsig"
	"x/src/middleware/log"
)

var logger log.Logger

var pkPool pubkeyPool

// pubkeyPool is the cache stores public keys of miners which is used for accelerated calculation
type pubkeyPool struct {
	minerPoolReader *MinerPoolReader
}

func InitPubkeyPool(minerPoolReader *MinerPoolReader) {
	pkPool = pubkeyPool{
		minerPoolReader: minerPoolReader,
	}
}

// GetMinerPK returns pubic key of the given id
// It firstly retrieves from the cache, if missed, it gets from the chain and updates the cache.
func GetMinerPubKey(id groupsig.ID) *groupsig.Pubkey {
	if !ready() {
		return nil
	}

	value, err := pkPool.minerPoolReader.GetPubkey(id)
	if err == nil {
		pk := groupsig.ByteToPublicKey(value)
		return &pk
	}
	return nil
}

func ready() bool {
	return pkPool.minerPoolReader != nil
}
