package logical

import (
	"x/src/consensus/groupsig"

	"encoding/hex"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"sync"
	"sync/atomic"
	"x/src/common"
	"x/src/consensus/access"
	"x/src/consensus/model"
	"x/src/consensus/net"
	"x/src/consensus/ticker"
	"x/src/core"
	"x/src/middleware/notify"
	"x/src/middleware/types"
)

//见证人处理器
type Processor struct {
	ready bool //是否已初始化完成
	conf  common.ConfManager
	mi    *model.SelfMinerInfo //////和组无关的矿工信息

	blockContexts    *CastBlockContexts   //组ID->组铸块上下文
	futureVerifyMsgs *FutureMessageHolder //存储缺失前一块的验证消息

	verifyMsgCaches *lru.Cache //缓存验证消息

	//joiningGroups *JoiningGroups //已加入未完成初始化的组(组初始化完成上链后，不再需要)。组内成员数据过程数据。
	belongGroups *access.JoinedGroupStorage //当前ID参与了哪些(已上链，可铸块的)组, 组id_str->组内私密数据（组外不可见或加速缓存）
	globalGroups *access.GroupAccessor      //全网组静态信息（包括待完成组初始化的组，即还没有组ID只有DUMMY ID的组）

	minerReader *access.MinerPoolReader
	//groupManager *GroupManager

	Ticker *ticker.GlobalTicker //全局定时器, 组初始化完成后启动
	vrf    atomic.Value         //vrfWorker

	MainChain  core.BlockChain
	GroupChain core.GroupChain
	NetServer  net.NetworkServer

	lock sync.Mutex
}

func (p Processor) getPrefix() string {
	return p.GetMinerID().ShortS()
}

//私密函数，用于测试，正式版本不提供
//func (p Processor) getMinerInfo() *model.SelfMinerDO {
//	return p.mi
//}
//
//func (p Processor) GetPubkeyInfo() model.PubKeyInfo {
//	return model.NewPubKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultPubKey())
//}

//初始化矿工数据（和组无关）
func (p *Processor) Init(mi model.SelfMinerInfo, conf common.ConfManager, joinedGroupStorage *access.JoinedGroupStorage) bool {
	p.ready = false
	p.lock = sync.Mutex{}
	p.conf = conf
	//p.futureBlockMsgs = NewFutureMessageHolder()
	p.futureVerifyMsgs = NewFutureMessageHolder()

	p.MainChain = core.GetBlockChain()
	p.GroupChain = core.GetGroupChain()
	p.mi = &mi
	p.globalGroups = access.NewGroupAccessor(p.GroupChain)
	//p.joiningGroups = NewJoiningGroups()
	p.belongGroups = joinedGroupStorage
	p.blockContexts = NewCastBlockContexts()
	p.NetServer = net.NewNetworkServer()

	p.minerReader = access.NewMinerPoolReader(core.MinerManagerImpl)
	//pkPoolInit(p.minerReader)

	//p.groupManager = NewGroupManager(p)
	p.Ticker = ticker.GetTickerInstance()

	if stdLogger != nil {
		stdLogger.Debugf("proc(%v) inited 2.\n", p.getPrefix())
		consensusLogger.Infof("ProcessorId:%v", p.getPrefix())
	}

	cache, err := lru.New(300)
	if err != nil {
		panic(err)
	}
	p.verifyMsgCaches = cache

	notify.BUS.Subscribe(notify.BlockAddSucc, p.onBlockAddSuccess)
	notify.BUS.Subscribe(notify.GroupAddSucc, p.onGroupAddSuccess)
	notify.BUS.Subscribe(notify.TransactionGotAddSucc, p.onMissTxAddSucc)
	notify.BUS.Subscribe(notify.AcceptGroup, p.onGroupAccept)
	//notify.BUS.Subscribe(notify.NewBlock, p.onNewBlockReceive)

	return true
}

//取得矿工ID（和组无关）
func (p Processor) GetMinerID() groupsig.ID {
	return p.mi.GetMinerID()
}

