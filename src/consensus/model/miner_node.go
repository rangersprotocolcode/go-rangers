package model

import (
	"x/src/middleware/types"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/base"
	"x/src/consensus/vrf"
)

const minerStake = 1

//矿工信息
type MinerInfo struct {
	SecretSeed base.Rand //私密随机数
	SecKey     groupsig.Seckey
	PubKey     groupsig.Pubkey
	ID         groupsig.ID

	VrfSK vrf.VRFPrivateKey
	VrfPK vrf.VRFPublicKey

	Stake     uint64
	MinerType byte

	ApplyHeight uint64
	AbortHeight uint64
}

func NewSelfMinerInfo(address []byte) MinerInfo {
	var mi MinerInfo
	mi.SecretSeed = base.RandFromBytes(address)
	mi.SecKey = *groupsig.NewSeckeyFromRand(mi.SecretSeed)
	mi.PubKey = *groupsig.GeneratePubkey(mi.SecKey)
	mi.Stake = minerStake
	mi.ID = groupsig.DeserializeID(address)

	var err error
	mi.VrfPK, mi.VrfSK, err = vrf.VRFGenerateKey(&mi)
	if err != nil {
		panic("generate vrf key error, err=" + err.Error())
	}
	return mi
}

func (mi MinerInfo) GenSecretForGroup(h common.Hash) base.Rand {
	r := base.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}

func (mi MinerInfo) GetMinerID() groupsig.ID {
	return mi.ID
}

func (md *MinerInfo) IsLight() bool {
	return md.MinerType == types.MinerTypeLight
}

func (md *MinerInfo) IsWeight() bool {
	return md.MinerType == types.MinerTypeHeavy
}

//在该高度是否可以铸块
func (md *MinerInfo) CanCastAt(h uint64) bool {
	//todo 这里不考虑高度？
	return md.IsWeight()
}

//在该高度是否可以加入组
func (md *MinerInfo) CanJoinGroupAt(h uint64) bool {
	//todo 这里不考虑高度？
	return md.IsLight()
}

func (md *MinerInfo) Read(p []byte) (n int, err error) {
	bs := md.SecretSeed.Bytes()
	if p == nil || len(p) < len(bs) {
		p = make([]byte, len(bs))
	}
	copy(p, bs)
	return len(bs), nil
}


//func (mi MinerInfo) GetSecretSeed() base.Rand {
//	return mi.SecretSeed
//}

//func (mi MinerInfo) GetDefaultSecKey() groupsig.Seckey {
//	return mi.SecKey
//}
//
//func (mi MinerInfo) GetDefaultPubKey() groupsig.Pubkey {
//	return mi.PubKey
//}

//type MinerDO struct {
//	PK          groupsig.Pubkey
//	VrfPK       vrf.VRFPublicKey
//	ID          groupsig.ID
//	Stake       uint64
//	NType       byte
//	ApplyHeight uint64
//	AbortHeight uint64
//}
//
////在该高度是否可以铸块
//func (md *MinerDO) CanCastAt(h uint64) bool {
//	return md.IsWeight()
//}
//
////在该高度是否可以加入组
//func (md *MinerDO) CanJoinGroupAt(h uint64) bool {
//	return md.IsLight()
//}
//
//func (md *MinerDO) IsLight() bool {
//	return md.NType == types.MinerTypeLight
//}
//
//func (md *MinerDO) IsWeight() bool {
//	return md.NType == types.MinerTypeHeavy
//}
//
//type SelfMinerDO struct {
//	MinerDO
//	SecretSeed base.Rand //私密随机数
//	SK         groupsig.Seckey
//	VrfSK      vrf.VRFPrivateKey
//}
//
//func (mi *SelfMinerDO) Read(p []byte) (n int, err error) {
//	bs := mi.SecretSeed.Bytes()
//	if p == nil || len(p) < len(bs) {
//		p = make([]byte, len(bs))
//	}
//	copy(p, bs)
//	return len(bs), nil
//}
//
//func NewSelfMinerDO(address []byte) SelfMinerDO {
//	var mi SelfMinerDO
//	mi.SecretSeed = base.RandFromBytes(address)
//	mi.SK = *groupsig.NewSeckeyFromRand(mi.SecretSeed)
//	mi.PK = *groupsig.GeneratePubkey(mi.SK)
//	mi.Stake = minerStake
//	mi.ID = groupsig.DeserializeID(address)
//
//	var err error
//	mi.VrfPK, mi.VrfSK, err = vrf.VRFGenerateKey(&mi)
//	if err != nil {
//		panic("generate vrf key error, err=" + err.Error())
//	}
//	return mi
//}
//
//func (mi SelfMinerDO) GetMinerID() groupsig.ID {
//	return mi.ID
//}
//
//func (mi SelfMinerDO) GetSecret() base.Rand {
//	return mi.SecretSeed
//}
//
//func (mi SelfMinerDO) GetDefaultSecKey() groupsig.Seckey {
//	return mi.SK
//}
//
//func (mi SelfMinerDO) GetDefaultPubKey() groupsig.Pubkey {
//	return mi.PK
//}
//
//func (mi SelfMinerDO) GenSecretForGroup(h common.Hash) base.Rand {
//	r := base.RandFromBytes(h.Bytes())
//	return mi.SecretSeed.DerivedRand(r[:])
//}
