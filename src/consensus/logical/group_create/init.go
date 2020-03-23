package group_create

import (
	"x/src/middleware/log"
	"x/src/core"
	"x/src/common"
	"x/src/consensus/groupsig"
	"sync"
	"x/src/consensus/model"
	"x/src/consensus/net"
	"x/src/consensus/access"
	"time"
	"github.com/hashicorp/golang-lru"
)

var groupCreateLogger log.Logger
var groupCreateDebugLogger log.Logger

var GroupCreateProcessor groupCreateProcessor

type groupCreateProcessor struct {
	minerInfo model.SelfMinerInfo
	context   *createGroupContext

	createGroupCache *lru.Cache //key:create group hash,value:create group base height
	//数组循环使用，用来存储已经创建过的组的高度
	//todo 有木有更好的方案？
	createdHeights      [50]uint64 // Identifies whether the group height has already been created
	createdHeightsIndex int

	groupInitContextCache groupInitContextCache
	joinedGroupStorage    *access.JoinedGroupStorage
	groupSignCollectorMap sync.Map // id==>group hash,value==>GroupPubkeyCollector

	minerReader   *access.MinerPoolReader
	groupAccessor *access.GroupAccessor

	blockChain core.BlockChain
	groupChain core.GroupChain

	NetServer net.NetworkServer
	lock      sync.RWMutex
}

