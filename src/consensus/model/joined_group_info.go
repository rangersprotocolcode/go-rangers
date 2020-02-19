package model

import (
	"sync"
	"x/src/consensus/groupsig"
	"x/src/common"
)

//JoinedGroup
// JoinedGroup stores group-related infos the current node joins in.
// Note that, nodes outside the group don't care the infos
//加入的组的信息和自身在组内的信息
type JoinedGroupInfo struct {
	GroupHash common.Hash
	GroupID   groupsig.ID     // Group ID
	GroupPK   groupsig.Pubkey // Group public key (backup, which can be got from the global group)

	SignSecKey          groupsig.Seckey            // Miner signature private key related to the group
	memberSignPubkeyMap map[string]groupsig.Pubkey // Group related public keys of all members

	lock sync.RWMutex
}

func NewJoindGroupInfo(signSeckey groupsig.Seckey, groupPubkey groupsig.Pubkey, groupHash common.Hash) *JoinedGroupInfo {
	joinedGroup := &JoinedGroupInfo{
		GroupHash:           groupHash,
		GroupPK:             groupPubkey,
		GroupID:             *groupsig.NewIDFromPubkey(groupPubkey),
		SignSecKey:          signSeckey,
		memberSignPubkeyMap: make(map[string]groupsig.Pubkey, 0),
	}
	return joinedGroup
}

func (joinedGroupInfo *JoinedGroupInfo) MemberSignPKNum() int {
	joinedGroupInfo.lock.RLock()
	defer joinedGroupInfo.lock.RUnlock()

	return len(joinedGroupInfo.memberSignPubkeyMap)
}

//getMemberMap
func (joinedGroupInfo *JoinedGroupInfo) GetMemberPKs() map[string]groupsig.Pubkey {
	joinedGroupInfo.lock.RLock()
	defer joinedGroupInfo.lock.RUnlock()

	m := make(map[string]groupsig.Pubkey , 0)
	for key, pk := range joinedGroupInfo.memberSignPubkeyMap {
		m[key] = pk
	}
	return m
}


func (joinedGroupInfo *JoinedGroupInfo) AddMemberSignPK(memberId groupsig.ID, signPK groupsig.Pubkey) {
	joinedGroupInfo.lock.Lock()
	defer joinedGroupInfo.lock.Unlock()

	joinedGroupInfo.memberSignPubkeyMap[memberId.GetHexString()] = signPK
}



// getMemSignPK get the signature public key of a member of the group
func (joinedGroupInfo *JoinedGroupInfo) GetMemberSignPK(memberId groupsig.ID) (pk groupsig.Pubkey, ok bool) {
	joinedGroupInfo.lock.RLock()
	defer joinedGroupInfo.lock.RUnlock()

	pk, ok = joinedGroupInfo.memberSignPubkeyMap[memberId.GetHexString()]
	return
}
