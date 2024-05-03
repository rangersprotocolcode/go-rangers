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
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"math/big"
	"sync"
	"time"
)

func (r *round0) Start() *Error {
	if r.started {
		return nil
	}

	r.number = 0
	r.processed = make(map[string]byte)
	r.lock = sync.Mutex{}
	r.started = true

	return nil
}

func (r *round0) Close() {
	notify.BUS.UnSubscribe(notify.BlockAddSucc, r)
	notify.BUS.UnSubscribe(notify.TransactionGotAddSucc, r)
}

func (r *round0) Update(msg model.ConsensusMessage) *Error {
	r.processed[msg.GetMessageID()] = 1
	ccm, _ := msg.(*model.ConsensusCastMessage)
	r.ccm = ccm
	r.bh = &ccm.BH
	r.logger.Infof("round0 update message, height: %d, castor: %s, hash: %s, partyId: %s", r.bh.Height, common.ToHex(r.bh.Castor), r.bh.Hash.String(), r.partyId)

	// check qn
	totalQN := r.blockchain.TotalQN()
	if totalQN > r.bh.TotalQN {
		return NewError(fmt.Errorf("qn error, height: %d, preHash: %s, signedQN: %d, current: %d", r.bh.Height, r.bh.PreHash.String(), totalQN, r.bh.TotalQN), "ccm", r.RoundNumber(), "", nil)
	}

	// check pre
	preBH := r.getBlockHeaderByHash(r.bh.PreHash)
	if nil == preBH {
		notify.BUS.Subscribe(notify.BlockAddSucc, r)
		r.logger.Warnf("no such preHash: %s, height: %d, waiting", r.bh.PreHash.String(), r.bh.Height)
		return nil
	}

	r.preBH = preBH
	return r.afterPreArrived()
}

