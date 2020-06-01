package core

import (
	"bytes"
	"math"
	"time"

	"x/src/common"
	"x/src/middleware"
	"x/src/middleware/log"
	"x/src/middleware/notify"
	"x/src/middleware/pb"
	"x/src/middleware/types"
	"x/src/network"
	"x/src/utility"

	"github.com/gogo/protobuf/proto"
)

const (
	groupSyncInterval          = 3 * time.Second
	sendGroupInfoInterval      = 3 * time.Second
	groupSyncCandidatePoolSize = 3
	groupSyncReqTimeout        = 3 * time.Second
)

var GroupSyncer *groupSyncer

type GroupInfo struct {
	Groups     []*types.Group
	IsTopGroup bool
}

type groupSyncer struct {
	syncing       bool
	candidate     string
	candidatePool map[string]uint64

	init                 bool
	reqTimeoutTimer      *time.Timer
	syncTimer            *time.Timer
	groupInfoNotifyTimer *time.Timer

	lock   middleware.Loglock
	logger log.Logger
}

func InitGroupSyncer() {
	GroupSyncer = &groupSyncer{syncing: false, candidate: "", candidatePool: make(map[string]uint64), lock: middleware.NewLoglock(""), init: false}
	GroupSyncer.logger = log.GetLoggerByIndex(log.GroupSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	GroupSyncer.reqTimeoutTimer = time.NewTimer(groupSyncReqTimeout)
	GroupSyncer.syncTimer = time.NewTimer(groupSyncInterval)
	GroupSyncer.groupInfoNotifyTimer = time.NewTimer(sendGroupInfoInterval)

	notify.BUS.Subscribe(notify.GroupHeight, GroupSyncer.groupHeightHandler)
	notify.BUS.Subscribe(notify.GroupReq, GroupSyncer.groupReqHandler)
	notify.BUS.Subscribe(notify.Group, GroupSyncer.groupHandler)

	go GroupSyncer.loop()
}

func (gs *groupSyncer) IsInit() bool {
	return gs.init
}

func (gs *groupSyncer) trySync() {
	gs.lock.Lock("trySyncGroup")
	defer gs.lock.Unlock("trySyncGroup")

	gs.syncTimer.Reset(groupSyncInterval)
	if gs.syncing {
		gs.logger.Debugf("Syncing to %s,do not sync anymore!", gs.candidate)
		return
	}

	id, candidateHeight := gs.getCandidateForSync()
	if id == "" {
		gs.logger.Debugf("Get no candidate for sync!")
		if !gs.init {
			gs.init = true
		}
		return
	}
	gs.logger.Debugf("Get candidate %s for sync!Candidate group height:%d", id, candidateHeight)
	gs.syncing = true
	gs.candidate = id
	gs.reqTimeoutTimer.Reset(blockSyncReqTimeout)

	go gs.requestGroupByGroupId(id, groupChainImpl.LastGroup().Id)
}

func (gs *groupSyncer) groupHeightHandler(msg notify.Message) {
	groupHeightMsg, ok := msg.GetData().(*notify.GroupHeightMessage)
	if !ok {
		gs.logger.Errorf("groupHeightHandler GetData assert not ok!")
		return
	}

	source := groupHeightMsg.Peer
	height := utility.ByteToUInt64(groupHeightMsg.HeightByte)

	localGroupHeight := groupChainImpl.Count()
	if !gs.isUsefulCandidate(localGroupHeight, height) {
		return
	}
	gs.addCandidatePool(source, height)
}

func (gs *groupSyncer) groupReqHandler(msg notify.Message) {
	groupReqMsg, ok := msg.GetData().(*notify.GroupReqMessage)
	if !ok {
		gs.logger.Errorf("groupReqHandler GetData assert not ok!")
		return
	}

	sourceId := groupReqMsg.Peer
	groupId := groupReqMsg.GroupIdByte
	gs.logger.Debugf("Rcv group req from:%s,id:%v\n", sourceId, groupId)
	groups := groupChainImpl.GetSyncGroupsById(groupId)

	l := len(groups)
	if l == 0 {
		gs.logger.Errorf("Get nil group by id:%v", groupId)
		//gs.sendGroups(sourceId, []*types.Group{}, true)
		return
	} else {
		var isTop bool
		if bytes.Equal(groups[l-1].Id, groupChainImpl.LastGroup().Id) {
			isTop = true
		}
		gs.sendGroups(sourceId, groups, isTop)
		gs.logger.Debugf("SendGroups:%s,lastId:%d\n", sourceId, groups[l-1].Id)
	}
}

func (gs *groupSyncer) groupHandler(msg notify.Message) {
	groupInfoMsg, ok := msg.GetData().(*notify.GroupInfoMessage)
	if !ok {
		gs.logger.Errorf("groupHandler GetData assert not ok!")
		return
	}

	groupInfo, e := gs.unMarshalGroupInfo(groupInfoMsg.GroupInfoByte)
	if e != nil {
		gs.logger.Errorf("Discard GROUP_MSG because of unmarshal error:%s", e.Error())
		return
	}

	sourceId := groupInfoMsg.Peer
	groups := groupInfo.Groups
	addGroupResult := true
	gs.logger.Debugf("Rcv groups ,from:%s,groups len %d", sourceId, len(groups))
	for _, group := range groupInfo.Groups {
		gs.logger.Debugf("AddGroup Id:%s,pre id:%s", common.BytesToAddress(group.Id).GetHexString(), common.BytesToAddress(group.Header.Parent).GetHexString())
		gs.logger.Debugf("Local height:%d,local top group id:%s", groupChainImpl.Count(), common.BytesToAddress(groupChainImpl.LastGroup().Id).GetHexString())
		e := groupChainImpl.AddGroup(group)
		if e != nil {
			gs.logger.Errorf("[GroupSyncer]add group on chain error:%s", e.Error())
			if e != common.ErrGroupAlreadyExist {
				addGroupResult = false
				break
			}
		}
	}
	if len(groups) == 0 {
		addGroupResult = false
	}

	if !addGroupResult {
		return
	}
	gs.logger.Debugf("Group sync finished! Set syncing false.Set candidate nil!")
	gs.lock.Lock("groupHandler")
	gs.candidate = ""
	gs.syncing = false
	gs.reqTimeoutTimer.Stop()
	gs.lock.Unlock("groupHandler")

	go gs.trySync()
}

func (gs *groupSyncer) getCandidateForSync() (string, uint64) {
	localGroupHeight := groupChainImpl.Count()
	gs.logger.Debugf("Local group height:%d", localGroupHeight)
	gs.candidatePoolDump()

	uselessCandidate := make([]string, 0, blockSyncCandidatePoolSize)
	for id, height := range gs.candidatePool {
		if !gs.isUsefulCandidate(localGroupHeight, height) {
			uselessCandidate = append(uselessCandidate, id)
			continue
		}
		if PeerManager.isEvil(id) {
			uselessCandidate = append(uselessCandidate, id)
		}
	}
	if len(uselessCandidate) != 0 {
		for _, id := range uselessCandidate {
			delete(gs.candidatePool, id)
		}
	}

	candidateId := ""
	var candidateMaxHeight uint64 = 0
	for id, height := range gs.candidatePool {
		if height > candidateMaxHeight {
			candidateId = id
			candidateMaxHeight = height
		}
	}

	return candidateId, candidateMaxHeight
}

func (gs *groupSyncer) addCandidatePool(id string, groupHeight uint64) {
	if PeerManager.isEvil(id) {
		gs.logger.Debugf("Group notify id:%s is marked evil.Drop it!", id)
		return
	}

	gs.lock.Lock("addCandidatePool")
	defer gs.lock.Unlock("addCandidatePool")
	if len(gs.candidatePool) < groupSyncCandidatePoolSize {
		gs.candidatePool[id] = groupHeight
		return
	}

	heightMinId := ""
	var minHeight uint64 = math.MaxUint64
	for id, height := range gs.candidatePool {
		if height <= minHeight {
			heightMinId = id
			minHeight = height
		}
	}
	if groupHeight > minHeight {
		delete(gs.candidatePool, heightMinId)
		gs.candidatePool[id] = groupHeight
		if !gs.syncing {
			go gs.trySync()
		}
	}
}

func (gs *groupSyncer) candidatePoolDump() {
	gs.logger.Debugf("Candidate Pool Dump:")
	for id, groupHeight := range gs.candidatePool {
		gs.logger.Debugf("Candidate id:%s,group height:%d", id, groupHeight)
	}
}

func (gs *groupSyncer) isUsefulCandidate(localGroupHeight uint64, candidateGroupHeight uint64) bool {
	if candidateGroupHeight <= localGroupHeight {
		return false
	}
	return true
}

func (gs *groupSyncer) loop() {
	for {
		select {
		case <-gs.groupInfoNotifyTimer.C:
			go gs.sendGroupHeightToNeighbor(groupChainImpl.Count())
		case <-gs.syncTimer.C:
			gs.logger.Debugf("Group sync time up! Try sync")
			go gs.trySync()
		case <-gs.reqTimeoutTimer.C:
			gs.logger.Debugf("Group sync to %s time out!", gs.candidate)
			PeerManager.markEvil(gs.candidate)
			gs.lock.Lock("req time out")
			gs.syncing = false
			gs.candidate = ""
			gs.lock.Unlock("req time out")
		}
	}
}

func (gs *groupSyncer) sendGroupHeightToNeighbor(localCount uint64) {
	gs.lock.Lock("sendGroupHeightToNeighbor")
	gs.groupInfoNotifyTimer.Reset(sendGroupInfoInterval)
	gs.lock.Unlock("sendGroupHeightToNeighbor")

	gs.logger.Debugf("Send local group height %d to neighbor!", localCount)
	body := utility.UInt64ToByte(localCount)
	message := network.Message{Code: network.GroupChainCountMsg, Body: body}
	network.GetNetInstance().Broadcast(message)
}

func (gs *groupSyncer) requestGroupByGroupId(id string, groupId []byte) {
	gs.logger.Debugf("Req group for %s,id:%v!", id, groupId)
	message := network.Message{Code: network.ReqGroupMsg, Body: groupId}
	network.GetNetInstance().Send(id, message)
}

func (gs *groupSyncer) sendGroups(targetId string, groups []*types.Group, isTop bool) {
	if len(groups) == 0 {
		logger.Debugf("Send nil group to:%s", targetId)
	} else {
		gs.logger.Debugf("Send group to %s,groups:%d-%d,isTop:%t", targetId, groups[0].GroupHeight, groups[len(groups)-1].GroupHeight, isTop)
	}
	body, e := marshalGroupInfo(groups, isTop)
	if e != nil {
		gs.logger.Errorf("sendGroup marshal group error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.GroupMsg, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func marshalGroupInfo(e []*types.Group, isTop bool) ([]byte, error) {
	var groups []*middleware_pb.Group
	for _, g := range e {
		groups = append(groups, types.GroupToPb(g))
	}

	groupInfo := middleware_pb.GroupInfo{Groups: groups, IsTopGroup: &isTop}
	return proto.Marshal(&groupInfo)
}

func (gs *groupSyncer) unMarshalGroupInfo(b []byte) (*GroupInfo, error) {
	message := new(middleware_pb.GroupInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		gs.logger.Errorf("unMarshalGroupInfo error:%s", e.Error())
		return nil, e
	}
	groups := make([]*types.Group, len(message.Groups))
	for i, g := range message.Groups {
		groups[i] = types.PbToGroup(g)
	}
	groupInfo := GroupInfo{Groups: groups, IsTopGroup: *message.IsTopGroup}
	return &groupInfo, nil
}
