// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package access

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
)

var minerPoolReaderInstance *MinerPoolReader

type MinerPoolReader struct {
	minerManager *service.MinerManager
}

func NewMinerPoolReader(mp *service.MinerManager) *MinerPoolReader {
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

	accountDB, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(hash)
	miner := minerManager.GetMinerById(id.Serialize(), common.MinerTypeProposer, accountDB)
	if miner == nil {
		//access.blog.log("getMinerById error id %v", id.ShortS())
		return nil
	}

	return reader.convert2MinerDO(miner)
}

func (reader *MinerPoolReader) GetCandidateMiners(h uint64, hash common.Hash) []model.MinerInfo {
	miners := reader.getAllMiner(common.MinerTypeValidator, hash)
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

func (reader *MinerPoolReader) GetTotalStake(h uint64, hash common.Hash) uint64 {
	return reader.minerManager.GetProposerTotalStake(h, hash)
}

func (reader *MinerPoolReader) getAllMiner(minerType byte, hash common.Hash) []*model.MinerInfo {
	iter := reader.minerManager.MinerIterator(minerType, hash)
	mds := make([]*model.MinerInfo, 0)
	for iter.Next() {
		if curr, err := iter.Current(); err != nil {
			logger.Errorf("minerManager iterator error %v", err)
			continue
		} else {
			logger.Debugf("minerManager iterator got %v", common.ToHex(curr.Id))
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
	}
	if !md.ID.IsValid() || !md.PubKey.IsValid() {
		logger.Warnf("Invalid miner! id %v, %v,miner public key:%v,%v", miner.Id, md.ID.GetHexString(), md.PubKey, md.PubKey.GetHexString())
	}
	return md
}
