package group_create

import (
	"fmt"
	"sync"
	"time"
	"x/src/middleware/types"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/consensus/base"
	"bytes"
)

// status enum of the CreatingGroupContext
const (
	waitingPong = 1 // waitingPong indicates the context is waiting for pong response from nodes
	waitingSign = 2 // waitingSign indicates the context is waiting for the group signature for the group-creating proposal
	sendInit    = 3 // sendInit indicates the context has send group init message to the members who make up the new group
)

//组创建基本信息，第一步就可以得到生成
type createGroupBaseInfo struct {
	parentGroupInfo *model.GroupInfo   // the parent group info
	baseBlockHeader *types.BlockHeader // the blockHeader the group-create routine based on
	baseGroup       *types.Group       // the last group of the groupchain
	candidates      []groupsig.ID      // the legal candidates
}

func newCreateGroupBaseInfo(sgi *model.GroupInfo, baseBH *types.BlockHeader, baseG *types.Group, cands []groupsig.ID) *createGroupBaseInfo {
	return &createGroupBaseInfo{
		parentGroupInfo: sgi,
		baseBlockHeader: baseBH,
		baseGroup:       baseG,
		candidates:      cands,
	}
}

func (ctx *createGroupBaseInfo) hasCandidate(uid groupsig.ID) bool {
	for _, id := range ctx.candidates {
		if id.IsEqual(uid) {
			return true
		}
	}
	return false
}

func (ctx *createGroupBaseInfo) readyHeight() uint64 {
	return ctx.baseBlockHeader.Height + model.Param.GroupReadyGap
}

//readyTimeout
func (ctx *createGroupBaseInfo) timeout(h uint64) bool {
	return h >= ctx.readyHeight()
}

// CreatingGroupContext stores the context info when parent group starting group-create routine
type createGroupContext struct {
	createGroupBaseInfo
	status  int8   // the context status(waitingPong,waitingSign,sendInit)
	memMask []byte // each non-zero bit indicates that the candidate at the subscript replied to the ping message and will become a full member of the new-group

	kings       []groupsig.ID // kings selected randomly from the parent group who responsible for node pings and new group proposal
	belongKings bool          // whether the current node is one of the kings

	pingID          string          // identify one ping process
	pongMap         map[string]byte // pong response received from candidates
	createTime      time.Time       // create time for the context, used to make local timeout judgments
	createTopHeight uint64          // the blockchain height when starting the group-create routine

	//gInfo
	groupInitInfo      *model.GroupInitInfo      // new group info generated during the routine and will be sent to the new-group members for consensus
	groupSignGenerator *model.GroupSignGenerator // group signature generator

	lock sync.RWMutex
}

func newCreateGroupContext(baseCtx *createGroupBaseInfo, kings []groupsig.ID, isKing bool, top uint64) *createGroupContext {
	pingIDBytes := baseCtx.baseBlockHeader.Hash.Bytes()
	pingIDBytes = append(pingIDBytes, baseCtx.baseGroup.Id...)
	cg := &createGroupContext{
		createGroupBaseInfo: *baseCtx,
		kings:               kings,
		status:              waitingPong,
		createTime:          time.Now(),
		belongKings:         isKing,
		createTopHeight:     top,
		pingID:              base.Data2CommonHash(pingIDBytes).Hex(),
		pongMap:             make(map[string]byte, 0),
		groupSignGenerator:  model.NewGroupSignGenerator(model.Param.GetGroupK(baseCtx.parentGroupInfo.GetMemberCount())),
	}

	return cg
}

func (ctx *createGroupContext) acceptPiece(from groupsig.ID, sign groupsig.Signature) (accept, recover bool) {
	accept, recover = ctx.groupSignGenerator.AddWitnessSign(from, sign)
	return
}

//pongDeadline
func (ctx *createGroupContext) isPongTimeout(h uint64) bool {
	return h >= ctx.baseBlockHeader.Height+model.Param.GroupWaitPongGap
}

func (ctx *createGroupContext) isKing() bool {
	return ctx.belongKings
}

