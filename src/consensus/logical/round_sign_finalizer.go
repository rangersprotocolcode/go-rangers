package logical

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"math/big"
)

func (r *round3) Update(msg model.ConsensusMessage) *Error {
	return nil
}

func (r *round3) CanAccept(msg model.ConsensusMessage) int {
	return -1
}

func (r *round3) NextRound() Round {
	return nil
}

func (r *round3) Start() *Error {
	bh := r.bh
	r.logger.Debugf("round3 start, hash: %s, height: %d", bh.Hash.String(), bh.Height)

	if r.blockchain.HasBlockByHash(bh.Hash) {
		return NewError(fmt.Errorf("blockheader already existed, height: %d, hash: %s", bh.Height, bh.Hash.String()), "finalizer", r.RoundNumber(), "", nil)
	}

	group, err := r.globalGroups.GetGroupByID(groupsig.DeserializeID(bh.GroupId))
	if nil != err {
		return NewError(fmt.Errorf("fail to get group, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}
	if err := r.checkSignature(group); err != nil {
		return err
	}

	block := r.blockchain.GenerateBlock(*bh)
	if block == nil {
		return NewError(fmt.Errorf("fail to generate block, height: %d, hash: %s", bh.Height, bh.Hash.String()), "finalizer", r.RoundNumber(), "", nil)
	}
	result := r.blockchain.AddBlockOnChain(block)
	if types.AddBlockSucc != result {
		return NewError(fmt.Errorf("fail to add block, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}
	r.logger.Infof("round3 add block, height: %d, hash: %s", bh.Height, bh.Hash.String())

	// send block if nesscessary
	r.broadcastNewBlock(group, *block)
	return nil
}

func (r *round3) broadcastNewBlock(group *model.GroupInfo, block types.Block) {
	bh := block.Header
	seed := big.NewInt(0).SetBytes(bh.Hash.Bytes()).Uint64()
	index := seed % uint64(group.GetMemberCount())
	id := group.GetMemberID(int(index)).GetBigInt()
	if id.Cmp(r.mi.GetBigInt()) == 0 {
		cbm := &model.ConsensusBlockMessage{
			Block: block,
		}
		r.netServer.BroadcastNewBlock(cbm)
		r.logger.Infof("round3 broadcasted block, height: %d, hash: %s", bh.Height, bh.Hash.String())
	}
}

func (r *round3) checkSignature(group *model.GroupInfo) *Error {
	bh := r.bh
	gpk := group.GetGroupPubKey()
	if !groupsig.VerifySig(gpk, bh.Hash.Bytes(), *groupsig.DeserializeSign(bh.Signature)) {
		return NewError(fmt.Errorf("fail to verify group sign, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}
	if !groupsig.VerifySig(gpk, r.preBH.Random, *groupsig.DeserializeSign(bh.Random)) {
		return NewError(fmt.Errorf("fail to verify random sign, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}

	return nil
}
