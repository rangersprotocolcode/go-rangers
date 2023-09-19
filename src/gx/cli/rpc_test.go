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

package cli

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"
)

func TestRPC(t *testing.T) {
	gx := NewGX()
	common.Init(0, "rp.ini", "dev")
	walletManager = newWallets()
	gx.initMiner("dev", "", "", "")

	host := "0.0.0.0"
	var port uint = 8989
	if err := StartRPC(host, port, ""); err != nil {
		panic(err)
	}

	tests := []struct {
		method string
		params []interface{}
	}{
		{"Rocket_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"50\",\"assets\":{\"1\":\"dj\"}}]", 1}},
		{"Rocket_getBalance", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_getAsset", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "1"}},
		{"Rocket_getAllAssets", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"-2.25\",\"assets\":{\"1\":\"dj11\",\"2\":\"yyyy\"}}]", 2}},
		{"Rocket_getAsset", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "1"}},
		{"Rocket_getAllAssets", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_getBalance", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"2.25\",\"assets\":{\"1\":\"\",\"2\":\"yyyy\"}}]", 3}},
		{"Rocket_getAsset", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "1"}},
		{"Rocket_getAllAssets", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_getBalance", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_notify", []interface{}{"tuntun", "a19d069d48d2e9392ec2bb41ecab0a72119d633b", "notify one"}},
		//{"Rocket_notifyGroup", []interface{}{"tuntun", "groupA","notify groupA"}},
		//{"Rocket_notifyBroadcast", []interface{}{"tuntun", "notify all"}},
		//{"GTAS_blockHeight", nil},
		//{"GTAS_getWallets", nil},
	}
	for _, test := range tests {
		res, err := rpcPost(host, port, test.method, test.params...)
		if err != nil {
			t.Errorf("%s failed: %v", test.method, err)
			continue
		}
		if res.Error != nil {
			t.Errorf("%s failed: %v", test.method, res.Error.Message)
			continue
		}
		if nil != res && nil != res.Result {
			data, _ := json.Marshal(res.Result.Data)
			log.Printf("%s response data: %s", test.method, data)
		}

	}

	time.Sleep(10000 * time.Second)
}

func TestStrToFloat(t *testing.T) {
	var a = "11.23456"
	b, _ := strconv.ParseFloat(a, 64)
	fmt.Printf("float :%v\n", b)

	c := strconv.FormatFloat(b, 'E', -1, 64)
	fmt.Printf("string :%v\n", c)

	var m = "2.28E-5"
	n, _ := strconv.ParseFloat(m, 64)
	fmt.Printf("float :%v\n", n)
}

func TestJSONString(t *testing.T) {
	m := make(map[string]string, 0)

	m["a"] = "b"
	//m["aa"] = "bc"
	data, _ := json.Marshal(m)
	log.Printf("response data: %s", data)

	mm := make(map[string]string, 0)
	json.Unmarshal(data, &mm)

	log.Printf("mm response data: %s", mm)

}

func TestSlice(t *testing.T) {
	a := []int{0, 1, 2, 3, 4}
	//删除第i个元素
	i := 4
	a = append(a[:i], a[i+1:]...)
	fmt.Println(a)
}

func TestNotifyId(t *testing.T) {
	gameId := "a"
	userId := "1"

	data := []byte(gameId)
	if 0 != len(userId) {
		data = append(data, []byte(userId)...)
	}

	md5Result := md5.Sum(data)
	idBytes := md5Result[4:12]

	id := uint64(binary.BigEndian.Uint64(idBytes))

	fmt.Println(id)
}

func TestGtasAPI_NewWallet(t *testing.T) {
	priv := common.HexStringToSecKey("0x04a75d51da3bf3e79da72fd778547613d4fca6fe0a99de24d364d9bf3151c18a37c2063ad91995ca41eb396bdc12ae03cdd1e7ab3b5cf6c82c2310438cd8a61d0d703f96985effb724e9af17031ab9456259860623b859f42ec79894e925a43c28")
	pub := priv.GetPubKey()
	address := pub.GetAddress()
	privKeyStr, walletAddress := pub.GetHexString(), address.GetHexString()

	fmt.Println(privKeyStr)
	fmt.Println(walletAddress)

	// 加入本地钱包
	//*ws = append(*ws, wallet{privKeyStr, walletAddress})
	//ws.store()

	var miner types.Miner
	miner.Id = address.Bytes()
	miner.PublicKey = pub.ToBytes()

	secretSeed := base.RandFromBytes(address.Bytes())
	vrfPK, vrfSK, _ := vrf.VRFGenerateKey(bytes.NewReader(secretSeed.Bytes()))
	miner.VrfPublicKey = vrfPK.GetBytes()

	fmt.Println(vrfPK.GetHexString())
	fmt.Println(vrfSK.GetHexString())
}
