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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package group_create

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"encoding/json"
	"io/ioutil"
	"strings"
)

type genesisGroup struct {
	GroupInfo model.GroupInfo
	VrfPubkey []vrf.VRFPublicKey
	Pubkeys   []groupsig.Pubkey
}

var genesisGroupInfo []*genesisGroup

//GenerateGenesis
func GetGenesisInfo() []*types.GenesisInfo {
	genesisGroups := getGenesisGroupInfo()
	var genesisInfos = make([]*types.GenesisInfo, 0)
	for _, genesis := range genesisGroups {
		sgi := &genesis.GroupInfo
		coreGroup := convertToGroup(sgi)
		vrfPKs := make([][]byte, sgi.GetMemberCount())
		pks := make([][]byte, sgi.GetMemberCount())

		for i, vpk := range genesis.VrfPubkey {
			vrfPKs[i] = vpk
		}
		for i, vpk := range genesis.Pubkeys {
			pks[i] = vpk.Serialize()
		}
		genesisGroupInfo := &types.GenesisInfo{Group: *coreGroup, VrfPKs: vrfPKs, Pks: pks}
		genesisInfos = append(genesisInfos, genesisGroupInfo)
	}
	return genesisInfos
}

//生成创世组成员信息
//BeginGenesisGroupMember
func (p *groupCreateProcessor) BeginGenesisGroupMember() {
	genesisGroups := getGenesisGroupInfo()
	for _, genesis := range genesisGroups {
		if !genesis.GroupInfo.MemExist(p.minerInfo.ID) {
			continue
		}

		jg := genGenesisJoinedGroup()
		sec := new(groupsig.Seckey)
		sec.SetHexString(common.GlobalConf.GetString("gx", "signSecKey", ""))
		jg.SignSecKey = *sec

		p.joinedGroupStorage.JoinGroup(jg, p.minerInfo.ID)
	}

}

func genGenesisJoinedGroup() *model.JoinedGroupInfo {
	genesisJoinedGroupString := `{"GroupHash":"0x9e2ab50121f6551bff20d5f58cd3d1d0bf0790af3f4bd8dcf2c4fb1a5a76f605","GroupID":"0x0ef498a01597b7fc464cb61165f9652b424cf219790fcd23ebd78520e969e5ca","GroupPK":"0x5192a3f736210f4c23244f81469b7eec452dc47264b1ae731464bf12c4a06cb205ca071c89768b58234075f55f74d4e1b8ac4dc5ea003d8351e50f5f951d2d3023a048231507eb08fc57256bc1efa30bd603b1e670ac7820cb58c160ad5d3cf60879f0dfd4f71b487550b2ae35caa67c2f139b89aad5a920b52022923eca37a4","SignSecKey":{},"MemberSignPubkeyMap":{"0x3202cc49a0ded70ed51726a62d8c50690d77d6f096884cdc02a1b3fc180d82c7":"0x49f0dcf0c3b05efcd0ce796e9e80c4064c863ca34c988c1eaa241cd64cb3a2a943e674159805473f365ebf8667e73d7e00cb72bcdaaa180afd2d86492f0cf6cb574f83d3196ff1f87d591e64d9f64d9ff7ea197440f2d0af211d5420238d241c647b5c9e4337b0a7cc80cf851908ed2c9efedb3a0d73208fecab40c5fafdff29","0x9b804faf0008235e5b63aeacf3f890931b19c02feb99b9e3814f3cae768c272c":"0x388d9975556ff9ec09f3793ee9f7cb3063bc31c6cc235ce7c2f4e303a8bacff93ff79a4f48ed4fd1ffb348c7c12c6b03c7004e65fe6d38fb64536959ea6dbb7e099e1ce33dbb488ffe67b096fd3fa806c592fd88646ead9171cfdb0463ae9a7373b287e99c2e154ea557f4ac766a4576ebf487dc8d19606344f1cc731b65ed26","0xa7fc7ab47c01a71b033075fd517493fa76c45bad58106515002b80ad7727b556":"0x89d11d4ca892cf42f65324dfd6b93a18148f209dca2f15575c023c2453f8972e11fb41526ddf1a76cb32f7fa683e30cf00dba8d9c6fdb8ab5d7dccb51e926d6b8b5933f348b9c2383759b0fff052540a2ee041e7a37cd0aca27ffb4b511b7eae2b15cbd1b689d3adc45af4a49d083a3818552cad67b3f037e1e73558aeb32b9d"}}`
	jg := new(model.JoinedGroupInfo)
	json.Unmarshal([]byte(genesisJoinedGroupString), jg)

	return jg
}

func getGenesisGroupInfo() []*genesisGroup {
	if genesisGroupInfo == nil {
		genesisGroupInfo = genGenesisGroupInfo()
	}
	return genesisGroupInfo
}

