// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package model

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/base"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/vrf"
)

const minerStake = 1

type MinerInfo struct {
	PubKey groupsig.Pubkey

	ID groupsig.ID

	VrfPK vrf.VRFPublicKey

	Stake     uint64
	MinerType byte

	ApplyHeight uint64
	AbortHeight uint64

	WorkingMiners uint64
}

type SelfMinerInfo struct {
	SecretSeed base.Rand
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

func (md *MinerInfo) CanCastAt(h uint64) bool {
	return md.IsWeight() && h > md.ApplyHeight
}

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
