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
	party := p.loadOrNewSignParty(key)

	if nil == party {
		return
	}
	party.Update(ccm)
}

func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	party := p.loadOrNewSignParty(cvm.BlockHash.Bytes())
	if nil == party {
		return
	}
	party.Update(cvm)
}

func (p *Processor) loadOrNewSignParty(keyBytes []byte) Party {
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

	party := &SignParty{belongGroups: p.belongGroups, blockchain: p.MainChain,
		minerReader: p.minerReader, globalGroups: p.globalGroups,
		mi: p.mi.ID, netServer: p.NetServer,
		baseParty: baseParty{
			logger:         p.logger,
			mtx:            sync.Mutex{},
			futureMessages: make(map[string]model.ConsensusMessage),
			Done:           make(chan byte, 1),
			CancelChan:     make(chan byte, 1),
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
				party.Close()

			}()
			return
		case err := <-party.Err:
			func() {
				p.partyLock.Lock()
				defer p.partyLock.Unlock()

				delete(p.partyManager, party.id)
				p.finishedParty.Add(party.id, 0)
				p.logger.Errorf("error: %s, id: %s", err, party.id)
				party.Close()
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
		case <-party.CancelChan:
			p.logger.Debugf("cancel old one, %s", party.id)
			return
		case realKey := <-party.ChangedId:
			func() {
				p.partyLock.Lock()
				defer p.partyLock.Unlock()

				p.finishedParty.Add(key, 0)

				if item, ok := p.partyManager[key]; ok {
					delete(p.partyManager, key)

					// check if already has some messages
					if party2, ok := p.partyManager[realKey]; ok {
						// merging future messages
						for _, msg := range party2.GetFutureMessage() {
							item.Update(msg)
						}
						party2.Cancel()
					}

					item.SetId(realKey)
					p.partyManager[realKey] = item

					p.logger.Infof("changeId, from %s to %s", key, realKey)
				} else {
					p.logger.Errorf("changeId error, from %s to %s", key, realKey)
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
