package logical

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"sync"
	"time"
)

func (p *Processor) OnMessageCastV2(ccm *model.ConsensusCastMessage) {
	key := p.generatePartyKey(ccm.BH)
	party := p.loadOrNewSignParty(key)

	party.Update(ccm)
}

func (p *Processor) OnMessageVerifyV2(cvm *model.ConsensusVerifyMessage) {
	party := p.loadOrNewSignParty(cvm.BlockHash.Bytes())

	party.Update(cvm)
}

func (p *Processor) loadOrNewSignParty(keyBytes []byte) Party {
	p.partyLock.Lock()
	defer p.partyLock.Unlock()

	key := common.ToHex(keyBytes)
	item, ok := p.partyManager[key]
	if ok {
		return item
	}

	party := SignParty{belongGroups: p.belongGroups, blockchain: p.MainChain,
		minerReader: p.minerReader, globalGroups: p.globalGroups,
		mi: p.mi.ID, netServer: p.NetServer,
		baseParty: baseParty{
			logger:         p.logger,
			mtx:            sync.Mutex{},
			futureMessages: make(map[string]model.ConsensusMessage),
			Done:           make(chan byte, 1),
			Err:            make(chan error, 1),
		},
	}
	if nil != party.Start() {
		p.partyManager[key] = &party

		// wait until finish
		go func() {
			var realKey string
			for {
				select {
				// timeout
				case <-time.After(10 * time.Second):
					delete(p.partyManager, key)
					delete(p.partyManager, realKey)
					return
				case err := <-party.Err:
					delete(p.partyManager, key)
					delete(p.partyManager, realKey)
					p.logger.Errorf("error: %s", err)
					return
				// finish signing
				case <-party.Done:
					delete(p.partyManager, realKey)
					return
				case realKey = <-party.ChangedId:
					func() {
						p.partyLock.Lock()
						defer p.partyLock.Unlock()

						item, ok := p.partyManager[key]
						if !ok {
							// error
							return
						}

						// check if already has some messages
						item2, ok2 := p.partyManager[realKey]
						if ok2 {
							// merging future messages
							item.StoreMessages(item2.GetFutureMessage())
						}

						p.partyManager[realKey] = item
						delete(p.partyManager, key)
					}()
				}
			}
		}()
	}

	return &party
}

func (p *Processor) generatePartyKey(bh types.BlockHeader) []byte {
	result := make([]byte, 0)
	result = append(result, utility.UInt64ToByte(bh.Height)...)
	result = append(result, bh.PreHash.Bytes()...)
	result = append(result, bh.Castor...)
	result = append(result, bh.ProveValue.Bytes()...)
	result = append(result, bh.TxTree.Bytes()...)
	result = append(result, bh.GroupId...)
	return result
}
