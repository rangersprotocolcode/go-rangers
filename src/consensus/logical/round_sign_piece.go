// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package logical

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	"fmt"
)

func (r *round2) Start() *Error {
	if r.started {
		return nil
	}

	bh := r.bh
	r.logger.Debugf("round2 start. hash: %s, height: %d", bh.Hash.String(), bh.Height)

	gid := groupsig.DeserializeID(bh.GroupId)
	group, err := r.globalGroups.GetGroupByID(gid)
	if err != nil || nil == group {
		return NewError(fmt.Errorf("cannot get group: %s, hash: %s, height: %d", gid.GetHexString(), bh.Hash.String(), bh.Height), "omv-start", r.RoundNumber(), "", nil)
	}
	threshold := model.Param.GetGroupK(group.GetMemberCount())
	r.gSignGenerator = model.NewGroupSignGenerator(threshold)
	r.rSignGenerator = model.NewGroupSignGenerator(threshold)

	if 0 == len(r.futureMessages) {
		return nil
	}

	idList := make([]string, 0)
	for id, msg := range r.futureMessages {
		idList = append(idList, id)
		if err := r.Update(msg); err != nil {
			return err
		}
	}

	for _, id := range idList {
		r.processed[id] = 1
		delete(r.futureMessages, id)
	}

	r.started = true
	return nil
}

func (r *round2) Close() {

}

func (r *round2) Update(msg model.ConsensusMessage) *Error {
	bh := r.bh

	cvm, ok := msg.(*model.ConsensusVerifyMessage)
	if !ok {
		return NewError(fmt.Errorf("cannot update for wrong msg"), "omv", r.RoundNumber(), "", nil)
	}
	r.logger.Debugf("round2 update, from: %s, hash: %s, height: %d", cvm.SignInfo.GetSignerID().GetHexString(), cvm.BlockHash.String(), bh.Height)

	if r.blockchain.HasBlockByHash(cvm.BlockHash) {
		return NewError(fmt.Errorf("already existed. hash: %s, height: %d", cvm.BlockHash.String(), bh.Height), "omv", r.RoundNumber(), "", nil)
	}

	gid := groupsig.DeserializeID(bh.GroupId)
	si := cvm.SignInfo

	// get pubKey
	pk, ok := group_create.GroupCreateProcessor.GetMemberSignPubKey(gid, si.GetSignerID())
	if !ok {
		r.logger.Errorf("GetMemberSignPubKey not ok, id: %s. hash: %s, height: %d", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), bh.Height)
		return nil
	}

	// check data
	if !si.VerifySign(pk) {
		r.logger.Errorf("fail to verify sign, id: %s. hash: %s, height: %d", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), bh.Height)
		return nil
	}

	// check signature
	sig := groupsig.DeserializeSign(cvm.RandomSign.Serialize())
	if sig == nil || sig.IsNil() {
		r.logger.Errorf("fail to deserialize bh random, id: %s. hash: %s, height: %d, random: %s", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), bh.Height, cvm.RandomSign.GetHexString())
		return nil
	}
	if !groupsig.VerifySig(pk, r.preBH.Random, *sig) {
		r.logger.Errorf("fail to verify random sign, id: %s. hash: %s, height: %d, random: %s", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), bh.Height, cvm.RandomSign.GetHexString())
		return nil
	}

	add, generate := r.gSignGenerator.AddWitnessSign(si.GetSignerID(), si.GetSignature())
	if !add {
		r.logger.Warnf("already had the piece, from: %s, hash: %s, height: %d", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), bh.Height)
		return nil
	}
	r.logger.Debugf("round2 add piece, from: %s, hash: %s, height: %d", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), bh.Height)

	radd, rgen := r.rSignGenerator.AddWitnessSign(si.GetSignerID(), *sig)
	if radd && generate && rgen {
		bh.Signature = r.gSignGenerator.GetGroupSign().Serialize()
		bh.Random = r.rSignGenerator.GetGroupSign().Serialize()
		r.canProcessed = true
		r.logger.Infof("round2 recovered group sign. hash: %s, height: %d, group sign: %s", bh.Hash.String(), bh.Height, common.ToHex(bh.Signature))
	}

	return nil
}

func (r *round2) CanAccept(msg model.ConsensusMessage) int {
	msgId := msg.GetMessageID()
	if _, ok := r.processed[msgId]; ok {
		return -1
	}
	if _, ok := r.futureMessages[msgId]; ok {
		return -1
	}

	_, ok := msg.(*model.ConsensusCastMessage)
	if ok {
		return 1
	}

	_, ok = msg.(*model.ConsensusVerifyMessage)
	if ok {
		return 0
	}

	return -1
}

func (r *round2) NextRound() Round {
	r.canProcessed = true
	r.number = 2
	return &round3{round2: r}
}
