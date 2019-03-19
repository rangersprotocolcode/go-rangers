package core

import (
	"sync"
	"encoding/json"
	"fmt"
	"encoding/binary"
	"x/src/middleware/db"
	"x/src/middleware/types"
	"x/src/middleware/notify"
	"x/src/utility"
	"bytes"
	"errors"
	"x/src/common"
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
}

func initGroupChain() {
	chain := &groupChain{}

	var err error
	chain.groups, err = db.NewDatabase(groupChainPrefix)
	if err != nil {
		panic("Init group chain error:" + err.Error())
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
		lastGroup = &consensusHelper.GenerateGenesisInfo().Group
		e := chain.save(lastGroup)
		if e != nil {
			panic("Add genesis group on chain failed:" + e.Error())
		}
	}
	chain.lastGroup = lastGroup
	groupChainImpl = chain
}

func (chain *groupChain) AddGroup(group *types.Group) error {
	if nil == group {
		return fmt.Errorf("nil group")
	}

	if logger != nil {
		logger.Debugf("Group chain add group %+v", group)
	}
	if exist, _ := chain.groups.Has(group.Id); exist {
		notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: *group,})
		return errors.New("group already exist")
	}

	ok, err := consensusHelper.CheckGroup(group)
	if !ok {
		if err == common.ErrCreateBlockNil {
			logger.Infof("Add group failed:depend on block!")
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

func (chain *groupChain) GetGroupByHeight(height uint64) (*types.Group) {
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

	notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: *group,})
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
