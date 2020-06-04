package model

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
)

const minerStake = 1

//矿工信息
type MinerInfo struct {
	PubKey groupsig.Pubkey
	ID     groupsig.ID

	VrfPK vrf.VRFPublicKey

	Stake     uint64
	MinerType byte

	ApplyHeight uint64
	AbortHeight uint64
}

type SelfMinerInfo struct {
	SecretSeed base.Rand //私密随机数
	SecKey     groupsig.Seckey
	VrfSK      vrf.VRFPrivateKey

	MinerInfo
}

func NewSelfMinerInfo(privateKey common.PrivateKey) SelfMinerInfo {
	var mi SelfMinerInfo
	mi.SecretSeed = base.RandFromBytes(privateKey.PrivKey.D.Bytes())
	mi.SecKey = *groupsig.NewSeckeyFromRand(mi.SecretSeed)
	mi.PubKey = *groupsig.GeneratePubkey(mi.SecKey)
	mi.Stake = minerStake
	idBytes := privateKey.GetPubKey().GetID()
	mi.ID = groupsig.DeserializeID(idBytes[:])

	var err error
	mi.VrfPK, mi.VrfSK, err = vrf.VRFGenerateKey(&mi)
	if err != nil {
		panic("generate vrf key error, err=" + err.Error())
	}
	return mi
}

func (mi SelfMinerInfo) GenSecretForGroup(h common.Hash) base.Rand {
	r := base.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}

func (mi MinerInfo) GetMinerID() groupsig.ID {
	return mi.ID
}

func (md *MinerInfo) IsLight() bool {
	return md.MinerType == common.MinerTypeValidator
}

func (md *MinerInfo) IsWeight() bool {
	return md.MinerType == common.MinerTypeProposer
}

//在该高度是否可以铸块
func (md *MinerInfo) CanCastAt(h uint64) bool {
	return md.IsWeight() && h > md.ApplyHeight
}

//在该高度是否可以加入组
func (md *MinerInfo) CanJoinGroupAt(h uint64) bool {
	return md.IsLight() && h > md.ApplyHeight
}

func (md *SelfMinerInfo) Read(p []byte) (n int, err error) {
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