func (p Processor) GetMinerInfo() *model.MinerInfo {
	return &p.mi.MinerInfo
}

////验证块的组签名是否正确
//func (p *Processor) verifyGroupSign(msg *model.ConsensusBlockMessage, preBH *types.BlockHeader) bool {
//	b := &msg.Block
//	bh := b.Header
//	var gid groupsig.ID
//	if gid.Deserialize(bh.GroupId) != nil {
//		panic("verifyGroupSign: group id Deserialize failed.")
//	}
//
//	blog := newBizLog("verifyGroupSign")
//	groupInfo := p.GetGroup(gid)
//	if !groupInfo.GroupID.IsValid() {
//		blog.log("get group is nil!, gid=" + gid.ShortS())
//		return false
//	}
//
//	//blog.log("gpk %v, bh hash %v, sign %v, rand %v", groupInfo.GroupPK.ShortS(), bh.Hash.ShortS(), bh.Signature, bh.Random)
//	if !msg.VerifySig(groupInfo.GroupPK, preBH.Random) {
//		blog.log("verifyGroupSig fail")
//		return false
//	}
//	return true
//}

//检查提案节点是否合法
func (p *Processor) isCastLegal(bh *types.BlockHeader, preHeader *types.BlockHeader) (ok bool, group *model.GroupInfo, err error) {
	blog := newBizLog("isCastLegal")
	castor := groupsig.DeserializeID(bh.Castor)
	minerDO := p.minerReader.GetProposeMiner(castor, preHeader.StateTree)
	if minerDO == nil {
		err = fmt.Errorf("minerDO is nil, id=%v", castor.ShortS())
		return
	}
	if !minerDO.CanCastAt(bh.Height) {
		err = fmt.Errorf("miner can't cast at height, id=%v, height=%v(%v-%v)", castor.ShortS(), bh.Height, minerDO.ApplyHeight, minerDO.AbortHeight)
		return
	}
	totalStake := p.minerReader.GetTotalStake(preHeader.Height)
	blog.log("totalStake %v", totalStake)
	if ok2, err2 := verifyBlockVRF(bh, preHeader, minerDO, totalStake); !ok2 {
		err = fmt.Errorf("vrf verify block fail, err=%v", err2)
		return
	}

	var gid = groupsig.DeserializeID(bh.GroupId)

	selectGroupIdFromCache := p.CalcVerifyGroupFromCache(preHeader, bh.CurTime, bh.Height)

	if selectGroupIdFromCache == nil {
		err = common.ErrSelectGroupNil
		stdLogger.Debugf("selectGroupId is nil")
		return
	}
	var verifyGid = *selectGroupIdFromCache

	if !selectGroupIdFromCache.IsEqual(gid) { //有可能组已经解散，需要再从链上取
		selectGroupIdFromChain := p.CalcVerifyGroupFromChain(preHeader, bh.CurTime, bh.Height)
		if selectGroupIdFromChain == nil {
			err = common.ErrSelectGroupNil
			return
		}
		//若内存与链不一致，则启动更新
		if !selectGroupIdFromChain.IsEqual(*selectGroupIdFromCache) {
			go p.updateGlobalGroups()
		}
		if !selectGroupIdFromChain.IsEqual(gid) {
			err = common.ErrSelectGroupInequal
			stdLogger.Debugf("selectGroupId from both cache and chain not equal, expect %v, receive %v.bh hash:%s,height:%d,castor:%s", selectGroupIdFromChain.ShortS(), gid.ShortS(), bh.Hash.String(), bh.Height, hex.EncodeToString(bh.Castor))
			return
		}
		verifyGid = *selectGroupIdFromChain
	}

	group = p.GetGroup(verifyGid) //取得合法的铸块组
	if !group.GroupID.IsValid() {
		err = fmt.Errorf("selectedGroup is not valid, expect gid=%v, real gid=%v", verifyGid.ShortS(), group.GroupID.ShortS())
		return
	}

	ok = true
	return
}

