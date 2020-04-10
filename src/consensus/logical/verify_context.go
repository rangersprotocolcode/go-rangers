package logical

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/middleware/types"
	"x/src/utility"
)

const (
	CBCS_IDLE      int32 = iota //非当前组
	CBCS_CASTING                //正在铸块
	CBCS_BLOCKED                //组内已有铸块完成
	CBCS_BROADCAST              //块已广播
	CBCS_TIMEOUT                //组铸块超时
)

type CAST_BLOCK_MESSAGE_RESULT int8 //出块和验证消息处理结果枚举

const (
	CBMR_PIECE_NORMAL      CAST_BLOCK_MESSAGE_RESULT = iota //收到一个分片，接收正常
	CBMR_PIECE_LOSINGTRANS                                  //收到一个分片, 缺失交易
	CBMR_THRESHOLD_SUCCESS                                  //收到一个分片且达到阈值，组签名成功
	CBMR_THRESHOLD_FAILED                                   //收到一个分片且达到阈值，组签名失败
	CBMR_IGNORE_REPEAT                                      //丢弃：重复收到该消息
	CBMR_IGNORE_KING_ERROR                                  //丢弃：king错误
	CBMR_STATUS_FAIL                                        //已经失败的
	CBMR_ERROR_UNKNOWN                                      //异常：未知异常
	CBMR_CAST_SUCCESS                                       //铸块成功
	CBMR_BH_HASH_DIFF                                       //slot已经被替换过了
	CBMR_VERIFY_TIMEOUT                                     //已超时
	CBMR_SLOT_INIT_FAIL                                     //slot初始化失败
	CBMR_SLOT_REPLACE_FAIL                                  //slot初始化失败
	CBMR_SIGNED_MAX_QN                                      //签过更高的qn
	CBMR_SIGN_VERIFY_FAIL                                   //签名错误
)

func CBMR_RESULT_DESC(ret CAST_BLOCK_MESSAGE_RESULT) string {
	switch ret {
	case CBMR_PIECE_NORMAL:
		return "正常分片"
	case CBMR_PIECE_LOSINGTRANS:
		return "交易缺失"
	case CBMR_THRESHOLD_SUCCESS:
		return "达到门限值组签名成功"
	case CBMR_THRESHOLD_FAILED:
		return "达到门限值但组签名失败"
	case CBMR_IGNORE_KING_ERROR:
		return "king错误"
	case CBMR_STATUS_FAIL:
		return "失败状态"
	case CBMR_IGNORE_REPEAT:
		return "重复消息"
	case CBMR_CAST_SUCCESS:
		return "已铸块成功"
	case CBMR_BH_HASH_DIFF:
		return "hash不一致，slot已无效"
	case CBMR_VERIFY_TIMEOUT:
		return "验证超时"
	case CBMR_SLOT_INIT_FAIL:
		return "slot初始化失败"
	case CBMR_SLOT_REPLACE_FAIL:
		return "slot替换失败"
	case CBMR_SIGNED_MAX_QN:
		return "签过更高qn"

	}
	return strconv.FormatInt(int64(ret), 10)
}

const (
	TRANS_INVALID_SLOT          int8 = iota //无效验证槽
	TRANS_DENY                              //拒绝该交易
	TRANS_ACCEPT_NOT_FULL                   //接受交易, 但仍缺失交易
	TRANS_ACCEPT_FULL_THRESHOLD             //接受交易, 无缺失, 验证已达到门限
	TRANS_ACCEPT_FULL_PIECE                 //接受交易, 无缺失, 未达到门限
)

func TRANS_ACCEPT_RESULT_DESC(ret int8) string {
	switch ret {
	case TRANS_INVALID_SLOT:
		return "验证槽无效"
	case TRANS_DENY:
		return "不接收该批交易"
	case TRANS_ACCEPT_NOT_FULL:
		return "接收交易,但仍缺失"
	case TRANS_ACCEPT_FULL_PIECE:
		return "交易收齐,等待分片"
	case TRANS_ACCEPT_FULL_THRESHOLD:
		return "交易收齐,分片已到门限"
	}
	return strconv.FormatInt(int64(ret), 10)
}

