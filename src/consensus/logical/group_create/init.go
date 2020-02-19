package logical

import (
	"x/src/middleware/log"
	"x/src/core"
	"x/src/common"
	"x/src/consensus/groupsig"
	"sync"
	"x/src/consensus/model"
	"x/src/consensus/net"
	"strings"
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

	p.joinedGroupStorage = access.InitJoinedGroupStorage()
	p.groupSignCollectorMap = sync.Map{}
	p.groupInitContextCache = newGroupInitContextCache()

	p.minerReader = access.NewMinerPoolReader(core.MinerManagerImpl, groupCreateLogger)
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




func (p *Processor) genBelongGroupStoreFile() string {
	storeFile := p.conf.GetString(ConsensusConfSection, "groupstore", "")
	if strings.TrimSpace(storeFile) == "" {
		storeFile = "groupstore" + p.conf.GetString("instance", "index", "")
	}
	return storeFile
}

// getSignKey get the signature private key of the miner in a certain group
func (p Processor) getSignKey(gid groupsig.ID) groupsig.Seckey {
	if jg := p.belongGroups.getJoinedGroup(gid); jg != nil {
		return jg.SignKey
	}
	return groupsig.Seckey{}
}

func (p *groupCreateProcessor) acceptGroup(staticGroup *StaticGroupInfo) {
	add := p.globalGroups.AddStaticGroup(staticGroup)
	blog := newBizLog("acceptGroup")
	blog.debug("Add to Global static groups, result=%v, groups=%v.", add, p.globalGroups.GetGroupSize())
	if staticGroup.MemExist(p.GetMinerID()) {
		p.prepareForCast(staticGroup)
	}
}

func (gm *GroupManager) onGroupAddSuccess(g *StaticGroupInfo) {
	ctx := gm.getContext()
	if ctx != nil && ctx.gInfo != nil && ctx.gInfo.GroupHash() == g.GInfo.GroupHash() {
		top := gm.mainChain.Height()
		groupLogger.Infof("onGroupAddSuccess info=%v, gHash=%v, gid=%v, costHeight=%v", ctx.logString(), g.GInfo.GroupHash().ShortS(), g.GroupID.ShortS(), top-ctx.createTopHeight)
		gm.removeContext()
	}
}

