// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package access

import (
	"com.tuntun.rocket/node/src/core"
	"sync"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"encoding/json"
	"github.com/hashicorp/golang-lru"
)

// key suffix definition when store the group infos to db
const (
	suffixSignKey = "_signKey"
	suffixGInfo   = "_gInfo"
)

//BelongGroups
// BelongGroups stores all group-related infos which is important to the members
type JoinedGroupStorage struct {
	groupChain core.GroupChain
	cache      *lru.Cache
	initMutex  sync.Mutex
}

//NewBelongGroups
func NewJoinedGroupStorage() *JoinedGroupStorage {
	return &JoinedGroupStorage{
		groupChain: core.GetGroupChain(),
	}
}

func (storage *JoinedGroupStorage) addJoinedGroupInfo(joinedGroupInfo *model.JoinedGroupInfo) {
	if !storage.ready() {
		storage.initStore()
	}
	logger.Debugf("Add joined group info: group id=%v", joinedGroupInfo.GroupID.ShortS())
	storage.cache.Add(joinedGroupInfo.GroupID.GetHexString(), joinedGroupInfo)
	storage.storeJoinedGroup(joinedGroupInfo)
}

func (storage *JoinedGroupStorage) GetJoinedGroupInfo(id groupsig.ID) *model.JoinedGroupInfo {
	if !storage.ready() {
		storage.initStore()
	}
	v, ok := storage.cache.Get(id.GetHexString())
	if ok {
		return v.(*model.JoinedGroupInfo)
	}
	jg := storage.load(id)
	if jg != nil {
		storage.cache.Add(jg.GroupID.GetHexString(), jg)
	}
	return jg
}

func (storage *JoinedGroupStorage) AddMemberSignPk(minerId groupsig.ID, groupId groupsig.ID, signPK groupsig.Pubkey) (*model.JoinedGroupInfo, bool) {
	if !storage.ready() {
		storage.initStore()
	}
	jg := storage.GetJoinedGroupInfo(groupId)
	if jg == nil {
		return nil, false
	}

	if _, ok := jg.GetMemberSignPK(minerId); !ok {
		jg.AddMemberSignPK(minerId, signPK)
		storage.saveGroupInfo(jg)
		return jg, true
	}
	return jg, false
}

func (storage *JoinedGroupStorage) LeaveGroups(gids []groupsig.ID) {
	if !storage.ready() {
		return
	}
	for _, gid := range gids {
		storage.cache.Remove(gid.GetHexString())
		storage.groupChain.DeleteJoinedGroup(gInfoSuffix(gid))
	}
}

func (storage *JoinedGroupStorage) Close() {
	if !storage.ready() {
		return
	}
	storage.cache = nil
	//storage.db.Close()
}

//IsMinerGroup
// IsMinerGroup detecting whether a group is a miner's ingot group
// (a miner can participate in multiple groups)
func (storage *JoinedGroupStorage) BelongGroup(groupId groupsig.ID) bool {
	return storage.GetJoinedGroupInfo(groupId) != nil
}

// joinGroup join a group (a miner ID can join multiple groups)
//			gid : group ID (not dummy id)
//			sk: user's group member signature private key
func (storage *JoinedGroupStorage) JoinGroup(joinedGroupInfo *model.JoinedGroupInfo, selfMinerId groupsig.ID) {
	logger.Infof("(%v):join group,group id=%v,secKey:%v\n", selfMinerId.GetHexString(), joinedGroupInfo.GroupID.ShortS(), joinedGroupInfo.SignSecKey.GetHexString())
	if !storage.BelongGroup(joinedGroupInfo.GroupID) {
		storage.addJoinedGroupInfo(joinedGroupInfo)
	}
	return
}

func (storage *JoinedGroupStorage) initStore() {
	storage.initMutex.Lock()
	defer storage.initMutex.Unlock()

	if storage.ready() {
		return
	}
	storage.cache = common.CreateLRUCache(30)
}

func (storage *JoinedGroupStorage) ready() bool {
	return storage.cache != nil
}

func (storage *JoinedGroupStorage) storeJoinedGroup(joinedGroupInfo *model.JoinedGroupInfo) {
	storage.saveSignSecKey(joinedGroupInfo)
	storage.saveGroupInfo(joinedGroupInfo)
}

func (storage *JoinedGroupStorage) saveSignSecKey(joinedGroupInfo *model.JoinedGroupInfo) {
	if !storage.ready() {
		return
	}
	storage.groupChain.SaveJoinedGroup(signKeySuffix(joinedGroupInfo.GroupID), joinedGroupInfo.SignSecKey.Serialize())
}

func (storage *JoinedGroupStorage) saveGroupInfo(joinedGroupInfo *model.JoinedGroupInfo) {
	if !storage.ready() {
		return
	}
	st := joinedGroupDO{
		GroupID:             joinedGroupInfo.GroupID,
		GroupPK:             joinedGroupInfo.GroupPK,
		MemberSignPubkeyMap: joinedGroupInfo.GetMemberPKs(),
	}
	bs, err := json.Marshal(st)
	if err != nil {
		logger.Errorf("marshal joinedGroupDO fail, err=%v", err)
	} else {
		storage.groupChain.SaveJoinedGroup(gInfoSuffix(joinedGroupInfo.GroupID), bs)
	}
}

func (storage *JoinedGroupStorage) load(gid groupsig.ID) *model.JoinedGroupInfo {
	if !storage.ready() {
		return nil
	}
	joinedGroupInfo := new(model.JoinedGroupInfo)
	joinedGroupInfo.MemberSignPubkeyMap = make(map[string]groupsig.Pubkey, 0)
	// Load signature private key
	bs, err := storage.groupChain.GetJoinedGroup(signKeySuffix(gid))
	if err != nil {
		logger.Infof("get signKey fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return nil
	}
	//logger.Debugf("load bs:%v,privateKey:%v",bs,storage.privateKey.GetHexString())
	joinedGroupInfo.SignSecKey.Deserialize(bs)

	// Load group information
	infoBytes, err := storage.groupChain.GetJoinedGroup(gInfoSuffix(gid))
	if err != nil {
		logger.Errorf("get groupInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return joinedGroupInfo
	}
	if err := json.Unmarshal(infoBytes, joinedGroupInfo); err != nil {
		logger.Errorf("unmarshal groupInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return joinedGroupInfo
	}
	return joinedGroupInfo
}

type joinedGroupDO struct {
	GroupID             groupsig.ID                // Group ID
	GroupPK             groupsig.Pubkey            // Group public key (backup, which can be taken from the global group)
	MemberSignPubkeyMap map[string]groupsig.Pubkey // Group member signature public key
}

func signKeySuffix(gid groupsig.ID) []byte {
	return []byte(gid.GetHexString() + suffixSignKey)
}

func gInfoSuffix(gid groupsig.ID) []byte {
	return []byte(gid.GetHexString() + suffixGInfo)
}
