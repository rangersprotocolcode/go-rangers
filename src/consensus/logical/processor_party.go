package logical

import (
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
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

func (p *Processor) loadOrNewSignParty(key []byte) SignParty {
	p.lock.Lock()
	defer p.lock.Unlock()

	item, ok := p.partyManager.Load(key)
	if ok {
		party := item.(SignParty)
		return party
	}

	party := SignParty{}
	if nil != party.Start() {
		p.partyManager.Store(key, party)

		// wait until finish
		go func() {
			var realKey []byte
			for {
				select {
				// timeout
				case <-time.After(10 * time.Second):
					p.partyManager.Delete(key)
					p.partyManager.Delete(realKey)
					return
				// finish signing
				case <-party.Done:
					p.partyManager.Delete(realKey)
					return
				case realKey = <-party.ChangedId:
					func() {
						p.lock.Lock()
						defer p.lock.Unlock()

						item, ok := p.partyManager.LoadAndDelete(key)
						if !ok {
							return
						}

						// check if already has some messages
						item2, ok2 := p.partyManager.Load(realKey)
						if ok2 {
							// merging future messages
							party := item.(Party)
							part2 := item2.(Party)
							party.StoreMessages(part2.GetFutureMessage())
							p.partyManager.Store(realKey, party)
						} else {
							p.partyManager.Store(realKey, item)
						}

					}()
				}
			}
		}()
	}

	return party
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
