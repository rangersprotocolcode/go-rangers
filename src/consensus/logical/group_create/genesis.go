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
	genesisJoinedGroupString := `{"GroupHash":"0x0000000000000000000000000000000000000000000000000000000000000000","GroupID":"0x2a63497b8b48bc85ae6f61576d4a2988e7b71e1c02898ea2a02ead17f076bf92","GroupPK":"0x1f1fc4d3707b60990e1e80fcdb941fd124cf617e187ffc032c2a5c721dea5bcf1272174f98f1f1fae039df6d661e695e39eca5dcbfa97adc84fe94ea9be0e1812c79239e11d0f762a58df0cae07ba81b6c63f687e40354b45e9a83d80dde0e2a27c44ab34a87afddb56ee809df11cd40d2376ca0e7409ec9ef9051c0d78c96e3","SignSecKey":{},"MemberSignPubkeyMap":{"0x445173ab39681491f688e8b5b11f3f51041ce0d05b5ddd75ccc86f4c3343a418":"0x14d852bd2ae1d416c62a4ca98ebfb4825c7da9681572360e3cf678888088bb141fd1546dd661ba8c8c23649a0e5562e72a62e2b8632d8c77585818820584f9391a398295f35bae8a40de7909962ad50c415b3ddc5924ee6877c9208bf81e17412f077393c0717d6fcd4623716c98de7246100de0fec8627e35967f7975948aa1","0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443":"0x1f0135c825f4fffb6d0f91a66d08e2b322e8adb2ee561b1b050d7de26d1f287b0c54067c47677581cd56f2976be170b54cdcf0aeba76513f38105894933e9d0524a648bfd572fe0c1c14672afef7be73ac411f3d76fdb7828690ab66f29847941754a9eb4f9fcb44ed2c9e497bb1df0517558224cb4bb955a959f9bd3760f9e8","0x9f67e8e7785f489a1e54a8ff8e7ca7859fcfc5c2cef2278d5bb6528a0c5c609e":"0x1a71862c8d6e22ab027d93200488ec57edc583dd6e72d213d3ea9032955d7e4a05641d0b1d3d3c2f2d8b5d981098ec3c504596a2f372db4d091f90f89b04cff50c3d8d16131b2a452c25a651725ec0d16db6cfbdf5645e1364b23b22db27959909bf5622e395528d8e77ed3b65efbb267708bb0af1105be0560d7634604e6eaa"}}`
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
	genesisGroupStr := `{"GroupInfo":{"GroupID":"0x2a63497b8b48bc85ae6f61576d4a2988e7b71e1c02898ea2a02ead17f076bf92","GroupPK":"0x1f1fc4d3707b60990e1e80fcdb941fd124cf617e187ffc032c2a5c721dea5bcf1272174f98f1f1fae039df6d661e695e39eca5dcbfa97adc84fe94ea9be0e1812c79239e11d0f762a58df0cae07ba81b6c63f687e40354b45e9a83d80dde0e2a27c44ab34a87afddb56ee809df11cd40d2376ca0e7409ec9ef9051c0d78c96e3","GroupInitInfo":{"GroupHeader":{"Hash":"0x204fdf622b91c056fa7c3252c3d52e1ef6ca9b97cf0f813b1f511416c823f63e","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-03-04T19:18:47.356371+08:00","MemberRoot":"0xb1f9f15905dea586d95e34ceed6098d625b5a90109e8310475f8068021eaf962","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x445173ab39681491f688e8b5b11f3f51041ce0d05b5ddd75ccc86f4c3343a418","0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","0x9f67e8e7785f489a1e54a8ff8e7ca7859fcfc5c2cef2278d5bb6528a0c5c609e"]},"MemberIndexMap":{"0x445173ab39681491f688e8b5b11f3f51041ce0d05b5ddd75ccc86f4c3343a418":0,"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443":1,"0x9f67e8e7785f489a1e54a8ff8e7ca7859fcfc5c2cef2278d5bb6528a0c5c609e":2},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["JrO1OIE+Ykmm7U33/2MRQ3tFc9NHugW6kr4r/ZcsXUw=","qKo5cMxxh3hT+I3Z1UM34407NuM2yQFIaW+4ywh2/Ls=","ft97PKiR2Y28QtlVvaRfhEeLeuXGxuK+bIZe8pu2mbg="],"Pubkeys":["0x01d3c0708e58d77b94ab6fa8af991a854e57fd3e491d6cf17b47f9a2b052e3e5027dc9e56aca4b232dbc6a860d34a17f16cd5b1edcdadfb33e54077a8c27fc6d265a3ea7a039db31ba8db0165987d1fb8fbf5e161c2bd67abee3a879dca35aac29cbaf949c6c57ac7fe315038e35976a08416e020a671afcc0e2c910a8662dbe","0x1f93119d345cbda059cc3b984f8f8e0be98cd0b4b5479e38beac5866c11cd6112c5a95f9f8fc54f5d003127cbd61e137320846e14aa6dccc4e681982bef08e17094e1b30137b8c4124ebf1c088bbd897104065e664bda1ff015b0881759adf7a1d3e4f439dae249c33f4a9684f5d04f9c0b05236c5ebafbc23ca891cea2cad43","0x04fd9d1e30a8627a1d27fbc62be906cf0a3d78ddad98e71096cf1fe06379a1231958e8757cf630348b9989f0612ceffd5e41b962abaacaf3e9cbd5295c1d27f911da7912cebb18232fd743a6a273d22fd15a87667692cff53a89c07c78e1b377104539171bf7b76cc7222ef0ff4f75b9936c449ae29401abe6614711fae894b9"]}`
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
