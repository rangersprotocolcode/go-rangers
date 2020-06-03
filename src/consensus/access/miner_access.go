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
)

var minerPoolReaderInstance *MinerPoolReader

type MinerPoolReader struct {
	minerManager *core.MinerManager
}

func NewMinerPoolReader(mp *core.MinerManager) *MinerPoolReader {
	if logger == nil {
		logger = log.GetLoggerByIndex(log.AccessLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	}
	if minerPoolReaderInstance == nil {
		minerPoolReaderInstance = &MinerPoolReader{
			minerManager: mp,
		}
	}
	return minerPoolReaderInstance
}

func (reader *MinerPoolReader) GetPubkey(id groupsig.ID) ([]byte, error) {
	return reader.minerManager.GetPubkey(id.Serialize())
}

func (reader *MinerPoolReader) GetProposeMiner(id groupsig.ID, hash common.Hash) *model.MinerInfo {
	minerManager := reader.minerManager
	if minerManager == nil {
		return nil
	}

	accountDB, _ := service.AccountDBManagerInstance.GetAccountDBByHash(hash)
	miner := minerManager.GetMinerById(id.Serialize(), common.MinerTypeProposer, accountDB)
	if miner == nil {
		//access.blog.log("getMinerById error id %v", id.ShortS())
		return nil
	}

	return reader.convert2MinerDO(miner)
}

func (reader *MinerPoolReader) GetCandidateMiners(h uint64) []model.MinerInfo {
	miners := reader.getAllMiner(common.MinerTypeValidator, h)
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

func (reader *MinerPoolReader) GetTotalStake(h uint64) uint64 {
	return reader.minerManager.GetProposerTotalStake(h)
}

func (reader *MinerPoolReader) getAllMiner(minerType byte, height uint64) []*model.MinerInfo {
	iter := reader.minerManager.MinerIterator(minerType, height)
	mds := make([]*model.MinerInfo, 0)
	for iter.Next() {
		if curr, err := iter.Current(); err != nil {
			continue
			logger.Errorf("minerManager iterator error %v", err)
		} else {
			md := reader.convert2MinerDO(curr)
			mds = append(mds, md)
		}
	}
	return mds
}

//convert2MinerDO
func (reader *MinerPoolReader) convert2MinerDO(miner *types.Miner) *model.MinerInfo {
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
