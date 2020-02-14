package model

import (
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/base"
	"x/src/consensus/vrf"
)

const minerStake = 1

type MinerDO struct {
	PK          groupsig.Pubkey
	VrfPK       vrf.VRFPublicKey
	ID          groupsig.ID
	Stake       uint64
	NType       byte
	ApplyHeight uint64
	AbortHeight uint64
}

//在该高度是否可以铸块
func (md *MinerDO) CanCastAt(h uint64) bool {
	return md.IsWeight()
}

//在该高度是否可以加入组
func (md *MinerDO) CanJoinGroupAt(h uint64) bool {
	return md.IsLight()
}

func (md *MinerDO) IsLight() bool {
	return md.NType == common.MinerTypeValidator
}

func (md *MinerDO) IsWeight() bool {
	return md.NType == common.MinerTypeProposer
}

type SelfMinerDO struct {
	MinerDO
	SecretSeed base.Rand //私密随机数
	SK         groupsig.Seckey
	VrfSK      vrf.VRFPrivateKey
}

func (mi *SelfMinerDO) Read(p []byte) (n int, err error) {
	bs := mi.SecretSeed.Bytes()
	if p == nil || len(p) < len(bs) {
		p = make([]byte, len(bs))
	}
	copy(p, bs)
	return len(bs), nil
}

func NewSelfMinerDO(address []byte) SelfMinerDO {
	var mi SelfMinerDO
	mi.SecretSeed = base.RandFromBytes(address)
	mi.SK = *groupsig.NewSeckeyFromRand(mi.SecretSeed)
	mi.PK = *groupsig.NewPubkeyFromSeckey(mi.SK)
	mi.Stake = minerStake
	mi.ID = groupsig.DeserializeId(address)

	var err error
	mi.VrfPK, mi.VrfSK, err = vrf.VRFGenerateKey(&mi)
	if err != nil {
		panic("generate vrf key error, err=" + err.Error())
	}
	return mi
}

func (mi SelfMinerDO) GetMinerID() groupsig.ID {
	return mi.ID
}

func (mi SelfMinerDO) GetSecret() base.Rand {
	return mi.SecretSeed
}

func (mi SelfMinerDO) GetDefaultSecKey() groupsig.Seckey {
	return mi.SK
}

func (mi SelfMinerDO) GetDefaultPubKey() groupsig.Pubkey {
	return mi.PK
}

func (mi SelfMinerDO) GenSecretForGroup(h common.Hash) base.Rand {
	r := base.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}