//addPong
func (ctx *createGroupContext) handlePong(h uint64, uid groupsig.ID) (add bool, size int) {
	if ctx.isPongTimeout(h) {
		return false, ctx.receivedPongCount()
	}
	ctx.lock.Lock()
	defer ctx.lock.Unlock()

	if ctx.hasCandidate(uid) {
		ctx.pongMap[uid.GetHexString()] = 1
		add = true
	}
	size = len(ctx.pongMap)
	return
}

//pongSize
func (ctx *createGroupContext) receivedPongCount() int {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return len(ctx.pongMap)
}

func (ctx *createGroupContext) getStatus() int8 {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return ctx.status
}

func (ctx *createGroupContext) setStatus(st int8) {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	ctx.status = st
}

//生成组初始化信息(mask groupheader, 成员)
func (ctx *createGroupContext) genGroupInitInfo(h uint64) bool {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()

	if ctx.groupInitInfo != nil {
		return true
	}
	if len(ctx.pongMap) == len(ctx.candidates) || ctx.isPongTimeout(h) {
		mask := ctx.generateMemberMask()
		gInfo := ctx.createGroupInitInfo(mask)

		ctx.groupInitInfo = gInfo
		ctx.memMask = mask
		return true
	}

	return false
}

func (ctx *createGroupContext) generateMemberMask() (mask []byte) {
	mask = make([]byte, (len(ctx.candidates)+7)/8)

	for i, id := range ctx.candidates {
		b := mask[i/8]
		if _, ok := ctx.pongMap[id.GetHexString()]; ok {
			b |= 1 << byte(i%8)
			mask[i/8] = b
		}
	}
	return
}

func (ctx *createGroupBaseInfo) createGroupInitInfo(mask []byte) *model.GroupInitInfo {
	memIds := ctx.recoverMemberSet(mask)
	gh := ctx.createGroupHeader(memIds)

	var memberBuffer bytes.Buffer
	for _, member := range memIds {
		memberBuffer.WriteString(member.ShortS() + ",")
	}
	groupCreateLogger.Debugf("member mask:%v.After mask,members:%s", mask, memberBuffer.String())

	return &model.GroupInitInfo{
		GroupHeader:  gh,
		GroupMembers: memIds,
	}
}

func (ctx *createGroupBaseInfo) recoverMemberSet(mask []byte) (ids []groupsig.ID) {
	ids = make([]groupsig.ID, 0)
	for i, id := range ctx.candidates {
		b := mask[i/8]
		if (b & (1 << byte(i%8))) != 0 {
			ids = append(ids, id)
		}
	}
	return
}
func (ctx *createGroupBaseInfo) createGroupHeader(memIds []groupsig.ID) *types.GroupHeader {
	pid := ctx.parentGroupInfo.GroupID
	theBH := ctx.baseBlockHeader
	gn := fmt.Sprintf("%s-%v", pid.GetHexString(), theBH.Height)
	extends := fmt.Sprintf("baseBlock:%v|%v|%v", theBH.Hash.Hex(), theBH.CurTime, theBH.Height)

	gh := &types.GroupHeader{
		Parent:       ctx.parentGroupInfo.GroupID.Serialize(),
		PreGroup:     ctx.baseGroup.Id,
		Name:         gn,
		Authority:    777,
		BeginTime:    theBH.CurTime,
		CreateHeight: theBH.Height,
		ReadyHeight:  ctx.readyHeight(),
		WorkHeight:   theBH.Height + model.Param.GroupWorkGap,
		MemberRoot:   model.GenGroupMemberRoot(memIds),
		Extends:      extends,
	}
	gh.DismissHeight = gh.WorkHeight + model.Param.GroupworkDuration

	gh.Hash = gh.GenHash()
	return gh
}

func (ctx *createGroupContext) String() string {
	return fmt.Sprintf("baseHeight=%v, topHeight=%v, candidates=%v, isKing=%v, parentGroup=%v, pongs=%v, elapsed=%v",
		ctx.baseBlockHeader.Height, ctx.createTopHeight, len(ctx.candidates), ctx.isKing(), ctx.parentGroupInfo.GroupID.ShortS(), ctx.receivedPongCount(), time.Since(ctx.createTime).String())
}
