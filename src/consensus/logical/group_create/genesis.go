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
	genesisJoinedGroupString := `{"GroupHash":"0x0aaea93ad8571f27f55d7217dfa7d7bfcf52f69447f6cedbb45e63cf20a076c6","GroupID":"0xbafd7806e7e8eedde0ba38cf268f4e2accee582a92557deb8488e134d2f74dfd","GroupPK":"0x1ae5ce35b47114baae9724f3c4eb7ecf08bdbc88784db6e3dc2edb25d1784e234da685a9d6cc57367bdacaab2b36ef81d2419f49fa4640b4c6cc83929a48154c7da40a074c4acf76831f2e14db48e6907af5410b21e3be76cc2c2412fb71fdb017af2316b4876c65cbbed0defaa4e8b401612a8eff0d2c3c334965cb36a26daa","SignSecKey":{},"MemberSignPubkeyMap":{"0x05bcd6acb406e04310c093b54b399052a68eb1f76d4a36f6db1b3402b6e43a19":"0x0d100ec81ddaa2fcd4c6271749038a69f9ad834b5e0fbfd69c41ab6d6455f4414e118edf2d09f0ecc4847447f68d3ed338769c5aefd44b34f04be7085127357c8c858a2ead7ac867b26e0f135392e241d17cf86acaf1dbd4576ff01c1844da852579df1067822a6fc827c63ad798f9f5f54c458d6263f70a814eee7cc44d029e","0x0f656deb14f9666d3fdf9dff7c1c430c0570223f10900c89c94a79324ad24085":"0x2de2f2a57554336b6afc9dbc96a976f50b7df217e8267d3d3665ed8d8d624294201755a409bc5dd38144ab79d97c32546f0a7f507645573eda5423e09f0cc7ee71914c1b9640a35e17cf20df8713e0a10cddb8074ceab72734a01121427bcc2b226454bf23407c2a9cec358c0555338035c87fe2bb9de78ed3fffb5a42c827ad","0xf9e8d636bffcc7440c587c88334dafe786b9d3762f129ff56f6e2323fb45d2e6":"0x50ae90c35e2d8f1af6a938b2b17ed081983f1fb40c7542fcb4a89856a4d3d7332d9680b5c4dcbdb8700d52dbf3478c5883e83fe53db864c52682616be8be147f70ce59e05b9aadaf44f9eaeaf4a25b410b399923b6f89c4b32643f467b119ec8217a3ae52ebbe4b4c1fb50989e15f53eca3f0e5231adfe179447d3ff8e034300"}}`
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
	genesisGroupStr := `{"GroupInfo":{"GroupID":"0xbafd7806e7e8eedde0ba38cf268f4e2accee582a92557deb8488e134d2f74dfd","GroupPK":"0x1ae5ce35b47114baae9724f3c4eb7ecf08bdbc88784db6e3dc2edb25d1784e234da685a9d6cc57367bdacaab2b36ef81d2419f49fa4640b4c6cc83929a48154c7da40a074c4acf76831f2e14db48e6907af5410b21e3be76cc2c2412fb71fdb017af2316b4876c65cbbed0defaa4e8b401612a8eff0d2c3c334965cb36a26daa","GroupInitInfo":{"GroupHeader":{"Hash":"0x0aaea93ad8571f27f55d7217dfa7d7bfcf52f69447f6cedbb45e63cf20a076c6","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-06-01T15:30:19.953384+08:00","MemberRoot":"0x2b7e09ea51f4ff99054f3e710e27b6f3fe34de9f1c836fef2f920863c4633914","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0xf9e8d636bffcc7440c587c88334dafe786b9d3762f129ff56f6e2323fb45d2e6","0x0f656deb14f9666d3fdf9dff7c1c430c0570223f10900c89c94a79324ad24085","0x05bcd6acb406e04310c093b54b399052a68eb1f76d4a36f6db1b3402b6e43a19"]},"MemberIndexMap":{"0x05bcd6acb406e04310c093b54b399052a68eb1f76d4a36f6db1b3402b6e43a19":2,"0x0f656deb14f9666d3fdf9dff7c1c430c0570223f10900c89c94a79324ad24085":1,"0xf9e8d636bffcc7440c587c88334dafe786b9d3762f129ff56f6e2323fb45d2e6":0},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["2BeLECHemK15iWzFr6efIX7zw3eIdSSYGIgAAKH52eA=","MwXBg415Y39ZsMij6Onn7wic+FOYJGxAQVFYB7wIzP0=","Q6YjBcekiqoEsJL6sSHWzgi0TWcCgvG28j8jCtGe778="],"Pubkeys":["0x21920596e1e44f95b877290a78a9a704b104d9750b0c5c071771e0957589b7e16468940bfd3c7e85460204724bee82d3f195c59fb6baac4984e2ad2a1b9df06727ebbb02366fe1603a7f290a1bc628fd0c06b2100b1f6741f28d3b863b88503407c9da3bd482b799f9bf210fca540a6ba0cbd640813b4439c2a3bbe12590f718","0x3d6d2188c8e56b51bebac0dac3ce35f4bcbeb6057bc694652a1d396bedd811602fa72b4d4717bc43d67ebcb0ca5e0fc60951b631394350d3297ce263a30065b527c87204a377031c0c7956601780a9167806a48e85250346a9b8d127919187f22b2aab58b98fe63659edccd9af9445fe3806b3b49e8f957f1b7a776f53a34ddf","0x8d1cadd3fbed6458c35c481a4e95b364aaf6d67ebad7adb1a98ccb1c4666274b4b80dfaaaf8d01083be3c4ae9e3d33eb32bc9b3f6e62f8ac0b445c20850dc633511062b5de54928cb4d89263e75824ac3057ab296d56cbac4e59820e0f687934116f7d819235e53fb9e213a4261c5cdb426c0b9b1d4068195d45a75a0a95965c"]}`
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