//genGenesisStaticGroupInfo
func genGenesisGroupInfo() []*genesisGroup {
	genesisGroupStr := `{"GroupInfo":{"GroupID":"0x0ef498a01597b7fc464cb61165f9652b424cf219790fcd23ebd78520e969e5ca","GroupPK":"0x5192a3f736210f4c23244f81469b7eec452dc47264b1ae731464bf12c4a06cb205ca071c89768b58234075f55f74d4e1b8ac4dc5ea003d8351e50f5f951d2d3023a048231507eb08fc57256bc1efa30bd603b1e670ac7820cb58c160ad5d3cf60879f0dfd4f71b487550b2ae35caa67c2f139b89aad5a920b52022923eca37a4","GroupInitInfo":{"GroupHeader":{"Hash":"0x9e2ab50121f6551bff20d5f58cd3d1d0bf0790af3f4bd8dcf2c4fb1a5a76f605","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-12-21T18:59:21.437805+08:00","MemberRoot":"0x9dd5ce3eea58adfae287f156a33f610ec4212863571e98cf14ff75381e97d44e","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x9b804faf0008235e5b63aeacf3f890931b19c02feb99b9e3814f3cae768c272c","0x3202cc49a0ded70ed51726a62d8c50690d77d6f096884cdc02a1b3fc180d82c7","0xa7fc7ab47c01a71b033075fd517493fa76c45bad58106515002b80ad7727b556"]},"MemberIndexMap":{"0x3202cc49a0ded70ed51726a62d8c50690d77d6f096884cdc02a1b3fc180d82c7":1,"0x9b804faf0008235e5b63aeacf3f890931b19c02feb99b9e3814f3cae768c272c":0,"0xa7fc7ab47c01a71b033075fd517493fa76c45bad58106515002b80ad7727b556":2},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["E01KfPuHsQJt50uP2qBr1J3Xa1D0lGAHk+svD6tRtl8=","gYYY7q3AKVG6QEZbE3MBJRlUgTLE/6+i0fBr2br7eqo=","BKNeNZqZ1EctVBVxY+A7HGP7+ic08LQ/63hMy5CwjXw="],"Pubkeys":["0x2fc1ec403367337b97364f17686d5ae5e149f0fae8c73b30bcd79193fa21ad497691687f09272623c70fbf7b2886b9ca39f5eb24b764f222a9e72fbdf0810c9c51ecdf2f1e49b99066d543f9d2e866aa5f6319858460643a9d762ccbb2285c476a2ebf0e9eb05074fa0ab98e29487490afd832e9ea8c6bdd09a7ac63274365c7","0x244ec71e1336209dfe1425c17517e410ddd1c365d565ac85f41491bb086493c88ce6208776929b9fd337c823a3cb286c3a2846c7fcfde6ea50328efc9c282267574d6fa7d58f24d9c474750b4b8f17b0e310f96fa14637225e4972206513f9982309e3011d55088de78a408b9e8e9b1aeef15d5b86769cf7aae0ae7e028cb661","0x844cb366c50ddea5ec4c5760370a1e6b391a6c9639b87e00eb3f7e089530deea13056f8803fbc55d2df92a5ba3a1430b49f86af82320e78ec74f25217e73a3e640080d3d087a6d7a0758179f646278aadb1a8124975bde780f880a503a507019534d89a65e810c40a95bd21fb8aa709e92e64006d1e66f2d3ccd3552ae942fc1"]}`
	splited := strings.Split(genesisGroupStr, "&&")
	var groups = make([]*genesisGroup, 0)
	for _, split := range splited {
		genesis := new(genesisGroup)
		err := json.Unmarshal([]byte(split), genesis)
		if err != nil {
			panic(err)
		}
		group := genesis.GroupInfo
		group.BuildMemberIndex()
		groups = append(groups, genesis)
	}

	return groups
}

//ConvertStaticGroup2CoreGroup
func convertToGroup(groupInfo *model.GroupInfo) *types.Group {
	members := make([][]byte, groupInfo.GetMemberCount())
	for idx, miner := range groupInfo.GroupInitInfo.GroupMembers {
		members[idx] = miner.Serialize()
	}
	return &types.Group{
		Header:    groupInfo.GetGroupHeader(),
		Id:        groupInfo.GroupID.Serialize(),
		PubKey:    groupInfo.GroupPK.Serialize(),
		Signature: groupInfo.GroupInitInfo.ParentGroupSign.Serialize(),
		Members:   members,
	}
}

func readGenesisJoinedGroup(file string, sgi *model.GroupInfo) *model.JoinedGroupInfo {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic("read genesis joinedGroup file failed!err=" + err.Error())
	}
	var group = new(model.JoinedGroupInfo)
	err = json.Unmarshal(data, group)
	if err != nil {
		panic(err)
	}
	group.GroupPK = sgi.GroupPK
	group.GroupID = sgi.GroupID
	return group
}
