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

package logical

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/utility"
	"sync"
	"time"
)


type verifyMsgCache struct {
	castMsg *model.ConsensusCastMessage
	verifyMsgs []*model.ConsensusVerifyMessage
	expire time.Time
	lock sync.RWMutex
}

func newVerifyMsgCache() *verifyMsgCache {
	return &verifyMsgCache{
		verifyMsgs: make([]*model.ConsensusVerifyMessage, 0),
		expire: utility.GetTime().Add(30*time.Second),
	}
}

func (c *verifyMsgCache) expired() bool {
    return utility.GetTime().After(c.expire)
}

func (c *verifyMsgCache) addVerifyMsg(msg *model.ConsensusVerifyMessage)  {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.verifyMsgs = append(c.verifyMsgs, msg)
}

func (c *verifyMsgCache) merge(msg *verifyMsgCache)  {
    c.lock.Lock()
    defer c.lock.Unlock()
	if msg.castMsg != nil && c.castMsg == nil {
		c.castMsg = msg.castMsg
	}
	if msg.verifyMsgs != nil {
		for _, m := range msg.verifyMsgs {
			c.verifyMsgs = append(c.verifyMsgs, m)
		}
	}
}

func (c *verifyMsgCache) getVerifyMsgs() []*model.ConsensusVerifyMessage {
    msgs := make([]*model.ConsensusVerifyMessage, len(c.verifyMsgs))
    c.lock.RLock()
    defer c.lock.RUnlock()
    copy(msgs, c.verifyMsgs)
    return msgs
}

func (c *verifyMsgCache) removeVerifyMsgs()  {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.verifyMsgs = make([]*model.ConsensusVerifyMessage, 0)
}

func (p *Processor) addVerifyCache(hash common.Hash, cache *verifyMsgCache)  {
	if ok, _ := p.verifyMsgCaches.ContainsOrAdd(hash, cache); ok {
		c := p.getVerifyMsgCache(hash)
		if c == nil {
			return
		}
		c.merge(cache)
	}
}

func (p *Processor) addVerifyMsgToCache(msg *model.ConsensusVerifyMessage)  {
	cache := p.getVerifyMsgCache(msg.BlockHash)
	if cache == nil {
		cache := newVerifyMsgCache()
		cache.addVerifyMsg(msg)
		p.addVerifyCache(msg.BlockHash, cache)
	} else {
		cache.addVerifyMsg(msg)
	}
}

func (p *Processor) addCastMsgToCache(msg *model.ConsensusCastMessage)  {
    cache := p.getVerifyMsgCache(msg.BH.Hash)
	if cache == nil {
		cache := newVerifyMsgCache()
		cache.castMsg = msg
		p.addVerifyCache(msg.BH.Hash, cache)
	} else {
		cache.castMsg = msg
	}
}

func (p *Processor) getVerifyMsgCache(hash common.Hash) *verifyMsgCache {
	v, ok := p.verifyMsgCaches.Get(hash)
	if !ok {
		return nil
	}
	return v.(*verifyMsgCache)
}

func (p *Processor) removeVerifyMsgCache(hash common.Hash)  {
	p.verifyMsgCaches.Remove(hash)
}