func (r *round0) afterPreArrived() *Error {
	bh := r.bh
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
		return NewError(fmt.Errorf("msg error, %s - %s, height: %d, preHash: %s", r.ccm.GenHash().String(), si.GetDataHash().String(), bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
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
	group, err := r.globalGroups.GetGroupByID(groupId)
	if nil != err {
		return NewError(fmt.Errorf("fail to get group, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}
	if !group.MemExist(r.mi) {
		return NewError(fmt.Errorf("not in group: %s, height: %d, preHash: %s", groupId.GetHexString(), bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}
	r.group = group

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

	r.logger.Debugf("round0 finish base check, height: %d, hash: %s, preHash: %s", bh.Height, bh.Hash.String(), bh.PreHash.String())
	return r.checkBlock()
}

func (r *round0) checkBlock() *Error {
	bh := r.bh
	r.logger.Debugf("round0 check block, height: %d, hash: %s, preHash: %s", bh.Height, bh.Hash.String(), bh.PreHash.String())

	// may change blockHash due to transactions execution
	lostTxs, ccr := core.GetBlockChain().VerifyBlock(bh)
	if -1 == ccr {
		return NewError(fmt.Errorf("blockheader error, height: %d, preHash: %s", bh.Height, bh.PreHash.String()), "ccm", r.RoundNumber(), "", nil)
	}

	//normalPieceVerify
	if 0 == len(lostTxs) {
		notify.BUS.UnSubscribe(notify.TransactionGotAddSucc, r)

		// decide whether send block
		seed := big.NewInt(0).SetBytes(bh.Hash.Bytes()).Uint64()
		index := seed % uint64(r.group.GetMemberCount())
		id := r.group.GetMemberID(int(index)).GetBigInt()
		r.isSend = id.Cmp(r.mi.GetBigInt()) == 0

		if err := r.checkBlockExisted(); err != nil {
			return err
		}

		r.normalPieceVerify()
		hashString := bh.Hash.String()

		r.changedId <- hashString

		r.logger.Infof("round0 changeId, from %s to %s", r.partyId, hashString)
		r.partyId = hashString
		r.canProcessed = true
	} else {
		r.lostTxs = make(map[common.Hashes]byte)
		for _, hash := range lostTxs {
			r.lostTxs[hash] = 0
		}
		r.logger.Warnf("lostTxs waiting, height: %d, id: %s, preHash: %s, len: %d", bh.Height, r.partyId, bh.PreHash.String(), len(lostTxs))
		notify.BUS.Subscribe(notify.TransactionGotAddSucc, r)
	}

	return nil
}

func (r *round0) checkBlockExisted() *Error {
	bh := r.bh

	block := r.blockchain.QueryBlockByHash(bh.Hash)
	if nil == block {
		return nil
	}

	r.logger.Warnf("block has generated. skip next rounds. hash: %s, id: %s, isSend: %v", r.bh.Hash.String(), r.partyId, r.isSend)

	if r.isSend {
		r.broadcastNewBlock(*block)
	}

	return NewError(fmt.Errorf("block already existed, height: %d, hash: %s", bh.Height, bh.Hash.String()), "ccm", r.RoundNumber(), "", nil)
}

func (r *round0) normalPieceVerify() {
	gid := groupsig.DeserializeID(r.bh.GroupId)
	var cvm model.ConsensusVerifyMessage
	cvm.BlockHash = r.bh.Hash
	skey := r.getSignKey(gid)

	if signInfo, ok := model.NewSignInfo(skey, r.mi, &cvm); ok {
		cvm.SignInfo = signInfo
		r.logger.Debugf("round0 sendVerifiedCast, hash: %s, group: %s, sign: %s", cvm.BlockHash.String(), gid.GetHexString(), cvm.SignInfo.GetSignature().GetHexString())
		cvm.GenRandomSign(skey, r.preBH.Random)
		r.netServer.SendVerifiedCast(&cvm, gid)
	} else {
		r.logger.Errorf("genSign fail, sk=%v %v", skey.ShortS(), r.belongGroups.BelongGroup(gid))
	}
}

// getSignKey get the signature private key of the miner in a certain group
func (r *round0) getSignKey(gid groupsig.ID) groupsig.Seckey {
	if jg := r.belongGroups.GetJoinedGroupInfo(gid); jg != nil {
		return jg.SignSecKey
	}
	return groupsig.Seckey{}
}

func (r *round0) CanAccept(msg model.ConsensusMessage) int {
	msgId := msg.GetMessageID()
	if _, ok := r.processed[msgId]; ok {
		return -1
	}
	if _, ok := r.futureMessages[msgId]; ok {
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

func (r *round0) NextRound() Round {
	r.Close()
	r.started = false

	r.canProcessed = false
	r.number = 1
	return &round1{round0: r}
}

func (r *round0) getBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	b := r.blockchain.QueryBlockByHash(hash)
	if b != nil {
		return b.Header
	}
	return nil
}

func (r *round0) HandleNetMessage(topic string, message notify.Message) {
	switch topic {
	case notify.BlockAddSucc:
		r.onBlockAddSuccess(message)
	case notify.TransactionGotAddSucc:
		r.onMissTxAddSucc(message)
	}
}

func (r *round0) onBlockAddSuccess(message notify.Message) {
	r.lock.Lock()
	defer r.lock.Unlock()

	block := message.GetData().(types.Block)
	bh := block.Header

	if bh.Height > r.bh.Height {
		r.errChan <- NewError(fmt.Errorf("higher block on the chain, %d > %d", bh.Height, r.bh.Height), "ccm", r.RoundNumber(), "", nil)
		return
	}

	if 0 != bytes.Compare(bh.Hash.Bytes(), r.bh.PreHash.Bytes()) {
		return
	}

	notify.BUS.UnSubscribe(notify.BlockAddSucc, r)
	r.logger.Warnf("preHash waiting successfully, %s, height: %d", bh.Hash.String(), r.bh.Height)
	r.preBH = bh
	if err := r.afterPreArrived(); nil != err {
		r.errChan <- err
	}
}

func (r *round0) onMissTxAddSucc(message notify.Message) {
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
		r.logger.Warnf("lostTxs waiting again, height: %d, id: %s, preHash: %s, len: %d", r.bh.Height, r.partyId, r.bh.PreHash.String(), len(r.lostTxs))
		return
	} else {
		r.logger.Warnf("lostTxs waiting successfully, height: %d, id: %s, preHash: %s", r.bh.Height, r.partyId, r.bh.PreHash.String())
	}

	err := r.checkBlock()
	if nil != err {
		r.errChan <- err
	}
}

// send block
func (r *round0) broadcastNewBlock(block types.Block) {
	bh := block.Header
	cbm := &model.ConsensusBlockMessage{
		Block: block,
	}
	r.netServer.BroadcastNewBlock(cbm)

	r.logger.Infof("broadcast block, height: %d, hash: %s", bh.Height, bh.Hash.String())
}
