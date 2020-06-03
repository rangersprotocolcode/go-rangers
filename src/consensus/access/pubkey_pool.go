package access

import (
	"github.com/hashicorp/golang-lru"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/core"
	"x/src/middleware/log"
)

var logger log.Logger

var pkPool pubkeyPool

// pubkeyPool is the cache stores public keys of miners which is used for accelerated calculation
type pubkeyPool struct {
	pkCache         *lru.Cache
	minerPoolReader *MinerPoolReader
}

func InitPubkeyPool(minerPoolReader *MinerPoolReader) {
	pkPool = pubkeyPool{
		pkCache:         common.CreateLRUCache(100),
		minerPoolReader: minerPoolReader,
	}
}

// GetMinerPK returns pubic key of the given id
// It firstly retrieves from the cache, if missed, it gets from the chain and updates the cache.
func GetMinerPubKey(id groupsig.ID) *groupsig.Pubkey {
	if !ready() {
		return nil
	}

	if v, ok := pkPool.pkCache.Get(id.GetHexString()); ok {
		return v.(*groupsig.Pubkey)
	}
	miner := pkPool.minerPoolReader.GetLightMiner(id)
	if miner == nil {
		miner = pkPool.minerPoolReader.GetProposeMiner(id, core.EmptyHash)
	}
	if miner != nil {
		pkPool.pkCache.Add(id.GetHexString(), &miner.PubKey)
		return &miner.PubKey
	}
	return nil
}

func ready() bool {
	return pkPool.minerPoolReader != nil
}
