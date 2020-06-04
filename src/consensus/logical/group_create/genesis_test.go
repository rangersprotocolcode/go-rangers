package group_create

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGenGenesisGroupInfo(t *testing.T) {
	genesisGroup := genGenesisGroupInfo()
	for i := range genesisGroup {
		group := genesisGroup[i]
		fmt.Println(group.GroupInfo)
	}
}

func TestGenGenesisJoinedGroup(t *testing.T) {
	jgs := genGenesisJoinedGroup()
	data, _ := json.Marshal(jgs)
	fmt.Println(string(data))
}
