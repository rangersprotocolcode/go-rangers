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
	p.partyLock.Lock()
	defer p.partyLock.Unlock()

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
		msgs, ok := p.futureMessages[key]
		if !ok {
			msgs = make([]model.ConsensusMessage, 1)
			p.futureMessages[key] = msgs
		}
		msgs = append(msgs, msg)

		msgs = p.futureMessages[key]
		p.logger.Infof("save futuremessage for: %s, after length: %d", key, len(msgs))
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
	if nil == party.Start() {
		p.partyManager[key] = party
		p.logger.Debugf("new party: %s", key)

		// wait until finish
		go p.waitUntilDone(party)

		return party
	}

	return nil
}

func (p *Processor) waitUntilDone(party *SignParty) {
	defer party.Close()

	key := party.id
	for {
		select {
		// timeout
		case <-time.After(10 * time.Second):
			func() {
				p.partyLock.Lock()
				defer p.partyLock.Unlock()

				delete(p.partyManager, party.id)
				p.logger.Errorf("timeout, id: %s, original: %s", party.id, key)
			}()
			return
		case err := <-party.Err:
			func() {
				p.partyLock.Lock()
				defer p.partyLock.Unlock()

				delete(p.partyManager, party.id)
				p.finishedParty.Add(party.id, 0)
				p.logger.Errorf("error: %s, id: %s", err, party.id)
			}()

			return
		case <-party.Done:
			func() {
				p.partyLock.Lock()
				defer p.partyLock.Unlock()

				delete(p.partyManager, party.id)
				p.finishedParty.Add(party.id, 0)
				p.logger.Infof("done, party: %s", party.id)
			}()

			return
		case realKey := <-party.ChangedId:
			func() {
				p.logger.Infof("start to changeId, from %s to %s", key, realKey)

				p.partyLock.Lock()
				defer p.partyLock.Unlock()

				p.logger.Infof("changeId,get plock, from %s to %s", key, realKey)
				p.finishedParty.Add(key, 0)
				delete(p.partyManager, key)
				p.logger.Infof("changeId, deleted old id, from %s to %s", key, realKey)

				party.SetId(realKey)
				p.partyManager[realKey] = party
				p.logger.Infof("changeId, set new id, from %s to %s", key, realKey)

				msgs := p.futureMessages[realKey]
				delete(p.futureMessages, realKey)
				p.logger.Infof("changeId, get future messages, from %s to %s. len: %d", key, realKey, len(msgs))
				if 0 != len(msgs) {
					for _, msg := range msgs {
						if nil == msg {
							continue
						}
						p.logger.Infof("changeId and update future messages, from %s to %s, msg: %s", key, realKey, msg.GetMessageID())
						party.Update(msg)
					}
				} else {
					p.logger.Infof("changeId, from %s to %s", key, realKey)
				}

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
