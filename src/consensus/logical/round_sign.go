package logical

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"sync"
	"time"
)

func (r *round1) Start() *Error {
	r.number = 0
	r.processed = make(map[string]byte)
	r.lock = sync.Mutex{}

	return nil
}

func (r *round1) Update(msg model.ConsensusMessage) *Error {
	r.processed[msg.GetMessageID()] = 1
	ccm, _ := msg.(*model.ConsensusCastMessage)
	r.logger.Debugf("update message, %v", ccm)
	r.ccm = ccm

	bh := ccm.BH

	// check qn
	totalQN := r.blockchain.TotalQN()
	if totalQN > bh.TotalQN {
		return NewError(fmt.Errorf("qn error, height: %d, preHash: %s, signed: %d, current: %d", bh.Height, bh.PreHash.String(), totalQN, bh.TotalQN), "ccm", r.RoundNumber(), "", nil)
	}

	// check pre
	preBH := r.getBlockHeaderByHash(bh.PreHash)
	if nil == preBH {
		notify.BUS.Subscribe(notify.BlockAddSucc, r.onBlockAddSuccess)
		r.logger.Warnf("no such preHash: %s, height: %d, waiting", bh.PreHash.String(), bh.Height)
		return nil
	}

	r.preBH = preBH
	return r.afterPreArrived()
}

func (r *round1) onBlockAddSuccess(message notify.Message) {
	r.lock.Lock()
	defer r.lock.Unlock()

	block := message.GetData().(types.Block)
	bh := block.Header
	if 0 != bytes.Compare(bh.Hash.Bytes(), r.ccm.BH.PreHash.Bytes()) {
		return
	}

	notify.BUS.UnSubscribe(notify.BlockAddSucc, r.onBlockAddSuccess)
	r.logger.Infof("received preBH: %s", bh.Hash.String())

	r.preBH = bh
	err := r.afterPreArrived()
	if nil != err {
		r.errChan <- err
	}
}

func (r *round1) afterPreArrived() *Error {
	bh := r.ccm.BH
	preBH := r.preBH

	// get castor publicKey
	castor := groupsig.DeserializeID(bh.Castor)
	castorDO := r.minerReader.GetProposeMiner(castor, r.preBH.StateTree)
	if castorDO == nil {
		return NewError(fmt.Errorf("castor error, height: %d, preHash: %s, castor: %s", bh.Height, bh.PreHash.String(), castor.GetHexString()), "ccm", r.RoundNumber(), "", nil)
	}
	pk := castorDO.PubKey
	if !pk.IsValid() {
		return NewError(fmt.Errorf("castorPK error, height: %d, preHash: %s, castor: %s", bh.Height, bh.PreHash.String(), castor.GetHexString()), "ccm", r.RoundNumber(), "", nil)
	}

	// check message sign
	si := r.ccm.SignInfo
	if r.ccm.GenHash() != si.GetDataHash() {
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
	if ok2, err := verifyBlockVRF(bh, preBH, castorDO, totalStake); !ok2 {
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

	return r.checkBlock()
}

func (r *round1) checkBlock() *Error {
	bh := r.ccm.BH
	preBH := r.preBH
	groupId := groupsig.DeserializeID(bh.GroupId)

	lostTxs, ccr := core.GetBlockChain().VerifyBlock(bh)
	if -1 == ccr {
		return NewError(fmt.Errorf("blockheader error, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	if r.blockchain.HasBlockByHash(bh.Hash) {
		return NewError(fmt.Errorf("blockheader already existed, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	//normalPieceVerify
	if 0 == len(lostTxs) {
		r.normalPieceVerify(*bh, *preBH, groupId)
		r.changedId <- bh.Hash.String()
		r.canProcessed = true
	} else {
		r.lostTxs = make(map[common.Hashes]byte)
		for _, hash := range lostTxs {
			r.lostTxs[hash] = 0
		}
		r.logger.Warnf("lostTxs waiting, height: %d, preHash: %s, len: %d", bh.Height, bh.PreHash.String(), len(lostTxs))
		notify.BUS.Subscribe(notify.TransactionGotAddSucc, r.onMissTxAddSucc)
	}

	return nil
}

func (r *round1) onMissTxAddSucc(message notify.Message) {
	r.lock.Lock()
	defer r.lock.Unlock()

	tgam, _ := message.(*notify.TransactionGotAddSuccMessage)

	transactions := tgam.Transactions
	for _, tx := range transactions {
		hashes := common.Hashes{}
		hashes[0] = tx.Hash
		hashes[1] = tx.SubHash

		delete(r.lostTxs, hashes)
	}

	if 0 != len(r.lostTxs) {
		return
	}

	notify.BUS.UnSubscribe(notify.TransactionGotAddSucc, r.onMissTxAddSucc)
	err := r.checkBlock()
	if nil != err {
		r.errChan <- err
	}
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

func (r *round1) NextRound() Round {
	r.canProcessed = false
	r.number = 1
	return &round2{round1: r}
}

func (r *round1) getBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	b := r.blockchain.QueryBlockByHash(hash)
	if b != nil {
		return b.Header
	}
	return nil
}
