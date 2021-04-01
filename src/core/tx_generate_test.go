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
	target := "0xe059d17139e2915d270ef8f3eee2f3e1438546ba2f06eb674dda0967846b6951"
	tx := types.Transaction{Type: 2, Source: source, Target: target, Time: utility.GetTime().String()}

	data := `{"id":"4FnRcTnikV0nDvjz7uLz4UOFRrovButnTdoJZ4RraVE=","publicKey":"VUD8/iw8JOtgJvyeBR/WlusU/L7+9jsae4z1s4QYZHN6fGVaqfZOI4LlUhaY5+15M0JwhZL+dCjMx8zJa0q/6IriorePaeCsjGt1lTRXYmhSzCZvebL4NP/oR09zDOTSP384WcyZHsV2MMRI7M+K2L3FO6JLI+u9jIAa3pgHQ5E=","vrfPublicKey":"og9c2j7LEkkQwQpSpjVAQ9jkZa/eriBlNrORJdhMe/8="}`
	var obj = types.Miner{}
	err := json.Unmarshal([]byte(data), &obj)
	if err != nil {
		fmt.Printf("ummarshal error:%v", err)
	}

	obj.Stake = 60000000
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

func TestAddMinerStakeTx(t *testing.T) {
	source := "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"
	target := "0xe059d17139e2915d270ef8f3eee2f3e1438546ba2f06eb674dda0967846b6951"
	tx := types.Transaction{Type: 5, Source: source, Target: target, Time: utility.GetTime().String()}

	data := `{"id":"4FnRcTnikV0nDvjz7uLz4UOFRrovButnTdoJZ4RraVE=","stake":60000000}`
	//applyData, _ := json.Marshal(data)
	//fmt.Printf("data:%v\n",string(applyData))

	tx.Data = data
	tx.Hash = tx.GenHash()

	privateKeyStr := "0x040a0c4baa2e0b927a2b1f6f93b317c320d4aa3a5b54c0a83f5872c23155dcf1455fb015a7699d4ef8491cc4c7a770e580ab1362a0e3af9f784dd2485cfc9ba7c1e7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"
	privateKey := common.HexStringToSecKey(privateKeyStr)
	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

//------------------------------------------------------------------------------------------------
func TestGetBlockNumberTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 602, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"11","hash":""}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetBalanceTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 99, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"11","hash":""}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetStorageTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 609, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"11","hash":"","address":"0x0200000000000000000000000000000000000000","key":"0x51ba50a9b4730aea7ecee86df6536297900f5b77"}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetCodeTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 610, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"","hash":"","address":"0x9ddeb4a196da90824feceeac57e6d5f1d82b2ad7"}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetBlockTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 603, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"","hash":"",address:"0x9ddeb4a196da90824feceeac57e6d5f1d82b2ad7"}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetBlockTransactionCountTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 607, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"","hash":""}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetTransactionTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 605, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"txHash":"0xf58b553d58d2ff88a01bcf936681984802d2006b6512b5eb4e47573c81400926"}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetTransactionFromBlockTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 608, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"","hash":"","index":"0"}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetReceiptTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 606, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"txHash":"0xf58b553d58d2ff88a01bcf936681984802d2006b6512b5eb4e47573c81400926"}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetNonceTx(t *testing.T) {
	source := "0x51ba50a9b4730aea7ecee86df6536297900f5b77"
	tx := types.Transaction{Type: 604, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := `{"height":"","hash":""}`
	tx.Data = string(data)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestTransferTx(t *testing.T) {
	source := "0x38780174572fb5b4735df1b7c69aee77ff6e9f49"
	target := "0x51ba50a9b4730aea7ecee86df6536297900f5b78"
	tx := types.Transaction{Type: 200, Source: source, Target: target, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	data := contractData{GasPrice: "1", GasLimit: "100000", TransferValue: "3", AbiData: "11"}
	dataBytes, _ := json.Marshal(data)
	tx.Data = string(dataBytes)

	tx.Hash = tx.GenHash()

	privateKeyStr := "0x040a0c4baa2e0b927a2b1f6f93b317c320d4aa3a5b54c0a83f5872c23155dcf1455fb015a7699d4ef8491cc4c7a770e580ab1362a0e3af9f784dd2485cfc9ba7c1e7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"
	privateKey := common.HexStringToSecKey(privateKeyStr)
	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

func TestGetPastLogsTx(t *testing.T) {
	source := "0x38780174572fb5b4735df1b7c69aee77ff6e9f49"
	tx := types.Transaction{Type: 611, Source: source, Time: utility.GetTime().String()}
	tx.SocketRequestId = "111"

	topics := [][]string{{"0x3a406d3871dab9676f7dbfa824f81f599698603527e1521006603c9118171e18"}}
	data := QueryLogData{FromBlock: 596173, ToBlock: 596173, Topics: topics}
	dataBytes, err := json.Marshal(&data)
	if err != nil {
		panic(err)
	}
	tx.Data = string(dataBytes)
	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
}

type contractData struct {
	GasPrice string `json:"gasPrice,omitempty"`
	GasLimit string `json:"gasLimit,omitempty"`

	TransferValue string `json:"transferValue,omitempty"`
	AbiData       string `json:"abiData,omitempty"`
}

type QueryLogData struct {
	FromBlock uint64 `json:"fromBlock,omitempty"`
	ToBlock   uint64 `json:"toBlock,omitempty"`

	Address []string   `json:"address,omitempty"`
	Topics  [][]string `json:"topics,omitempty"`
}
