package access

import (
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/consensus/vrf"
	"x/src/core"
	"x/src/middleware/log"
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
)

var minerPoolReaderInstance *MinerPoolReader

type MinerPoolReader struct {
	minerPool       *core.MinerManager
	totalStakeCache uint64
}

func NewMinerPoolReader(mp *core.MinerManager) *MinerPoolReader {
	if logger == nil {
		logger = log.GetLoggerByIndex(log.AccessLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	}
	if minerPoolReaderInstance == nil {
		minerPoolReaderInstance = &MinerPoolReader{
			minerPool: mp,
		}
	}
	return minerPoolReaderInstance
}

func (access *MinerPoolReader) GetLightMiner(id groupsig.ID) *model.MinerInfo {
	minerPool := access.minerPool
	if minerPool == nil {
		return nil
	}
	miner := minerPool.GetMinerById(id.Serialize(), common.MinerTypeValidator, nil)
	if miner == nil {
		//access.blog.log("getMinerById error id %v", id.ShortS())
		return nil
	}
	return access.convert2MinerDO(miner)
}

func (access *MinerPoolReader) GetProposeMiner(id groupsig.ID, hash common.Hash) *model.MinerInfo {
	minerPool := access.minerPool
	if minerPool == nil {
		return nil
	}

	var accountDB *account.AccountDB
	accountDB = nil
	if hash != core.EmptyHash {
		accountDB, _ = service.AccountDBManagerInstance.GetAccountDBByHash(hash)
	}

	miner := minerPool.GetMinerById(id.Serialize(), common.MinerTypeProposer, accountDB)
	if miner == nil {
		//access.blog.log("getMinerById error id %v", id.ShortS())
		return nil
	}
	return access.convert2MinerDO(miner)
}

func (access *MinerPoolReader) GetCandidateMiners(h uint64) []model.MinerInfo {
	miners := access.getAllMiner(common.MinerTypeValidator, h)
	rets := make([]model.MinerInfo, 0)
	logger.Debugf("all light nodes size %v", len(miners))
	for _, md := range miners {
		//access.blog.log("%v %v %v %v", md.ID.ShortS(), md.ApplyHeight, md.NType, md.CanJoinGroupAt(h))
		if md.CanJoinGroupAt(h) {
			rets = append(rets, *md)
		}
	}
	return rets
}

func (access *MinerPoolReader) GetTotalStake(h uint64, cache bool) uint64 {
	if cache && access.totalStakeCache > 0 {
		return access.totalStakeCache
	}
	st := access.minerPool.GetProposerTotalStake(h)
	access.totalStakeCache = st
	return st
}

func (access *MinerPoolReader) getAllMiner(minerType byte, height uint64) []*model.MinerInfo {
	iter := access.minerPool.MinerIterator(minerType, height)
	mds := make([]*model.MinerInfo, 0)
	for iter.Next() {
		if curr, err := iter.Current(); err != nil {
			continue
			logger.Errorf("minerManager iterator error %v", err)
		} else {
			md := access.convert2MinerDO(curr)
			mds = append(mds, md)
		}
	}
	return mds
}

//convert2MinerDO
func (access *MinerPoolReader) convert2MinerDO(miner *types.Miner) *model.MinerInfo {
	if miner == nil {
		return nil
	}
	md := &model.MinerInfo{
		ID:          groupsig.DeserializeID(miner.Id),
		PubKey:      groupsig.ByteToPublicKey(miner.PublicKey),
		VrfPK:       vrf.VRFPublicKey(miner.VrfPublicKey),
		Stake:       miner.Stake,
		MinerType:   miner.Type,
		ApplyHeight: miner.ApplyHeight,
		AbortHeight: miner.AbortHeight,
	}
	if !md.ID.IsValid() || !md.PubKey.IsValid() {
		logger.Warnf("Invalid miner! id %v, %v,miner public key:%v,%v", miner.Id, md.ID.GetHexString(), md.PubKey, md.PubKey.GetHexString())
	}
	return md
}

//func (access *MinerPoolReader) genesisMiner(miners []*types.Miner)  {
//    access.minerPool.AddGenesesMiner(miners)
//}
