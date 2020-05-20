package group_create

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/consensus/vrf"
	"x/src/middleware/types"
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
		genesisGroupInfo := &types.GenesisInfo{Group: *coreGroup, VrfPKs: vrfPKs, Pks: pks,}
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
	genesisJoinedGroupString :=`{"GroupHash":"0x496ca9159c99b65825588099195c3d5afbfe068b9e630e27a1df317185b66c11","GroupID":"0x5ef6041d4ccf7682dc60ad6b16be509f49f39305e4f80dc374d2d5fb95052917","GroupPK":"0x1426aae13007f0c6468d7cc822c7bfbbeef04dfac47e9d55b5e3df8250dcff3e5dfa64dd3599c2858bd3deda15177f792732cccdc1347d64693ae2c192f815630bbc9038c098e4c5af4d392a9b3c95af0f4776e46b246b001bb6942188d0f6ae0f1b50a96270f6fbcd38587176eb95182628437760018c8520c67f638e82cf30","SignSecKey":{},"MemberSignPubkeyMap":{"0x05b231ac9481c219957ca7965efecd24c84b2b2bdff6f4b18f090e5059aa87ea":"0x1b8f2afc75fc24688a197f3a90b632cb41a18dd6f490eb31833073e0c3acc73f66bd321100c8ecf9b48f0f2e6dc65c99e2733aebb17aa0241f74eb37442f3909073f9468711807f9b80891d19b4e58bb8d9d44adcd3c2da41da138fbfe1e342611bfa095eee5ea0166bea1d742e2783117202f14b07d0ff06f7f0846395a4afa","0xedcf4a3457361ed75e9d4531362b5d1c8a482c94db3fdb815f821fc18ef59d89":"0x59bbeaf9994228d25e2f32486e159e37ae172a5ef51730ec1ed66a5a685826e6131f3cc596785f4465c3c191cc99e8f98d578167d1383239d524619ae5230dce10764f7731a3cb3bb4d3724e4c777c51e1460935cc414e24ee62a2f9428b54850bdfdd44b34fa0a3903aeeeac322337b8f52c9aaf97a6af28486d3b4d43ca13d","0xf0de943d6dbc269600a606f3624f2f91abaaf547bf733b10c56776ee8b800d9e":"0x8dcf6cdbb10e6a74613910ed5a8afae4ac68aff4fed6babc63c22f1dfc3d3cc05a63085068a0d0e4d91268ff876d6baddde81074bb0b67d5e8266b4ba49feccd294558a3b8b4d0786bfa056601e33f353e4d111e977c87d43ca95119e7498e887d469e47589a1b8d860d1d160824eaa6d65820901a5fa1538482b6831b4154a5"}}`
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
	genesisGroupStr :=`{"GroupInfo":{"GroupID":"0x5ef6041d4ccf7682dc60ad6b16be509f49f39305e4f80dc374d2d5fb95052917","GroupPK":"0x1426aae13007f0c6468d7cc822c7bfbbeef04dfac47e9d55b5e3df8250dcff3e5dfa64dd3599c2858bd3deda15177f792732cccdc1347d64693ae2c192f815630bbc9038c098e4c5af4d392a9b3c95af0f4776e46b246b001bb6942188d0f6ae0f1b50a96270f6fbcd38587176eb95182628437760018c8520c67f638e82cf30","GroupInitInfo":{"GroupHeader":{"Hash":"0x496ca9159c99b65825588099195c3d5afbfe068b9e630e27a1df317185b66c11","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-05-07T18:28:39.723564+08:00","MemberRoot":"0x6917712b132dd183cf4ed97650c387537289dd56057c99e3035268d17da0fab9","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0xf0de943d6dbc269600a606f3624f2f91abaaf547bf733b10c56776ee8b800d9e","0xedcf4a3457361ed75e9d4531362b5d1c8a482c94db3fdb815f821fc18ef59d89","0x05b231ac9481c219957ca7965efecd24c84b2b2bdff6f4b18f090e5059aa87ea"]},"MemberIndexMap":{"0x05b231ac9481c219957ca7965efecd24c84b2b2bdff6f4b18f090e5059aa87ea":2,"0xedcf4a3457361ed75e9d4531362b5d1c8a482c94db3fdb815f821fc18ef59d89":1,"0xf0de943d6dbc269600a606f3624f2f91abaaf547bf733b10c56776ee8b800d9e":0},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["p8DyXDbvL3vt6Xfv34i2zwEkBK6wGGuP+Z3vAwGgw+o=","Ce+KDs2XEHAQYqkxJ3nOCvmRznR4LTXnz4EIUXFgvh8=","YfbY5JjMMPoxuC8TGeV4YWJxEwGlo9Wfeg0LxXJ4Ylk="],"Pubkeys":["0x30d25ba1ff580ecc065bea30b8e6321192ff044a76bc6b3a9532817c145b0b7943a427bd39888d884784e7cceffb340b6d9e51e489cef2bcc71bba26dd1a6f804dec7f3a3f14ae27d41159c9b76ab712c5b6b08f686eb46aa59aff32938e41364b5f6d5d26a56ec398d6d1bed23ea6f6bdbcb04372686b771a0a37fddcaab794","0x34658d34271b6b26a03d797b11b5cfcc093826eb86d14a67c0f1b20e54e6128f85f203133517345b5b033a28bad8ddf98f920da5a38d2fa9ce5ba81ed3b7d9241bf5074255d379819d7899b83234c7d457fa6410d6c409ebafa24c5d833dd605222de5c470ea113a542d15500e5d17d57bbce9527ef3fb1d87386688dd85d533","0x5defe6bf21fb6ad7e207a53cd95691b5b636f6195efee3235d808e4013e9c805517c97aceaf2bed292f1efbdfea96266c7e69e2d6360880850d5ede84bc6aa527b3451796163e8ef7090e9d6057da1f78e473eae0c6ceb22cd233a12df8981d58bb62d10bbb715f4f1e8b563b91080542abe1a2c1f90649450284c16e4eb348a"]}`
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
