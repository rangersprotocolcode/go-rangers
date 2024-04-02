package logical

import (
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	"encoding/hex"
	"fmt"
)

func (r *round2) Start() *Error {
	gid := groupsig.DeserializeID(r.ccm.BH.GroupId)
	group, err := r.globalGroups.GetGroupByID(gid)
	if err != nil || nil == group {
		return NewError(fmt.Errorf("cannot get group: %s, hash: %s, height: %d", gid.GetHexString(), r.ccm.BH.Hash.String(), r.ccm.BH.Height), "omv-start", r.RoundNumber(), "", nil)
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
	return nil
}

func (r *round2) Update(msg model.ConsensusMessage) *Error {
	cvm, ok := msg.(*model.ConsensusVerifyMessage)
	if !ok {
		return NewError(fmt.Errorf("cannot update for wrong msg"), "omv", r.RoundNumber(), "", nil)
	}

	if r.blockchain.HasBlockByHash(cvm.BlockHash) {
		return NewError(fmt.Errorf("already existed. hash: %s, height: %d", cvm.BlockHash.String(), r.ccm.BH.Height), "omv", r.RoundNumber(), "", nil)
	}

	gid := groupsig.DeserializeID(r.ccm.BH.GroupId)
	si := cvm.SignInfo

	// get pubKey
	pk, ok := group_create.GroupCreateProcessor.GetMemberSignPubKey(gid, si.GetSignerID())
	if !ok {
		r.logger.Errorf("GetMemberSignPubKey not ok, id: %s. hash: %s, height: %d", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), r.ccm.BH.Height)
		return nil
	}

	// check data
	if !si.VerifySign(pk) {
		r.logger.Errorf("fail to verify sign, id: %s. hash: %s, height: %d", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), r.ccm.BH.Height)
		return nil
	}

	// check signature
	sig := groupsig.DeserializeSign(cvm.RandomSign.Serialize())
	if sig == nil || sig.IsNil() {
		r.logger.Errorf("fail to deserialize bh random, id: %s. hash: %s, height: %d, random: %s", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), r.ccm.BH.Height, cvm.RandomSign.GetHexString())
		return nil
	}
	if !groupsig.VerifySig(pk, r.preBH.Random, *sig) {
		r.logger.Errorf("fail to verify random sign, id: %s. hash: %s, height: %d, random: %s", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), r.ccm.BH.Height, cvm.RandomSign.GetHexString())
		return nil
	}

	add, generate := r.gSignGenerator.AddWitnessSign(si.GetSignerID(), si.GetSignature())
	if !add {
		r.logger.Warnf("already had the piece, id: %s. hash: %s, height: %d", si.GetSignerID().GetHexString(), cvm.BlockHash.String(), r.ccm.BH.Height)
		return nil
	}

	radd, rgen := r.rSignGenerator.AddWitnessSign(si.GetSignerID(), *sig)
	if radd && generate && rgen {
		r.ccm.BH.Signature = r.gSignGenerator.GetGroupSign().Serialize()
		r.ccm.BH.Random = r.rSignGenerator.GetGroupSign().Serialize()
		r.logger.Debugf("Recovered group sign.Block hash:%v,group sign:%v", r.ccm.BH.Hash.String(), hex.EncodeToString(r.ccm.BH.Signature))

		r.canProcessed = true
	}

	return nil
}

func (r *round2) CanAccept(msg model.ConsensusMessage) int {
	if _, ok := r.processed[msg.GetMessageID()]; ok {
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
	return nil
}
