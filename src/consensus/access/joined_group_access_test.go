// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package access

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/db"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestGetJoinedGroupInfo(t *testing.T) {
	genesisGroups := genGenesisGroupInfo()
	genesis := genesisGroups[0]
	sgi := &genesis.GroupInfo
	gid := sgi.GroupID

	jg := load(gid, "0x445173ab39681491f688e8b5b11f3f51041ce0d05b5ddd75ccc86f4c3343a418")
	data, _ := json.Marshal(jg)
	fmt.Println(string(data))
	fmt.Println(jg.SignSecKey.GetHexString())
}

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

type genesisGroup struct {
	GroupInfo model.GroupInfo
	VrfPubkey []vrf.VRFPublicKey
	Pubkeys   []groupsig.Pubkey
}

func load(gid groupsig.ID, privateKeyStr string) *model.JoinedGroupInfo {
	db, err := db.NewLDBDatabase("/Users/daijia/go/src/x/deploy/daily/groupstore3", 1, 1)
	if err != nil {
		panic("newLDBDatabase fail, file=" + "" + "err=" + err.Error())
	}
	defer db.Close()

	joinedGroupInfo := new(model.JoinedGroupInfo)
	joinedGroupInfo.MemberSignPubkeyMap = make(map[string]groupsig.Pubkey, 0)
	// Load signature private key
	bs, err := db.Get(signKeySuffix(gid))
	if err != nil {
		logger.Errorf("get signKey fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return nil
	}

	sk := common.HexStringToSecKey(privateKeyStr)
	minerInfo := model.NewSelfMinerInfo(*sk)
	privateKey := getEncryptPrivateKey(minerInfo)
	m, err := privateKey.Decrypt(rand.Reader, bs)
	if err != nil {
		logger.Errorf("decrypt signKey fail, err=%v", err.Error())
		return nil
	}
	joinedGroupInfo.SignSecKey.Deserialize(m)

	// Load group information
	infoBytes, err := db.Get(gInfoSuffix(gid))
	if err != nil {
		logger.Errorf("get groupInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return joinedGroupInfo
	}
	if err := json.Unmarshal(infoBytes, joinedGroupInfo); err != nil {
		logger.Errorf("unmarshal groupInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return joinedGroupInfo
	}
	return joinedGroupInfo
}

func getEncryptPrivateKey(mi model.SelfMinerInfo) common.PrivateKey {
	seed := mi.SecKey.GetHexString() + mi.ID.GetHexString()

	encryptPrivateKey := common.GenerateKey(seed)
	return encryptPrivateKey
}
