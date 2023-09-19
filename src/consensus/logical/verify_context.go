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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package logical

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	CBCS_IDLE int32 = iota
	CBCS_CASTING
	CBCS_BLOCKED
	CBCS_BROADCAST
	CBCS_TIMEOUT
)

type CAST_BLOCK_MESSAGE_RESULT int8

const (
	CBMR_PIECE_NORMAL CAST_BLOCK_MESSAGE_RESULT = iota
	CBMR_PIECE_LOSINGTRANS
	CBMR_THRESHOLD_SUCCESS
	CBMR_THRESHOLD_FAILED
	CBMR_IGNORE_REPEAT
	CBMR_IGNORE_KING_ERROR
	CBMR_STATUS_FAIL
	CBMR_ERROR_UNKNOWN
	CBMR_CAST_SUCCESS
	CBMR_BH_HASH_DIFF
	CBMR_VERIFY_TIMEOUT
	CBMR_SLOT_INIT_FAIL
	CBMR_SLOT_REPLACE_FAIL
	CBMR_SIGNED_MAX_QN
	CBMR_SIGN_VERIFY_FAIL
)

func CBMR_RESULT_DESC(ret CAST_BLOCK_MESSAGE_RESULT) string {
	switch ret {
	case CBMR_PIECE_NORMAL:
		return "CBMR_PIECE_NORMAL"
	case CBMR_PIECE_LOSINGTRANS:
		return "CBMR_PIECE_LOSINGTRANS"
	case CBMR_THRESHOLD_SUCCESS:
		return "CBMR_THRESHOLD_SUCCESS"
	case CBMR_THRESHOLD_FAILED:
		return "CBMR_THRESHOLD_FAILED"
	case CBMR_IGNORE_KING_ERROR:
		return "CBMR_IGNORE_KING_ERROR"
	case CBMR_STATUS_FAIL:
		return "CBMR_STATUS_FAIL"
	case CBMR_IGNORE_REPEAT:
		return "CBMR_IGNORE_REPEAT"
	case CBMR_CAST_SUCCESS:
		return "CBMR_CAST_SUCCESS"
	case CBMR_BH_HASH_DIFF:
		return "CBMR_BH_HASH_DIFF"
	case CBMR_VERIFY_TIMEOUT:
		return "CBMR_VERIFY_TIMEOUT"
	case CBMR_SLOT_INIT_FAIL:
		return "CBMR_SLOT_INIT_FAIL"
	case CBMR_SLOT_REPLACE_FAIL:
		return "CBMR_SLOT_REPLACE_FAIL"
	case CBMR_SIGNED_MAX_QN:
		return "CBMR_SIGNED_MAX_QN"

	}
	return strconv.FormatInt(int64(ret), 10)
}

const (
	TRANS_INVALID_SLOT int8 = iota
	TRANS_DENY
	TRANS_ACCEPT_NOT_FULL
	TRANS_ACCEPT_FULL_THRESHOLD
	TRANS_ACCEPT_FULL_PIECE
)

func TRANS_ACCEPT_RESULT_DESC(ret int8) string {
	switch ret {
	case TRANS_INVALID_SLOT:
		return "TRANS_INVALID_SLOT"
	case TRANS_DENY:
		return "TRANS_DENY"
	case TRANS_ACCEPT_NOT_FULL:
		return "TRANS_ACCEPT_NOT_FULL"
	case TRANS_ACCEPT_FULL_PIECE:
		return "TRANS_ACCEPT_FULL_PIECE"
	case TRANS_ACCEPT_FULL_THRESHOLD:
		return "TRANS_ACCEPT_FULL_THRESHOLD"
	}
	return strconv.FormatInt(int64(ret), 10)
}

type QN_QUERY_SLOT_RESULT int

type VerifyContext struct {
	prevBH          *types.BlockHeader
	blockHash       common.Hash
	createTime      time.Time
	expireTime      time.Time
	consensusStatus int32
	slot            *SlotContext
	broadcastSlot   *SlotContext

	blockCtx  *BlockContext
	signedNum int32
	lock      sync.RWMutex
}

func newVerifyContext(bc *BlockContext, expire time.Time, preBH *types.BlockHeader, blockHash common.Hash) *VerifyContext {
	ctx := &VerifyContext{
		prevBH:          preBH,
		blockHash:       blockHash,
		blockCtx:        bc,
		expireTime:      expire,
		createTime:      utility.GetTime(),
		consensusStatus: CBCS_CASTING,
	}
	return ctx
}

func (vc *VerifyContext) increaseSignedNum() {
	atomic.AddInt32(&vc.signedNum, 1)
}

func (vc *VerifyContext) isCasting() bool {
	status := atomic.LoadInt32(&vc.consensusStatus)
	return !(status == CBCS_IDLE || status == CBCS_TIMEOUT)
}

func (vc *VerifyContext) castSuccess() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == CBCS_BLOCKED
}
func (vc *VerifyContext) broadCasted() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == CBCS_BROADCAST
}

func (vc *VerifyContext) markTimeout() {
	if !vc.castSuccess() && !vc.broadCasted() {
		atomic.StoreInt32(&vc.consensusStatus, CBCS_TIMEOUT)
	}
}

func (vc *VerifyContext) markCastSuccess() {
	atomic.StoreInt32(&vc.consensusStatus, CBCS_BLOCKED)
}

