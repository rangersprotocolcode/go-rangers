package core

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"x/src/common"
	"x/src/middleware/db"
	"x/src/middleware/notify"
	"x/src/middleware/types"
	"x/src/utility"
)

const (
	lastGroupKey     = "gcurrent"
	groupCountKey    = "gcount"
	groupChainPrefix = "group"
)

var groupChainImpl GroupChain

type groupChain struct {
	count uint64

	lock sync.RWMutex

	lastGroup *types.Group

	groups db.Database // key:id, value:group && key:number, value:id

	joinedGroups *db.LDBDatabase
}

func initGroupChain() {
	chain := &groupChain{}

	var err error
	chain.groups, err = db.NewDatabase(groupChainPrefix)
	if err != nil {
		panic("Init group chain error:" + err.Error())
	}

	chain.joinedGroups, err = db.NewLDBDatabase(common.GlobalConf.GetString(db.ConfigSec, db.DefaultJoinedGroupDatabaseKey, "jgs"), 1, 1)
	if err != nil {
		panic("newLDBDatabase fail, file=" + "" + "err=" + err.Error())
	}

	lastGroupId, _ := chain.groups.Get([]byte(lastGroupKey))
	count, _ := chain.groups.Get([]byte(groupCountKey))
	var lastGroup *types.Group
	if lastGroupId != nil {
		data, _ := chain.groups.Get(lastGroupId)
		err := json.Unmarshal(data, &lastGroup)
		if err != nil {
			panic("Unmarshal last group failed:" + err.Error())
		}
		chain.count = utility.ByteToUInt64(count)
	} else {
		genesisGroups := consensusHelper.GenerateGenesisInfo()
		for _, genesis := range genesisGroups {
			e := chain.save(&genesis.Group)
			if e != nil {
				panic("Add genesis group on chain failed:" + e.Error())
			}
		}
		lastGroup = &genesisGroups[len(genesisGroups)-1].Group
	}
	chain.lastGroup = lastGroup
	groupChainImpl = chain
}

func (chain *groupChain) AddGroup(group *types.Group) error {
	if nil == group {
		return fmt.Errorf("nil group")
	}

	if logger != nil {
		logger.Debugf("Group chain add group %+v", common.Bytes2Hex(group.Id))
	}
	if exist, _ := chain.groups.Has(group.Id); exist {
		notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: *group})
		return common.ErrGroupAlreadyExist
	}

	ok, err := consensusHelper.CheckGroup(group)
	if !ok {
		if err == common.ErrCreateBlockNil {
			logger.Infof("Add group failed:depend on block!")
		} else {
			logger.Infof("Add group failed:%v", err.Error())
		}
		return err
	}

	chain.lock.Lock()
	defer chain.lock.Unlock()
	exist, _ := chain.groups.Has(group.Header.Parent)
	if !exist {
		return fmt.Errorf("parent is not existed on group chain!Parent id:%v", group.Header.Parent)
	}

	if !bytes.Equal(chain.lastGroup.Id, group.Header.PreGroup) {
		return fmt.Errorf("pre not equal lastgroup!Pre group id:%v,local last group id:%v", group.Header.PreGroup, chain.lastGroup.Id)
	}

	return chain.save(group)
}

func (chain *groupChain) GetGroupById(id []byte) *types.Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	return chain.getGroupById(id)
}

func (chain *groupChain) GetGroupByHeight(height uint64) *types.Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	return chain.getGroupByHeight(height)
}

func (chain *groupChain) LastGroup() *types.Group {
	return chain.lastGroup
}

func (chain *groupChain) Count() uint64 {
	return chain.count
}

func (chain *groupChain) Close() {
	chain.groups.Close()
}

func (chain *groupChain) Iterator() *GroupIterator {
	return &GroupIterator{current: chain.lastGroup}
}