func (p *groupCreateProcessor) Init(minerInfo model.SelfMinerInfo, joinedGroupStorage *access.JoinedGroupStorage) {
	groupCreateLogger = log.GetLoggerByIndex(log.GroupCreateLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	groupCreateDebugLogger = log.GetLoggerByIndex(log.GroupCreateDebugLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	p.minerInfo = minerInfo
	p.createdHeightsIndex = 0

	p.createGroupCache = common.CreateLRUCache(100)

	p.joinedGroupStorage = joinedGroupStorage
	p.groupSignCollectorMap = sync.Map{}
	p.groupInitContextCache = newGroupInitContextCache()

	p.minerReader = access.NewMinerPoolReader(core.MinerManagerImpl)
	access.InitPubkeyPool(p.minerReader)
	p.groupAccessor = access.NewGroupAccessor(core.GetGroupChain())

	p.blockChain = core.GetBlockChain()
	p.groupChain = core.GetGroupChain()
	p.NetServer = net.NewNetworkServer()

	p.lock = sync.RWMutex{}
}

// getMemberSignPubKey get the signature public key of the member in the group
func (p *groupCreateProcessor) GetMemberSignPubKey(groupId groupsig.ID, minerId groupsig.ID) (pk groupsig.Pubkey, ok bool) {
	if jg := p.joinedGroupStorage.GetJoinedGroupInfo(groupId); jg != nil {
		pk, ok = jg.GetMemberSignPK(minerId)
		if !ok && !p.minerInfo.ID.IsEqual(minerId) {
			p.askSignPK(minerId, groupId)
		}
	}
	return
}

func (p *groupCreateProcessor) getInGroupSignSecKey(groupId groupsig.ID) groupsig.Seckey {
	if joinedGroup := p.joinedGroupStorage.GetJoinedGroupInfo(groupId); joinedGroup != nil {
		return joinedGroup.SignSecKey
	}
	return groupsig.Seckey{}
}

func (p *groupCreateProcessor) OnGroupAddSuccess(g *model.GroupInfo) {
	ctx := p.context
	if ctx != nil && ctx.groupInitInfo != nil && ctx.groupInitInfo.GroupHash() == g.GroupInitInfo.GroupHash() {
		top := p.blockChain.Height()
		groupCreateLogger.Infof("onGroupAddSuccess info=%v, gHash=%v, gid=%v, costHeight=%v", ctx.String(), g.GroupInitInfo.GroupHash().ShortS(), g.GroupID.ShortS(), top-ctx.createTopHeight)
		p.removeContext()
		groupCreateDebugLogger.Infof("Group create success. Group hash:%s, group id:%s\n", ctx.groupInitInfo.GroupHash().String(), g.GroupID.GetHexString())
	}
	//p.joiningGroups.Clean(sgi.GInfo.GroupHash())
	//p.globalGroups.removeInitedGroup(sgi.GInfo.GroupHash())

	//p.groupInitContextCache.Clean(g.GroupInitInfo.GroupHash())
	p.groupSignCollectorMap.Delete(g.GroupInitInfo.GroupHash())
	if p.joinedGroupStorage.BelongGroup(g.GroupID) {
		p.groupInitContextCache.RemoveContext(g.GroupInitInfo.GroupHash())
		//退出DUMMY 网络
		p.NetServer.ReleaseGroupNet(g.GroupInitInfo.GroupHash().String())
	}
	p.createGroupCache.Remove(g.GroupInitInfo.GroupHash())

}

func (p *groupCreateProcessor) removeContext() {
	p.context = nil
}

func (p *groupCreateProcessor) ReleaseGroups(topHeight uint64) (needDimissGroups []groupsig.ID) {
	//在当前高度解散的组不应立即从缓存删除，延缓一个建组周期删除。保证该组解散前夕建的块有效
	groups := p.groupAccessor.GetDismissGroups(topHeight - model.Param.CreateGroupInterval)
	ids := make([]groupsig.ID, 0)
	for _, g := range groups {
		ids = append(ids, g.GroupID)
	}

	if len(ids) > 0 {
		groupCreateLogger.Debugf("clean group %v\n", len(ids))
		needDimissGroups = ids
		p.groupAccessor.RemoveGroupsFromCache(ids)
		p.joinedGroupStorage.LeaveGroups(ids)
		for _, g := range groups {
			gid := g.GroupID
			//quit group net.real group:group id
			p.NetServer.ReleaseGroupNet(gid.GetHexString())
			p.groupInitContextCache.RemoveContext(g.GroupInitInfo.GroupHash())
		}
	}

	//释放超时未建成组的组网络和相应的dummy组
	invalidDummyGroups := make([]common.Hash, 0)
	p.groupInitContextCache.forEach(func(gc *groupInitContext) bool {
		if gc.groupInitInfo == nil || gc.status == GisGroupInitDone {
			return true
		}
		groupInitInfo := gc.groupInitInfo
		gHash := groupInitInfo.GroupHash()
		//已经达到组可以开始工作的高度，但是组还没建成
		if groupInitInfo.ReadyTimeout(topHeight) {
			if topHeight < groupInitInfo.GroupHeader.ReadyHeight+model.Param.CreateGroupInterval {
				p.tryReqSharePiece(gc)
			} else {
				invalidDummyGroups = append(invalidDummyGroups, gHash)
			}
		}
		return true
	})
	for _, groupHash := range invalidDummyGroups {
		groupCreateLogger.Debugf("DissolveGroupNet dummyGroup from joiningGroups gHash %v", groupHash.ShortS())
		//quit group net.group hash
		p.NetServer.ReleaseGroupNet(groupHash.Hex())
		p.groupInitContextCache.RemoveContext(groupHash)
	}

	gctx := p.context
	if gctx != nil && gctx.timeout(topHeight) {
		groupCreateLogger.Infof("releaseRoutine:info=%v, elapsed %v. ready timeout.", gctx.String(), time.Since(gctx.createTime))
		p.removeContext()
	}

	p.forEach(func(ig *groupPubkeyCollector) bool {
		hash := ig.groupInitInfo.GroupHash()
		if ig.groupInitInfo.ReadyTimeout(topHeight) {
			groupCreateLogger.Debugf("remove groupPubkeyCollector, gHash %v", hash.ShortS())
			//quit group net.group hash
			p.NetServer.ReleaseGroupNet(hash.Hex())
			p.groupSignCollectorMap.Delete(hash)
		}
		return true
	})

	//清理超时的签名公钥请求
	cleanSignPkReqRecord()
	return
}

func (p *groupCreateProcessor) tryReqSharePiece(gc *groupInitContext) {
	waitPieceIds := make([]string, 0)
	waitIds := make([]groupsig.ID, 0)
	for _, mem := range gc.groupInitInfo.GroupMembers {
		if !gc.nodeInfo.hasSharePiece(mem) {
			waitPieceIds = append(waitPieceIds, mem.ShortS())
			waitIds = append(waitIds, mem)
		}
	}

	msg := &model.ReqSharePieceMessage{
		GroupHash: gc.groupInitInfo.GroupHash(),
	}
	groupCreateLogger.Infof("reqSharePieceRoutine:req size %v, ghash=%v", len(waitIds), gc.groupInitInfo.GroupHash().ShortS())
	if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, msg); ok {
		msg.SignInfo = signInfo
		for _, receiver := range waitIds {
			groupCreateLogger.Infof("reqSharePieceRoutine:req share piece msg from %v, ghash=%v", receiver, gc.groupInitInfo.GroupHash().ShortS())
			p.NetServer.ReqSharePiece(msg, receiver)
		}
	} else {
		groupCreateLogger.Infof("gen req sharepiece sign fail, ski=%v %v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.ShortS())
	}

}