func (p *Processor) getMinerPos(gid groupsig.ID, uid groupsig.ID) int32 {
	sgi := p.GetGroup(gid)
	return int32(sgi.GetMemberPosition(uid))
}

///////////////////////////////////////////////////////////////////////////////
////取得自己参与的某个铸块组的公钥片段（聚合一个组所有成员的公钥片段，可以生成组公钥）
//func (p Processor) GetMinerPubKeyPieceForGroup(gid groupsig.ID) groupsig.Pubkey {
//	var pub_piece groupsig.Pubkey
//	gc := p.joiningGroups.GetGroup(gid)
//	node := gc.GetNode()
//	if node != nil {
//		pub_piece = node.GetSeedPubKey()
//	}
//	return pub_piece
//}
//
////取得自己参与的某个铸块组的私钥片段（聚合一个组所有成员的私钥片段，可以生成组私钥）
////用于测试目的，正式版对外不提供。
//func (p Processor) getMinerSecKeyPieceForGroup(gid groupsig.ID) groupsig.Seckey {
//	var secPiece groupsig.Seckey
//	gc := p.joiningGroups.GetGroup(gid)
//	node := gc.GetNode()
//	if node != nil {
//		secPiece = node.getSeedSecKey()
//	}
//	return secPiece
//}

//取得特定的组
func (p Processor) GetGroup(gid groupsig.ID) *model.GroupInfo {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g
	}
}

//取得一个铸块组的公钥(processer初始化时从链上加载)
func (p Processor) getGroupPubKey(gid groupsig.ID) groupsig.Pubkey {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g.GetGroupPubKey()
	}

}

//func outputBlockHeaderAndSign(prefix string, bh *types.BlockHeader, si *model.SignData) {
//	//bbyte, _ := bh.CurTime.MarshalBinary()
//	//jbyte, _ := bh.CurTime.MarshalJSON()
//	//textbyte, _ := bh.CurTime.MarshalText()
//	//log.Printf("%v, bh.curTime %v, byte=%v, jsonByte=%v, textByte=%v, nano=%v, utc=%v, local=%v, location=%v\n", prefix, bh.CurTime, bbyte, jbyte, textbyte, bh.CurTime.UnixNano(), bh.CurTime.UTC(), bh.CurTime.Local(), bh.CurTime.Location().String())
//
//	//var castor groupsig.ID
//	//castor.Deserialize(bh.Castor)
//	//txs := ""
//	//if bh.Transactions != nil {
//	//	for _, tx := range bh.Transactions {
//	//		txs += GetHashPrefix(tx) + ","
//	//	}
//	//}
//	//txs = "[" + txs + "]"
//	//log.Printf("%v, BLOCKINFO: height= %v, castor=%v, hash=%v, txs=%v, txtree=%v, statetree=%v, receipttree=%v\n", prefix, bh.Height, GetIDPrefix(castor), GetHashPrefix(bh.Hash), txs, GetHashPrefix(bh.TxTree), GetHashPrefix(bh.StateTree), GetHashPrefix(bh.ReceiptTree))
//	//
//	//if si != nil {
//	//	log.Printf("%v, SIDATA: datahash=%v, sign=%v, signer=%v\n", prefix, GetHashPrefix(si.DataHash), si.DataSign.GetHexString(), GetIDPrefix(si.SignMember))
//	//}
//}

func (p *Processor) ExistInGroup(gHash common.Hash) bool {
	//initingGroup := p.globalGroups.GetInitedGroup(gHash)
	//if initingGroup == nil {
	//	return false
	//}
	//return initingGroup.MemberExist(p.GetMinerID())
	return false
}

// getSignKey get the signature private key of the miner in a certain group
func (p Processor) getSignKey(gid groupsig.ID) groupsig.Seckey {
	if jg := p.belongGroups.GetJoinedGroupInfo(gid); jg != nil {
		return jg.SignSecKey
	}
	return groupsig.Seckey{}
}

//func (p *Processor) getInGroupSeckeyInfo(gid groupsig.ID) model.SecKeyInfo {
//	return model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(gid))
//}
