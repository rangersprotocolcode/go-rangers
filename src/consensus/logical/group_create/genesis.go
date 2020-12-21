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
	genesisJoinedGroupString := `{"GroupHash":"0xb92a4aceba5d8d2197c61d5b23ff64b745404d459f7faeec45ff2bb35429d2cc","GroupID":"0x57160ffd9a17c444edaef80202d55d7593be73b2c9466391061f2c43208ef735","GroupPK":"0x79b63f51cc9bb5ab1e6ba324f766b48d929cbd9f8b2df3bf99c9beed34429a5626d4509ae8a925ecf376dd95c20b180bf411d092065049a2565415bf8be1a4053cd0cc25f2a477ef8ffe5a417a62407ec53ce7c8b931501b7ae4591f85607d092536697636d0f623afd92adb47a8020b25b04dd44f121e0945829d03964af240","SignSecKey":{},"MemberSignPubkeyMap":{"0x0b94a75753cef8a6579b467f589de2ce9593192bc8dba19ca11fcfc5efc2ae16":"0x60f97d02c76cbab3963c83528d839b3da9bda6ffa1679fe70b3bb5d86fd0ba171e76ac639090212da4f12b3a62b055cdfbe465b89d005b59b2d60c126ebf1076109607eafac833d7c50e48e5c52b62218a2b330484aec83ff8658e36e02ab7634dae0c5f23dc9c5c38620c9ce928369c831374091d698725f166b29c45ce6ad1","0x6e6ce756c9cc9f296b0dd3480d947ca632f36de6f5cd459bc05364cd57b339de":"0x5f69d04efeb84011d5a7d510cb28367b9b548f12119552ba3c7f88c879d21289193bb14b6f6574bc8660e8066309d6060da64bc72f43ffd4c29e7c06d5f09e0461799c281b21416255af155275b8767fc785fb945328e0ca39ac5e893e9f0a0a163da27923adbf7d631f05c0596d5dfe21bcbc172838938553629fd4a762369f","0x8d5083e77fabd6a6a3aac32855b591c24eea8ee58435deb1882fe3ce1d1bf2df":"0x1cbca8d79c072f4af7aa8044c4f8d90dfbe59acd37fd3a75db116b542151866517fd4af9f7fcd63da38c0f4b21e82e7958a7000a54a605850de2f84107f2a67377ac18339a6fc1f5af37d773cceb3d554c67d9c273703ac2165079c7c037c34688d3bc76661db3fc2357c4b8fced5d031fceff9ff1566f22b027b2b86ccca3be"}}`
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
	genesisGroupStr := `{"GroupInfo":{"GroupID":"0x57160ffd9a17c444edaef80202d55d7593be73b2c9466391061f2c43208ef735","GroupPK":"0x79b63f51cc9bb5ab1e6ba324f766b48d929cbd9f8b2df3bf99c9beed34429a5626d4509ae8a925ecf376dd95c20b180bf411d092065049a2565415bf8be1a4053cd0cc25f2a477ef8ffe5a417a62407ec53ce7c8b931501b7ae4591f85607d092536697636d0f623afd92adb47a8020b25b04dd44f121e0945829d03964af240","GroupInitInfo":{"GroupHeader":{"Hash":"0xb92a4aceba5d8d2197c61d5b23ff64b745404d459f7faeec45ff2bb35429d2cc","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-12-21T20:12:20.748244+08:00","MemberRoot":"0x6675df6244d77774d9e5469b5a0539ba03b45fd59be042b1437ed74fcededc76","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x6e6ce756c9cc9f296b0dd3480d947ca632f36de6f5cd459bc05364cd57b339de","0x0b94a75753cef8a6579b467f589de2ce9593192bc8dba19ca11fcfc5efc2ae16","0x8d5083e77fabd6a6a3aac32855b591c24eea8ee58435deb1882fe3ce1d1bf2df"]},"MemberIndexMap":{"0x0b94a75753cef8a6579b467f589de2ce9593192bc8dba19ca11fcfc5efc2ae16":1,"0x6e6ce756c9cc9f296b0dd3480d947ca632f36de6f5cd459bc05364cd57b339de":0,"0x8d5083e77fabd6a6a3aac32855b591c24eea8ee58435deb1882fe3ce1d1bf2df":2},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["umJDpqAudCSbikwOF4abs4yyFWBN/2uakTy05Z38V94=","UzcroAODq+YkSL8dU23DuwG4bQa9dnqJ91qcFvhaRAI=","DuZyt3jlFSrgrcSY/o++I1d77cL+MSJZe1+sUV9ogdM="],"Pubkeys":["0x0dfd23d64042770b93fad29860fd28bf4b58b6696abec7aafd42d872e6186f1a563dd92946896ebc377a7c4ae97d20c02aa7750dd266c422cbfb6b49b5e228a11b06ad59d4c0ae56ee1a3631e4b922a89027480a308358fa91113d5d0582f80a749135faac3789a8da3398efc4d9f2f4173eb76fce26b785c799561cf98ec178","0x475ec03b13341f470f38c36b8dc20c9663942e2966de3e776d1e83d2badf57c8623606066ff35669bab5d9e78460375c8d38b9fc55c9e1b9382d43c434156186715e7b20009dfcb36e58030f4c58132968d8840cf3500125b71d9edd48dc8a7c516255cc466e304e6f51d3cbf065d5e94373785dc713787ad0bf6dcf096295c4","0x5567306bda284dbead63e82dd1d252f3fe9e054bdbc660e7a6a9d3439898349b834bf90c0423f720322f75b25754c9f1a6b27ce0b9d9375f3551eed41b74d957414e5a48cc9b046af6140bf56a019a7174e4b4f02d94fe0f1ef9b9c8944d84a4503579a5b2e6b6175017de1220ae8ab29a6c9d0eb0a8fb8a97a860b6dc9a5b88"]}`
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