func (vc *VerifyContext) markBroadcast() bool {
	return atomic.CompareAndSwapInt32(&vc.consensusStatus, CBCS_BLOCKED, CBCS_BROADCAST)
}

// 铸块是否过期
func (vc *VerifyContext) castExpire() bool {
	return false
}

func (vc *VerifyContext) baseCheck(bh *types.BlockHeader, sender groupsig.ID) (slot *SlotContext, err error) {
	if vc.castSuccess() || vc.broadCasted() {
		err = fmt.Errorf("already casted")
		return
	}
	if vc.castExpire() {
		vc.markTimeout()
		err = fmt.Errorf("timeout: " + vc.expireTime.String())
		return
	}
	slot = vc.slot
	if slot != nil {
		if slot.GetSlotStatus() >= SS_RECOVERD {
			err = fmt.Errorf("slot cannot accept piece，status: %v", slot.slotStatus)
			return
		}
		if _, ok := slot.gSignGenerator.GetWitnessSign(sender); ok {
			err = fmt.Errorf("duplicated: %v", sender.ShortS())
			return
		}
	}

	return
}

func (vc *VerifyContext) prepareSlot(bh *types.BlockHeader, blog *bizLog) (*SlotContext, error) {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	if sc := vc.slot; sc != nil {
		blog.log("prepareSlot find exist, status %v", sc.GetSlotStatus())
		return sc, nil
	} else {
		sc = createSlotContext(bh, vc.blockCtx.threshold())
		//sc.init(bh)
		vc.slot = sc
		return sc, nil
	}
}

func (vc *VerifyContext) UserVerified(bh *types.BlockHeader, signData *model.SignInfo, pk groupsig.Pubkey, slog *slowLog) (ret CAST_BLOCK_MESSAGE_RESULT, err error) {
	blog := newBizLog("UserVerified")

	slog.addStage("prePareSlot")
	slot, err := vc.prepareSlot(bh, blog)
	if err != nil {
		blog.log("prepareSlot fail, err %v", err)
		return CBMR_ERROR_UNKNOWN, fmt.Errorf("prepareSlot fail, err %v", err)
	}
	slog.endStage()

	slog.addStage("initIfNeeded")
	slot.initIfNeeded()
	slog.endStage()

	if slot.IsFailed() {
		return CBMR_STATUS_FAIL, fmt.Errorf("slot fail")
	}
	if _, err2 := vc.baseCheck(bh, signData.GetSignerID()); err2 != nil {
		err = err2
		return
	}
	isProposal := slot.castor.IsEqual(signData.GetSignerID())

	if isProposal {
		slog.addStage("vCastorSign")
		b := signData.VerifySign(pk)
		slog.endStage()

		if !b {
			err = fmt.Errorf("verify castorsign fail, id %v, pk %v", signData.GetSignerID().ShortS(), pk.ShortS())
			return
		}

	} else {
		slog.addStage("vMemSign")
		b := signData.VerifySign(pk)
		slog.endStage()

		if !b {
			err = fmt.Errorf("verify sign fail, id %v, pk %v, sig %v hash %v", signData.GetSignerID().ShortS(), pk.GetHexString(), signData.GetSignature().GetHexString(), signData.GetDataHash().Hex())
			return
		}
		sig := groupsig.DeserializeSign(bh.Random)
		if sig == nil || sig.IsNil() {
			err = fmt.Errorf("deserialize bh random fail, random %v", bh.Random)
			return
		}
		slog.addStage("vMemRandSign")
		b = groupsig.VerifySig(pk, vc.prevBH.Random, *sig)
		slog.endStage()

		if !b {
			err = fmt.Errorf("random sign verify fail")
			return
		}
	}

	if isProposal {
		return CBMR_PIECE_NORMAL, nil
	}
	return slot.AcceptVerifyPiece(bh, signData)
}

func (vc *VerifyContext) AcceptTrans(slot *SlotContext, ths []common.Hashes) int8 {

	if !slot.IsValid() {
		return TRANS_INVALID_SLOT
	}
	accept := slot.AcceptTrans(ths)
	if !accept {
		return TRANS_DENY
	}
	if slot.HasTransLost() {
		return TRANS_ACCEPT_NOT_FULL
	}
	st := slot.GetSlotStatus()

	if st == SS_RECOVERD || st == SS_VERIFIED {
		return TRANS_ACCEPT_FULL_THRESHOLD
	} else {
		return TRANS_ACCEPT_FULL_PIECE
	}
}

func (vc *VerifyContext) Clear() {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	vc.slot = nil
	vc.broadcastSlot = nil
}

func (vc *VerifyContext) shouldRemove(topHeight uint64) bool {
	return vc.prevBH == nil || vc.prevBH.Height+10 < topHeight
}

func (vc *VerifyContext) checkBroadcast() *SlotContext {
	if !vc.castSuccess() {
		//blog.log("not success st=%v", vc.consensusStatus)
		return nil
	}

	now := utility.GetTime()
	if now.Sub(vc.createTime).Seconds() < float64(model.Param.MaxWaitBlockTime) {
		//blog.log("not the time, creatTime %v, now %v, since %v", vc.createTime, utility.GetTime(), time.Since(vc.createTime).String())
		return nil
	}

	return vc.slot
}