type QN_QUERY_SLOT_RESULT int //根据QN查找插槽结果枚举

type VerifyContext struct {
	prevBH          *types.BlockHeader
	blockHash       common.Hash
	createTime      time.Time
	expireTime      time.Time //铸块超时时间
	consensusStatus int32     //铸块状态
	slot            *SlotContext
	broadcastSlot   *SlotContext
	//castedQNs []int64 //自己铸过的qn
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

//铸块是否过期
func (vc *VerifyContext) castExpire() bool {
	//return utility.GetTime().After(vc.expireTime)
	return false
}

func (vc *VerifyContext) baseCheck(bh *types.BlockHeader, sender groupsig.ID) (slot *SlotContext, err error) {
	if vc.castSuccess() || vc.broadCasted() {
		err = fmt.Errorf("已出块")
		return
	}
	if vc.castExpire() {
		vc.markTimeout()
		err = fmt.Errorf("已超时" + vc.expireTime.String())
		return
	}
	slot = vc.slot
	if slot != nil {
		if slot.GetSlotStatus() >= SS_RECOVERD {
			err = fmt.Errorf("slot不接受piece，状态%v", slot.slotStatus)
			return
		}
		if _, ok := slot.gSignGenerator.GetWitnessSign(sender); ok {
			err = fmt.Errorf("重复消息%v", sender.ShortS())
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

//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
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

	//警惕并发
	if slot.IsFailed() {
		return CBMR_STATUS_FAIL, fmt.Errorf("slot fail")
	}
	if _, err2 := vc.baseCheck(bh, signData.GetSignerID()); err2 != nil {
		err = err2
		return
	}
	isProposal := slot.castor.IsEqual(signData.GetSignerID())

	if isProposal { //提案者 // 不可能是提案者了
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
	//如果是提案者，因为提案者没有对块进行签名，则直接返回
	if isProposal {
		return CBMR_PIECE_NORMAL, nil
	}
	return slot.AcceptVerifyPiece(bh, signData)
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
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

//判断该context是否可以删除
func (vc *VerifyContext) shouldRemove(topHeight uint64) bool {
	//未出过块, 但高度已经低于200块, 可以删除
	return vc.prevBH == nil || vc.prevBH.Height+200 < topHeight
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
	//var maxQNSlot *SlotContext
	//
	//vc.lock.RLock()
	//defer vc.lock.RUnlock()
	//qns := make([]uint64, 0)
	//
	//slot := vc.slot
	//if !slot.IsRecovered() {
	//	continue
	//}
	//qns = append(qns, slot.BH.TotalQN)
	//if maxQNSlot == nil {
	//	maxQNSlot = slot
	//} else {
	//	if maxQNSlot.BH.TotalQN < slot.BH.TotalQN {
	//		maxQNSlot = slot
	//	} else if maxQNSlot.BH.TotalQN == slot.BH.TotalQN {
	//		v1 := vrf.VRFProof2Hash(maxQNSlot.BH.ProveValue.Bytes()).Big()
	//		v2 := vrf.VRFProof2Hash(slot.BH.ProveValue.Bytes()).Big()
	//		if v1.Cmp(v2) < 0 {
	//			maxQNSlot = slot
	//		}
	//	}
	//}
	//
	//if maxQNSlot != nil {
	//	blog.log("select max qn=%v, hash=%v, height=%v, hash=%v, all qn=%v", maxQNSlot.BH.TotalQN, maxQNSlot.BH.Hash.ShortS(), maxQNSlot.BH.Height, maxQNSlot.BH.Hash.ShortS(), qns)
	//}
	//return maxQNSlot
}
