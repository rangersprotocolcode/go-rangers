package model

import (
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"fmt"
	"sync"
)

type GroupSignGenerator struct {
	threshold      int                           //阈值
	witnessSignMap map[string]groupsig.Signature //见证人列表

	groupSign groupsig.Signature //输出的组签名
	lock      sync.RWMutex
}

func NewGroupSignGenerator(threshold int) *GroupSignGenerator {
	return &GroupSignGenerator{
		witnessSignMap: make(map[string]groupsig.Signature, 0),
		threshold:      threshold,
	}
}

func (gs *GroupSignGenerator) Threshold() int {
	return gs.threshold
}

func (gs *GroupSignGenerator) WitnessCount() int {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return len(gs.witnessSignMap)
}

func (gs *GroupSignGenerator) GetWitnessSign(id groupsig.ID) (groupsig.Signature, bool) {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	if s, ok := gs.witnessSignMap[id.GetHexString()]; ok {
		return s, true
	}
	return groupsig.Signature{}, false
}

func (gs *GroupSignGenerator) AddWitnessSign(id groupsig.ID, signature groupsig.Signature) (add bool, generated bool) {
	if gs.SignRecovered() {
		return false, true
	}

	return gs.addWitnessForce(id, signature)
}

func (gs *GroupSignGenerator) SignRecovered() bool {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return gs.groupSign.IsValid()
}

func (gs *GroupSignGenerator) GetGroupSign() groupsig.Signature {
	gs.lock.RLock()
	defer gs.lock.RUnlock()

	return gs.groupSign
}

func (gs *GroupSignGenerator) VerifyGroupSign(gpk groupsig.Pubkey, data []byte) bool {
	return groupsig.VerifySig(gpk, data, gs.GetGroupSign())
}
func (gs *GroupSignGenerator) String() string {
	return fmt.Sprintf("当前分片数%v，需分片数%v", gs.WitnessCount(), gs.threshold)
}

//不检查是否已经recover，只是把分片加入
func (gs *GroupSignGenerator) addWitnessForce(id groupsig.ID, signature groupsig.Signature) (add bool, generated bool) {
	gs.lock.Lock()
	defer gs.lock.Unlock()

	key := id.GetHexString()
	if _, ok := gs.witnessSignMap[key]; ok {
		return false, false
	}
	gs.witnessSignMap[key] = signature

	if len(gs.witnessSignMap) >= gs.threshold {
		return true, gs.genGroupSign()
	}
	return true, false
}

func (gs *GroupSignGenerator) genGroupSign() bool {
	if gs.groupSign.IsValid() {
		return true
	}

	sig := groupsig.RecoverGroupSignature(gs.witnessSignMap, gs.threshold)
	if sig == nil {
		return false
	}
	gs.groupSign = *sig
	if len(gs.groupSign.Serialize()) == 0 {
		//stdL("!!!!!!!!!!!!!!!!!!!!!!!!!!!1sign is empty!")
	}
	return true
}
