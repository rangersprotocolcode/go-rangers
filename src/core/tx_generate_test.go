// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/statemachine"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"testing"
)

func TestStartStateMachineTx(t *testing.T) {
	containerConfig := statemachine.ContainerConfig{Priority: 0, Game: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54",
		Image: "littlebear234/genesis_image:latest", Detached: true, Hostname: "genesis_host_name"}

	port := statemachine.Port{Host: 0, Target: 0}
	ports := statemachine.Ports{port}
	containerConfig.Ports = ports

	containerConfig.DownloadUrl = "littlebear234/genesis_image:latest"
	containerConfig.DownloadProtocol = "pull"

	tx := types.Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "", Type: types.TransactionTypeAddStateMachine, Time: "12121"}
	tx.Data = containerConfig.TOJSONString()

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestProposerApplyTx(t *testing.T) {
	source := "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"
	target := "0x9a5adc4b83ed4272f8af0c46686a938701391ae3576b77891e2ab4e7438f442e"
	tx := types.Transaction{Type: 2, Source: source, Target: target, Time: utility.GetTime().String()}

	data := `{"id":"mlrcS4PtQnL4rwxGaGqThwE5GuNXa3eJHiq050OPRC4=","publicKey":"BOu0RbvBDBlVUySzb+ojoE7BTO67yhYQWdOvqClYG+Qu11SFY79i1lDou9VkPfnpX0KPhlvtpTIIK3IIR2K1meM=","vrfPublicKey":"Dw7zNJeE4wj+diK2c/P+9raL6R72SY1ySbleYVihJtU="}`
	var obj = types.Miner{}
	err := json.Unmarshal([]byte(data), &obj)
	if err != nil {
		fmt.Printf("ummarshal error:%v", err)
	}

	obj.Stake = 2000000
	obj.Type = common.MinerTypeProposer

	applyData, _ := json.Marshal(obj)
	//fmt.Printf("data:%v\n",string(applyData))

	tx.Data = string(applyData)
	tx.Hash = tx.GenHash()

	privateKeyStr := "0x040a0c4baa2e0b927a2b1f6f93b317c320d4aa3a5b54c0a83f5872c23155dcf1455fb015a7699d4ef8491cc4c7a770e580ab1362a0e3af9f784dd2485cfc9ba7c1e7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"
	privateKey := common.HexStringToSecKey(privateKeyStr)
	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestVerifierApplyTx(t *testing.T) {
	source := "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"
	target := "0x4788022fee69b8bf287c0b69b90d40773fb1a3a4251faf3f5c0181cdc3fb78ab"
	tx := types.Transaction{Type: 2, Source: source, Target: target, Time: utility.GetTime().String()}

	data := `{"id":"R4gCL+5puL8ofAtpuQ1Adz+xo6QlH68/XAGBzcP7eKs=","publicKey":"BCOds4TtM4LySN+jiKVUU7T18Yu5KHXKdu+mloMB2H//NNgeREOtzdPI5XxGug+eo+WTAkdIjBhjD7cKOjEaTWo=","vrfPublicKey":"2jtqai9hfhcBNzHrxiYUOisiHdMTKM9Xr5yILDQ9uvs="}`
	var obj = types.Miner{}
	err := json.Unmarshal([]byte(data), &obj)
	if err != nil {
		fmt.Printf("ummarshal error:%v", err)
	}

	obj.Stake = 2000000
	obj.Type = common.MinerTypeValidator

	applyData, _ := json.Marshal(obj)
	//fmt.Printf("data:%v\n",string(applyData))

	tx.Data = string(applyData)
	tx.Hash = tx.GenHash()

	privateKeyStr := "0x040a0c4baa2e0b927a2b1f6f93b317c320d4aa3a5b54c0a83f5872c23155dcf1455fb015a7699d4ef8491cc4c7a770e580ab1362a0e3af9f784dd2485cfc9ba7c1e7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"
	privateKey := common.HexStringToSecKey(privateKeyStr)
	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}
