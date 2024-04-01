package logical

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"time"
)

func (r *round1) Start() *Error {
	r.started = true
	r.processed = make(map[string]byte)

	return nil
}
func (r *round1) Update(msg model.ConsensusMessage) *Error {
	r.processed[msg.GetMessageID()] = 1

	ccm, _ := msg.(*model.ConsensusCastMessage)
	r.logger.Debugf("update message, %v", ccm)

	bh := ccm.BH

	// check qn
	totalQN := r.blockchain.TotalQN()
	if totalQN > bh.TotalQN {
		return NewError(fmt.Errorf("qn error, height: %d, preHash: %s, signed: %d, current: %d", bh.Height, bh.PreHash.String(), totalQN, bh.TotalQN), "ccm", r.RoundNumber(), "", nil)
	}

	// check pre
	preBH := r.getBlockHeaderByHash(bh.PreHash)
	if nil == preBH {
		return NewError(fmt.Errorf("no such prehash, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	// get castor publicKey
	castor := groupsig.DeserializeID(bh.Castor)
	castorDO := r.minerReader.GetProposeMiner(castor, preBH.StateTree)
	if castorDO == nil {
		return NewError(fmt.Errorf("castor error, height: %d, preHash: %s, castor: %s", bh.Height, bh.PreHash.String(), castor.GetHexString()), "ccm", r.RoundNumber(), "", nil)
	}
	pk := castorDO.PubKey
	if !pk.IsValid() {
		return NewError(fmt.Errorf("castorPK error, height: %d, preHash: %s, castor: %s", bh.Height, bh.PreHash.String(), castor.GetHexString()), "ccm", r.RoundNumber(), "", nil)
	}

	// check message sign
	si := ccm.SignInfo
	if msg.GenHash() != si.GetDataHash() {
		return NewError(fmt.Errorf("msg error, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}
	if !si.VerifySign(pk) {
		return NewError(fmt.Errorf("sign check error, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	// check castor
	if !castorDO.CanCastAt(bh.Height) {
		return NewError(fmt.Errorf("miner can't cast at height, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}
	totalStake := r.minerReader.GetTotalStake(preBH.Height, preBH.StateTree)
	if ok2, err := verifyBlockVRF(&bh, preBH, castorDO, totalStake); !ok2 {
		r.logger.Errorf("fail to verifyVRF. err: %s", err)
		return NewError(fmt.Errorf("vrf verify block fail, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	// check group
	groupId := groupsig.DeserializeID(bh.GroupId)
	if !r.belongGroups.BelongGroup(groupId) {
		return NewError(fmt.Errorf("not in group: %s, height: %d, preHash: %s", groupId.GetHexString(), bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}
	hash := CalcRandomHash(preBH, bh.CurTime)
	selectGroup, err := r.globalGroups.SelectVerifyGroupFromCache(hash, bh.Height)
	if err != nil {
		return NewError(fmt.Errorf("cannot get group fromcache: %s, height: %d, preHash: %s", groupId.GetHexString(), bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}
	if !selectGroup.IsEqual(groupId) {
		selectGroup, err = r.globalGroups.SelectVerifyGroupFromChain(hash, bh.Height)
		if err != nil {
			return NewError(fmt.Errorf("cannot get group fromchain: %s, height: %d, preHash: %s", groupId.GetHexString(), bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
		}
		if !selectGroup.IsEqual(groupId) {
			return NewError(fmt.Errorf("select group error: %s, height: %d, preHash: %s", groupId.GetHexString(), bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
		}
	}

	// check block time
	timeNow := utility.GetTime()
	deviationTime := bh.CurTime.Add(time.Second * -1)
	if !bh.CurTime.After(preBH.CurTime) || !timeNow.After(deviationTime) {
		return NewError(fmt.Errorf("time error, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	lostTxs, ccr := core.GetBlockChain().VerifyBlock(&bh)
	if -1 == ccr {
		return NewError(fmt.Errorf("blockheader error, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	if 0 == len(lostTxs) {
		//normalPieceVerify
		r.normalPieceVerify(bh, *preBH, groupId)

		r.changedId <- bh.Hash.String()
		r.canProcessed = true
	} else {
		//m := notify.TransactionGotAddSuccMessage{Transactions: txs, Peer: from}
		//notify.BUS.Publish(notify.TransactionGotAddSucc, &m)
	}
	return nil
}

func (r *round1) normalPieceVerify(bh, prevBH types.BlockHeader, gid groupsig.ID) {
	var cvm model.ConsensusVerifyMessage
	cvm.BlockHash = bh.Hash
	skey := r.getSignKey(gid)

	if signInfo, ok := model.NewSignInfo(skey, r.mi, &cvm); ok {
		cvm.SignInfo = signInfo
		r.logger.Debug("SendVerifiedCast seckey=%v, miner id=%v,data hash:%v,sig:%v", skey.GetHexString(), r.mi.GetHexString(), cvm.SignInfo.GetDataHash().String(), cvm.SignInfo.GetSignature().GetHexString())
		cvm.GenRandomSign(skey, prevBH.Random)
		r.netServer.SendVerifiedCast(&cvm, gid)
	} else {
		r.logger.Errorf("genSign fail, sk=%v %v", skey.ShortS(), r.belongGroups.BelongGroup(gid))
	}
}

// getSignKey get the signature private key of the miner in a certain group
func (r *round1) getSignKey(gid groupsig.ID) groupsig.Seckey {
	if jg := r.belongGroups.GetJoinedGroupInfo(gid); jg != nil {
		return jg.SignSecKey
	}
	return groupsig.Seckey{}
}

func (r *round1) CanAccept(msg model.ConsensusMessage) int {
	if _, ok := r.processed[msg.GetMessageID()]; ok {
		return -1
	}

	_, ok := msg.(*model.ConsensusCastMessage)
	if ok {
		return 0
	}

	_, ok = msg.(*model.ConsensusVerifyMessage)
	if ok {
		return 1
	}

	return -1
}
func (r *round1) CanProceed() bool {
	return r.canProcessed
}
func (r *round1) NextRound() Round {
	return nil
}

func (r *round1) getBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	b := r.blockchain.QueryBlockByHash(hash)
	if b != nil {
		return b.Header
	}
	return nil
}
