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
)

var groupCreateLogger log.Logger

var GroupCreateProcessor *groupCreateProcessor

type groupCreateProcessor struct {
	minerInfo model.MinerInfo
	context   *createGroupContext

	//数组循环使用，用来存储已经创建过的组的高度
	//todo 有木有更好的方案？
	createdHeights      [50]uint64 // Identifies whether the group height has already been created
	createdHeightsIndex int

	joinedGroupStorage    *access.JoinedGroupStorage
	groupSignCollectorMap sync.Map // id==>group hash,value==>GroupPubkeyCollector
	groupInitContextCache groupInitContextCache

	minerReader   *access.MinerPoolReader
	groupAccessor *access.GroupAccessor

	blockChain core.BlockChain
	groupChain core.GroupChain

	NetServer net.NetworkServer
	lock      sync.RWMutex
}

func (p *groupCreateProcessor) Init(minerInfo model.MinerInfo) {
	groupCreateLogger = log.GetLoggerByIndex(log.GroupCreateLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	p.minerInfo = minerInfo
	p.createdHeightsIndex = 0

	p.joinedGroupStorage = access.GetJoinedGroupStorageInstance()
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

func (p *groupCreateProcessor) removeContext() {
	p.context = nil
}

// getMemberSignPubKey get the signature public key of the member in the group
func (p *groupCreateProcessor) getMemberSignPubKey(groupId groupsig.ID, minerId groupsig.ID) (pk groupsig.Pubkey, ok bool) {
	if jg := p.joinedGroupStorage.GetJoinedGroupInfo(groupId); jg != nil {
		pk, ok = jg.GetMemberSignPK(minerId)
		if !ok && !p.minerInfo.ID.IsEqual(minerId) {
			p.askSignPK(minerId, groupId)
		}
	}
	return
}

func (p *groupCreateProcessor) onGroupAddSuccess(g *model.GroupInfo) {
	ctx := p.context
	if ctx != nil && ctx.groupInitInfo != nil && ctx.groupInitInfo.GroupHash() == g.GroupInitInfo.GroupHash() {
		top := p.blockChain.Height()
		groupCreateLogger.Infof("onGroupAddSuccess info=%v, gHash=%v, gid=%v, costHeight=%v", ctx.String(), g.GroupInitInfo.GroupHash().ShortS(), g.GroupID.ShortS(), top-ctx.createTopHeight)
		p.removeContext()
	}
}
