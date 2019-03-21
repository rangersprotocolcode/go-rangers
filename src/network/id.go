package network

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type MemberInfo struct {
	ProposerList    []Id        `yaml:"proposerList"`
	VerifyGroupList []GroupInfo `yaml:"verifyGroupList"`
}

type GroupInfo struct {
	GroupId string `yaml:"groupId"`
	Members []Id   `yaml:"members"`
}

type Id struct {
	MinerId string `yaml:"minerId"`
	NetId   string `yaml:"netId"`
}

const defaultMemberInfoPath = "member_info.yaml"

var netMemberInfo MemberInfo
//var proposerList = []string{"0xa7a4b347a626d2353050328763f74cd85f83173d6886b24501c39b5c456446d2", "0xf664571c43c13c49ccf032b9d823bedc9f62e5a5858a80d42cd5ee6d897237f0", "0x86805452068d92d647ae9b7a048a09515c875d71eb3d0d3d788c3f3334aaf124"}
//
//var verifyGroupList = []string{"0x015fc80b99d8904205b768e04ccea60b67756fd8176ce27e95f5db1da9e57735"}
//var verifyGroupsInfo = map[string][]string{"0x015fc80b99d8904205b768e04ccea60b67756fd8176ce27e95f5db1da9e57735": {"0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019", "0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76", "0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02"}}

var net2MinerIdMap map[string]string

var miner2NetIdMap map[string]string

func initNetMembers(path string) {
	if path == "" {
		path = defaultMemberInfoPath
	}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic("Init net member error:" + err.Error())
	}
	err = yaml.UnmarshalStrict(bytes, &netMemberInfo)
	if err != nil {
		panic("Unmarshal member info error:" + err.Error())
	}
	netMemberInfo.dump()

	miner2NetIdMap = make(map[string]string)
	net2MinerIdMap = make(map[string]string)

	for _, proposer := range netMemberInfo.ProposerList {
		miner2NetIdMap[proposer.MinerId] = proposer.NetId
		net2MinerIdMap[proposer.NetId] = proposer.MinerId
	}
	for _, group := range netMemberInfo.VerifyGroupList {
		for _, verifier := range group.Members {
			miner2NetIdMap[verifier.MinerId] = verifier.NetId
			net2MinerIdMap[verifier.NetId] = verifier.MinerId
		}
	}
}

func getNetId(minerId string) string {
	netId, found := miner2NetIdMap[minerId]
	if !found {
		Logger.Errorf("Unknown miner id:%s", minerId)
		return ""
	}
	return netId
}

func getMinerId(netId string) string {
	minerId, found := net2MinerIdMap[netId]
	if !found {
		Logger.Errorf("Unknown net id:%s", minerId)
		return ""
	}
	return minerId
}

func (m MemberInfo) dump() {
	Logger.Debugf("Member info:")
	Logger.Debugf("Proposer list:")
	for _, proposer := range m.ProposerList {
		Logger.Debugf("  %s [%s]", proposer.MinerId, proposer.NetId)
	}
	for _, group := range m.VerifyGroupList {
		Logger.Debugf("Group id:%s", group.GroupId)
		Logger.Debugf("Member:")
		for _, id := range group.Members {
			Logger.Debugf("  %s[%s]", id.MinerId, id.NetId)
		}
	}
	Logger.Debugf("end\n")
}
