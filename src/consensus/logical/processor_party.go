// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package logical

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"runtime/debug"
	"sync"
	"time"
)

func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) {
	key := p.generatePartyKey(ccm.BH)
	party := p.loadOrNewSignParty(key, ccm, true)

	if nil == party {
		return
	}
	party.Update(ccm)
}

func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	party := p.loadOrNewSignParty(cvm.BlockHash.Bytes(), cvm, false)
	if nil == party {
		return
	}
	party.Update(cvm)
}

func (p *Processor) loadOrNewSignParty(keyBytes []byte, msg model.ConsensusMessage, isNew bool) Party {
	p.partyLock.Lock("loadOrNewSignParty")
	defer p.partyLock.Unlock("loadOrNewSignParty")

	key := common.ToHex(keyBytes)
	item, ok := p.partyManager[key]
	if ok {
		p.logger.Debugf("get party: %s", key)
		return item
	}

	if p.finishedParty.Contains(key) {
		p.logger.Warnf("party: %s already done", key)
		return nil
	}

	if !isNew {
		var msgs []model.ConsensusMessage
		msgsRaw, ok := p.futureMessages.Get(key)
		if !ok {
			msgs = make([]model.ConsensusMessage, 0)
		} else {
			msgs = msgsRaw.([]model.ConsensusMessage)
		}
		msgs = append(msgs, msg)
		p.futureMessages.Add(key, msgs)

		p.logger.Infof("save future message for: %s, after length: %d", key, len(msgs))
		return nil
	}

	party := &SignParty{belongGroups: p.belongGroups, blockchain: p.MainChain,
		minerReader: p.minerReader, globalGroups: p.globalGroups,
		mi: p.mi.ID, netServer: p.NetServer,
		baseParty: baseParty{
			logger:         p.logger,
			mtx:            sync.Mutex{},
			futureMessages: make(map[string]model.ConsensusMessage),
			Done:           make(chan byte, 1),
			Err:            make(chan error, 1),
			id:             key,
		},
	}
	if err := party.Start(); err == nil {
		p.partyManager[key] = party
		p.logger.Debugf("new party: %s", key)

		// wait until finish
		go p.waitUntilDone(party)

		return party
	} else {
		p.logger.Errorf("fail to start party, %s", err)
		return nil
	}

}

func (p *Processor) waitUntilDone(party *SignParty) {
	defer func() {
		party.Close()
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("recover error：%s\n%s", r, string(debug.Stack()))
		}
	}()

	key := party.id
	for {
		select {
		// timeout
		case <-time.After(10 * time.Second):
			func() {
				p.partyLock.Lock("timeout")
				defer p.partyLock.Unlock("timeout")

				delete(p.partyManager, party.id)
				p.logger.Errorf("timeout, id: %s, original: %s", party.id, key)
			}()
			return
		case err := <-party.Err:
			func() {
				p.partyLock.Lock("err")
				defer p.partyLock.Unlock("err")

				delete(p.partyManager, party.id)
				p.finishedParty.Add(party.id, 0)
				p.logger.Errorf("error: %s, id: %s", err, party.id)
			}()

			return
		case <-party.Done:
			func() {
				p.partyLock.Lock("done")
				defer p.partyLock.Unlock("done")

				delete(p.partyManager, party.id)
				p.finishedParty.Add(party.id, 0)
				p.logger.Infof("done, party: %s", party.id)
			}()

			return
		case realKey := <-party.ChangedId:
			func() {
				p.partyLock.Lock("changeId")
				defer p.partyLock.Unlock("changeId")

				p.logger.Infof("start to changeId, from %s to %s", key, realKey)

				p.finishedParty.Add(key, 0)
				delete(p.partyManager, key)

				party.SetId(realKey)
				p.partyManager[realKey] = party

				if msgsRaw, ok := p.futureMessages.Get(realKey); ok {
					msgs := msgsRaw.([]model.ConsensusMessage)
					p.futureMessages.Remove(realKey)
					p.logger.Infof("changeId, get future messages, from %s to %s. len: %d", key, realKey, len(msgs))
					for _, msg := range msgs {
						if nil == msg {
							p.logger.Infof("changeId and update future messages, from %s to %s, nil msg", key, realKey)
							continue
						}

						go func(m model.ConsensusMessage) {
							p.logger.Infof("changeId and update future messages, from %s to %s, msg: %s", key, realKey, m.GetMessageID())
							party.Update(m)
						}(msg)
					}
				}

				p.logger.Infof("fin changeId, from %s to %s", key, realKey)
			}()
		}
	}
}

func (p *Processor) generatePartyKey(bh types.BlockHeader) []byte {
	result := make([]byte, 0)
	result = append(result, utility.UInt64ToByte(bh.Height)...)
	result = append(result, bh.PreHash.Bytes()...)
	result = append(result, bh.Castor...)
	result = append(result, bh.ProveValue.Bytes()...)
	result = append(result, bh.TxTree.Bytes()...)
	result = append(result, bh.GroupId...)
	return common.Sha256(result)
}
