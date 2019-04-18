package network

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type MemberInfo struct {
	ProposerList    []string    `yaml:"proposerList"`
	VerifyGroupList []GroupInfo `yaml:"verifyGroupList"`
}

type GroupInfo struct {
	GroupId string   `yaml:"groupId"`
	Members []string `yaml:"members"`
}

const defaultMemberInfoPath = "member_info.yaml"

var netMemberInfo MemberInfo

var net2MinerIdMap map[string]string

var miner2NetIdMap map[string]string

func getNetMemberInfo(path string) {
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
}

func (m MemberInfo) dump() {
	Logger.Debugf("Member info:")
	Logger.Debugf("Proposer list:")
	for _, proposer := range m.ProposerList {
		Logger.Debugf("  %s ", proposer)
	}
	for _, group := range m.VerifyGroupList {
		Logger.Debugf("Group id:%s", group.GroupId)
		Logger.Debugf("Member:")
		for _, id := range group.Members {
			Logger.Debugf("  %s", id)
		}
	}
	Logger.Debugf("end\n")
}
