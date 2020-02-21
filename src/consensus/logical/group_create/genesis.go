package group_create

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/consensus/groupsig"
	"time"
	"x/src/consensus/model"
	"x/src/consensus/vrf"
)

type genesisGroup struct {
	groupInfo model.GroupInfo
	vrfPubkey []vrf.VRFPublicKey
	pubkeys   []groupsig.Pubkey
}

var genesisGroupInfo []*genesisGroup

//GenerateGenesis
func GetGenesisInfo() []*types.GenesisInfo {
	genesisGroups := getGenesisGroupInfo()
	var genesisInfos = make([]*types.GenesisInfo, 0)
	for _, genesis := range genesisGroups {
		sgi := &genesis.groupInfo
		coreGroup := convertToGroup(sgi)
		vrfPKs := make([][]byte, sgi.GetMemberCount())
		pks := make([][]byte, sgi.GetMemberCount())

		for i, vpk := range genesis.vrfPubkey {
			vrfPKs[i] = vpk
		}
		for i, vpk := range genesis.pubkeys {
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
		if !genesis.groupInfo.MemExist(p.minerInfo.ID) {
			continue
		}
		sgi := &genesis.groupInfo

		jg := p.joinedGroupStorage.GetJoinedGroupInfo(sgi.GroupID)
		if jg == nil {
			time.Sleep(time.Second * 1)
			panic("genesisMember find join_group fail" + sgi.GroupID.GetHexString())
		}
		p.joinedGroupStorage.JoinGroup(jg, p.minerInfo.ID)
	}

}

func getGenesisGroupInfo() []*genesisGroup {
	if genesisGroupInfo == nil {
		f := common.GlobalConf.GetSectionManager("consensus").GetString("genesis_sgi_conf", "genesis_sgi.config")
		if nil != common.DefaultLogger {
			common.DefaultLogger.Debugf("generate genesis info %s", f)
		}
		genesisGroupInfo = genGenesisGroupInfo(f)
	}
	return genesisGroupInfo
}

//genGenesisStaticGroupInfo
func genGenesisGroupInfo(f string) []*genesisGroup {
	var genesisGroupStr string
	if strings.TrimSpace(f) != "" {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			panic(err)
		}
		genesisGroupStr = string(data)
	}
	//common.DefaultLogger.Errorf("genesisGroupStr:%s", genesisGroupStr)
	splited := strings.Split(genesisGroupStr, "&&")
	var groups = make([]*genesisGroup, 0)
	for _, split := range splited {
		genesis := new(genesisGroup)
		err := json.Unmarshal([]byte(split), genesis)
		if err != nil {
			panic(err)
		}
		group := genesis.groupInfo
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

//func generateGenesisGroupHeader(memIds []groupsig.ID) *types.GroupHeader {
//	//parentId := "0xf2cb0c8e7d1086c8f311a28f51871564c5cc31361e7d1f1498d3306d54fd729b"
//	//var id groupsig.ID
//	//id.SetHexString(parentId)
//	//idByte := id.Serialize()
//	gh := &types.GroupHeader{
//		Name:          "GX genesis group",
//		Authority:     777,
//		BeginTime:     time.Now(),
//		CreateHeight:  0,
//		ReadyHeight:   1,
//		WorkHeight:    0,
//		DismissHeight: common.MaxUint64,
//		MemberRoot:    model.GenGroupMemberRoot(memIds),
//		Extends:       "",
//		//Parent:        idByte,
//		//PreGroup:      idByte,
//	}
//
//	gh.Hash = gh.GenHash()
//	return gh
//}
