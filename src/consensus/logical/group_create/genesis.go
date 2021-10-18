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
	genesisJoinedGroupString := `{"GroupHash":"0x275d1ec7e591231e9de02a0d29cf465a5dfed5aea15feff938f85714bcc52296","GroupID":"0xd588982453c1f6278564b44667573ec3eb6ce326a63e33e89b091cfe3de2a887","GroupPK":"0x608fdd9a055d2e688401220c9e3d0e9c2f26c354b3be07f1a06c4c6d769ae3cd38828f2c7b5b531fff8306a61fbdfa94e5eb59cecff7b28f4abba21f6037a2792d36d3544b2d7057405482a2ebde48f06e302dd1c91fbb6a47d393bb3f19f5660e2909d46e8445f857f1e63dff2bd10913663ca3c8a6bcbc64b0d49d92693251","SignSecKey":{},"MemberSignPubkeyMap":{"0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77":"0x81765735aa48fa38eab581a961630adb459bca8b33a540faed3abaf5baa0a4e44ca776e946ffe9b857421c21062ac3837684b72ef36d174d140f0990972f1da94509dba0edb65dd72b647143776b0d6c25c3a47d0419581d2730bae0e3c0ca34450cfb9346b7fadd6737ffb2d78048ec0360f2a57eb8db870d79d8e51ca31817","0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123":"0x6a2a1db50572d6f026ab8274357e961229dbc0dc73e6e5e746fc08237875dffd725bb28a4e4e7e4b5fcc299ab3ff2fb18e64b1822a44b4486b2045d34b5b7e2d8a6a90742f02122be4efd09c6fb454ce606ee80666fd75f792c0e96140bc6b9283b90f32b5916bfa721e18422f5255a73ca9330151367bbc1731ff797a5c7496","0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160":"0x8cad31d6d8a33b3de6cf964c23aa66e17e56b5aa5914c9d3702871a2e2b41cb63c34a8741ac08ac8d64147b3b72b4ea26f59e6257d26ff59dc0e5088ea0a7e3188d20929c11b012b1e6ed2c2ef4e4b29b9ca6b4e1bd5adff6b899fba6e9df731837b85e47e53969b414839ee28efc4a245e7489726b9c9e5ff2d50fa1e676a4d"}}`
	jg := new(model.JoinedGroupInfo)
	err := json.Unmarshal([]byte(genesisJoinedGroupString), jg)
	if nil != err {
		panic(err)
	}

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
	genesisGroupStr := `{"GroupInfo":{"GroupID":"0xd588982453c1f6278564b44667573ec3eb6ce326a63e33e89b091cfe3de2a887","GroupPK":"0x608fdd9a055d2e688401220c9e3d0e9c2f26c354b3be07f1a06c4c6d769ae3cd38828f2c7b5b531fff8306a61fbdfa94e5eb59cecff7b28f4abba21f6037a2792d36d3544b2d7057405482a2ebde48f06e302dd1c91fbb6a47d393bb3f19f5660e2909d46e8445f857f1e63dff2bd10913663ca3c8a6bcbc64b0d49d92693251","GroupInitInfo":{"GroupHeader":{"Hash":"0x275d1ec7e591231e9de02a0d29cf465a5dfed5aea15feff938f85714bcc52296","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-12-23T10:31:42.523464+08:00","MemberRoot":"0x519750e82dd561f03eabae5ee7d1f11df0f6c14eb356757ad7bb7de7a51f1448","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123","0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77","0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160"]},"MemberIndexMap":{"0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77":1,"0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123":0,"0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160":2},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["JevvUFyYiszl6wg3fPRA1zUBo8TKiphhjJy7Hy0nfcU=","8QZCqd7xDETZ1eA3QaQJUWsAcwYCwrM43UhyLGh9VP4=","mLYbDvkkPQFwnVRZCI59t1iLum4qdMIkEzADhpohvRY="],"Pubkeys":["0x81b7daee9550ac17c0446cb42bcf8cc87155fa36d641320fd351f5cfc5ce435a20457d5f3fe83d79aff4fb2cbcc0812242040d066463998eb5729866f1ce3979671fb242204c961cf2723321729aacbfff2c014f943ab8794785be47319efd76078d7cc657750903e24be9f09685fee361d321963fbc93075b9bdb18dd6f53f1","0x27bfa326a01e6640f61666801a6fc1acc37a18d5d4804c92e304dacef25d606f385fe9f050aeb4d5c9ee4c9d442dd535d65380aa26bfc33d92ab6007b91afdf41ddd68c1c6ab3e11154578eea8a364084fe8e4b263d675dd44ad8be223b75bcc0db0d4cbc4651eb2e5351ab5e01f3d3298227bbe342692582ab5afff39d237b9","0x2ece69e08094f7cdd3f19db009b907225438eea203ac19a27b2c7ac43350569105549567de059c3d247dd2fb9190f77f732d89f84d7540c752b7f837fabad6498c1570d42804397c1beee1876548b0c7abf804076dcfab61511dbd6a84b480230e90cfe8c2fa03e6e997fac2b5507f79eb6a2fdd215cc92bf343b7473b4b4df1"]}`
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
