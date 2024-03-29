package logical

import (
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
)

func (p *Processor) OnMessageCastV2(ccm *model.ConsensusCastMessage) {
	p.lock.Lock()
	defer p.lock.Unlock()

	key := p.generatePartyKey(ccm.BH)
	if _, ok := p.partyManager.Load(key); ok {
		return
	}

	party := SignParty{}
	if nil != party.Start() {
		p.partyManager.Store(key, party)
		go func() {
			<-party.Done
			p.partyManager.Delete(key)
		}()

		party.Update(ccm)
	}
}

func (p *Processor) generatePartyKey(bh types.BlockHeader) []byte {
	result := make([]byte, 0)
	result = append(result, utility.UInt64ToByte(bh.Height)...)
	result = append(result, bh.PreHash.Bytes()...)
	result = append(result, bh.Castor...)
	result = append(result, bh.ProveValue.Bytes()...)
	result = append(result, bh.TxTree.Bytes()...)
	return result
}