func (chain *groupChain) availableGroupsAt(h uint64) []*types.Group {
	iter := chain.Iterator()
	gs := make([]*types.Group, 0)
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		if g.Header.DismissHeight > h {
			gs = append(gs, g)
		} else {
			genesis := chain.GetGroupByHeight(0)
			gs = append(gs, genesis)
			break
		}
	}
	return gs
}

func (chain *groupChain) GetAvailableGroupsByMinerId(height uint64, minerId []byte) []*types.Group {
	allGroups := chain.availableGroupsAt(height)
	group := make([]*types.Group, 0)

	for _, g := range allGroups {
		for _, mem := range g.Members {
			if bytes.Equal(mem, minerId) {
				group = append(group, g)
				break
			}
		}
	}

	return group
}

func (chain *groupChain) getGroupByHeight(height uint64) *types.Group {
	groupId, _ := chain.groups.Get(generateKey(height))
	if nil != groupId {
		return chain.getGroupById(groupId)
	}
	return nil
}

func (chain *groupChain) getGroupById(id []byte) *types.Group {
	data, _ := chain.groups.Get(id)
	if nil == data || 0 == len(data) {
		return nil
	}

	var group *types.Group
	err := json.Unmarshal(data, &group)
	if err != nil {
		logger.Errorf("Unmarshal group error:%s", err.Error())
		return nil
	}
	return group
}

func (chain *groupChain) save(group *types.Group) error {
	group.GroupHeight = chain.count
	data, err := json.Marshal(group)
	if err != nil {
		logger.Errorf("Marshal group error:%s", err.Error())
		return err
	}

	chain.groups.Put(group.Id, data)
	chain.groups.Put([]byte(lastGroupKey), group.Id)
	chain.groups.Put(generateKey(chain.count), group.Id)
	chain.count++
	chain.groups.Put([]byte(groupCountKey), utility.UInt64ToByte(chain.count))
	chain.lastGroup = group
	logger.Debugf("Add group on chain success! Group id:%s,group pubkey:%s", hex.EncodeToString(group.Id), hex.EncodeToString(group.PubKey))

	if nil != notify.BUS {
		notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: *group})
	}
	if GroupSyncer != nil {
		GroupSyncer.sendGroupHeightToNeighbor(chain.count)
	}
	return nil
}

func generateKey(i uint64) []byte {
	return intToBytes(i)
}

func intToBytes(n uint64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(n))
	return buf
}

func (iterator *GroupIterator) Current() *types.Group {
	return iterator.current
}

func (iterator *GroupIterator) MovePre() *types.Group {
	iterator.current = groupChainImpl.GetGroupById(iterator.current.Header.PreGroup)
	return iterator.current
}

func (chain *groupChain) GetSyncGroupsById(id []byte) []*types.Group {
	result := make([]*types.Group, 0)
	group := chain.getGroupById(id)
	if group == nil {
		return result
	}
	return chain.GetSyncGroupsByHeight(group.GroupHeight+1, 5)
}

func (chain *groupChain) GetSyncGroupsByHeight(height uint64, limit int) []*types.Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	return chain.getSyncGroupsByHeight(height, limit)
}

func (chain *groupChain) getSyncGroupsByHeight(height uint64, limit int) []*types.Group {
	result := make([]*types.Group, 0)
	for i := 0; i < limit; i++ {
		groupId, _ := chain.groups.Get(generateKey(height + uint64(i)))
		if nil != groupId {
			result = append(result, chain.getGroupById(groupId))
		} else {
			break
		}
	}

	return result
}

func (chain *groupChain) SaveJoinedGroup(id []byte, value []byte) bool {
	err := chain.joinedGroups.Put(id, value)
	return err == nil
}

func (chain *groupChain) GetJoinedGroup(id []byte) ([]byte, error) {
	return chain.joinedGroups.Get(id)
}

func (chain *groupChain) DeleteJoinedGroup(id []byte) bool {
	err := chain.joinedGroups.Delete(id)
	return err == nil
}
