package logical

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
	Group StaticGroupInfo
	VrfPK []vrf.VRFPublicKey
	Pks   []groupsig.Pubkey
}

var genesisGroupInfo []*genesisGroup

func GetGenesisGroupInfo() []*genesisGroup {
	if genesisGroupInfo == nil {
		f := common.GlobalConf.GetSectionManager("consensus").GetString("genesis_sgi_conf", "genesis_sgi.config")
		if nil != common.DefaultLogger {
			common.DefaultLogger.Debugf("generate genesis info %s", f)
		}
		genesisGroupInfo = genGenesisStaticGroupInfo(f)
	}
	return genesisGroupInfo
}

func GenerateGenesis() []*types.GenesisInfo {
	genesisGroups := GetGenesisGroupInfo()
	var genesisInfos = make([]*types.GenesisInfo, 0)
	for _, genesis := range genesisGroups {
		sgi := &genesis.Group
		coreGroup := ConvertStaticGroup2CoreGroup(sgi)
		vrfPKs := make([][]byte, sgi.GetMemberCount())
		pks := make([][]byte, sgi.GetMemberCount())

		for i, vpk := range genesis.VrfPK {
			vrfPKs[i] = vpk
		}
		for i, vpk := range genesis.Pks {
			pks[i] = vpk.Serialize()
		}
		genesisGroupInfo := &types.GenesisInfo{Group: *coreGroup, VrfPKs: vrfPKs, Pks: pks,}
		genesisInfos = append(genesisInfos, genesisGroupInfo)
	}
	return genesisInfos
}

//生成创世组成员信息
func (p *Processor) BeginGenesisGroupMember() {

	genesisGroups := GetGenesisGroupInfo()
	for _, genesis := range genesisGroups {
		if !genesis.Group.MemExist(p.GetMinerID()) {
			continue
		}
		sgi := &genesis.Group

		jg := p.belongGroups.getJoinedGroup(sgi.GroupID)
		if jg == nil {
			time.Sleep(time.Second * 1)
			panic("genesisMember find join_group fail")
		}
		p.joinGroup(jg)
	}

}

func generateGenesisGroupHeader(memIds []groupsig.ID) *types.GroupHeader {
	//parentId := "0x3f36baa255bb4df4f07144f8ac366629ec990888677db80a03612212a0a9a3f7"
	//var id groupsig.ID
	//id.SetHexString(parentId)
	//idByte := id.Serialize()
	gh := &types.GroupHeader{
		Name:          "GX genesis group",
		Authority:     777,
		BeginTime:     time.Now(),
		CreateHeight:  0,
		ReadyHeight:   1,
		WorkHeight:    0,
		DismissHeight: common.MaxUint64,
		MemberRoot:    model.GenMemberRootByIds(memIds),
		Extends:       "",
		//Parent:        idByte,
		//PreGroup:      idByte,
	}

	gh.Hash = gh.GenHash()
	return gh
}

func genGenesisStaticGroupInfo(f string) []*genesisGroup {
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
		group := genesis.Group
		group.buildMemberIndex()
		groups = append(groups, genesis)
	}

	return groups
}

func readGenesisJoinedGroup(file string, sgi *StaticGroupInfo) *JoinedGroup {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic("read genesis joinedGroup file failed!err=" + err.Error())
	}
	var group = new(JoinedGroup)
	err = json.Unmarshal(data, group)
	if err != nil {
		panic(err)
	}
	group.GroupPK = sgi.GroupPK
	group.GroupID = sgi.GroupID
	return group
}
