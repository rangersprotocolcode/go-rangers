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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"
	"time"
)

const minerApplyInfoFile = "batch_miner_apply_info.txt"
const testnetMinerApplyInfoFile = "batch_miner_apply_info_testnet.txt"

type piece struct {
	id        string
	minerInfo types.Miner
}

func TestBatchMinerApply(t *testing.T) {
	pieceList := parseFile()
	txList := make([]string, 0)

	for i := 0; i < len(pieceList); i++ {
		//genesis miner
		//piece := pieceList[i]
		//if i < 4 {
		//	if i == 3 {
		//		piece.minerInfo.Type = common.MinerTypeProposer
		//		piece.minerInfo.Stake = 5000000
		//	} else {
		//		piece.minerInfo.Type = common.MinerTypeValidator
		//		piece.minerInfo.Stake = 300000
		//	}
		//	minerApplyData, err := json.Marshal(piece.minerInfo)
		//	if err != nil {
		//		fmt.Printf("marshal miner info error:%v\n", err)
		//	}
		//	txStr := genMinerApplyTx(piece.id, string(minerApplyData))
		//	txList = append(txList, txStr)
		//}

		if i < 4 {
			continue
		}
		piece := pieceList[i]
		if i < 7 {
			piece.minerInfo.Type = common.MinerTypeProposer
			piece.minerInfo.Stake = 5000
		} else {
			piece.minerInfo.Type = common.MinerTypeValidator
			piece.minerInfo.Stake = 5000
		}
		minerApplyData, err := json.Marshal(piece.minerInfo)
		if err != nil {
			fmt.Printf("marshal miner info error:%v\n", err)
		}
		txStr := genMinerApplyTx(string(minerApplyData))
		txList = append(txList, txStr)
	}

	sendTxToGate(txList)
}

func TestBatchMinerAdd(t *testing.T) {
	pieceList := parseFile()
	txList := make([]string, 0)

	for i := 0; i < len(pieceList); i++ {
		//genesis miner
		//piece := pieceList[i]
		//if i < 4 {
		//	if i == 3 {
		//		piece.minerInfo.Type = common.MinerTypeProposer
		//		piece.minerInfo.Stake = 5000000
		//	} else {
		//		piece.minerInfo.Type = common.MinerTypeValidator
		//		piece.minerInfo.Stake = 300000
		//	}
		//	minerApplyData, err := json.Marshal(piece.minerInfo)
		//	if err != nil {
		//		fmt.Printf("marshal miner info error:%v\n", err)
		//	}
		//	txStr := genMinerAddTx(piece.id, string(minerApplyData))
		//	txList = append(txList, txStr)
		//}

		if i < 4 {
			continue
		}
		piece := pieceList[i]
		if i < 9 {
			//if i < 7 {
			piece.minerInfo.Type = common.MinerTypeProposer
			piece.minerInfo.Stake = 1000
		} else {
			piece.minerInfo.Type = common.MinerTypeValidator
			piece.minerInfo.Stake = 1000
		}
		minerApplyData, err := json.Marshal(piece.minerInfo)
		if err != nil {
			fmt.Printf("marshal miner info error:%v\n", err)
		}
		txStr := genMinerAddTx(string(minerApplyData))
		txList = append(txList, txStr)
	}

	sendTxToGate(txList)
}

type MinerRefundData struct {
	Amount  string
	MinerId string
}

func TestMinerRefund(t *testing.T) {
	privateKeyStr := "0x047710ad0d7e786dcf5a4b47265ac3d8f5585e88f01dd7156dd464e834fa02096d0d6c1fb1c3010d460dff41ccdfb069d771f96eb71324089d1dfc2c882ed6851f9d628ff7ac459b53dffc85f21ce65f0d03b830b9bf0ed24e30027d67e2c953e2"
	privateKey := common.BytesToSecKey(common.FromHex(privateKeyStr))
	source := privateKey.GetPubKey().GetAddress().String()
	txList := make([]string, 0)
	//miner 15
	//source := "0xaf7ec99107ab9920b8ddc00d9ae2bdcc4a4fcd63"
	tx := types.Transaction{Type: types.TransactionTypeMinerRefund, Source: source, Time: time.Now().String(),ChainId: "8888"}
	str :=`{"amount":"100","minerId":"0xdd50117c639e6134598755b55d8d9fb8b76b2e635880e5853549e62574ecd686"}`
	var refund MinerRefundData
	json.Unmarshal(utility.StrToBytes(str),&refund)

	amount := MinerRefundData{Amount: "100", MinerId: "0xdd50117c639e6134598755b55d8d9fb8b76b2e635880e5853549e62574ecd686"}
	data, _ := json.Marshal(amount)
	tx.Data = utility.BytesToStr(data)
	tx.Hash = tx.GenHash()

	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	txStr := tx.ToTxJson().ToString()
	fmt.Printf("%s\n\n", txStr)

	txList = append(txList, txStr)

	sendTxToGate(txList)
}

func TestMinerAccount(t *testing.T) {
	privateKeyStr := "0x04ebd3320f9f6cda80ee8dbeb0b926a91e099c343909ab06f56c5e6a775bc695e4221399406956e9733154fcc9332bce0221f41ed4d5869bed808fdd1d4bf5fb752669c65ab2e4c800b17ca5259eb2d6736aa4eca1e95d9a0397e122ebda96de00"
	privateKey := common.BytesToSecKey(common.FromHex(privateKeyStr))

	source := privateKey.GetPubKey().GetAddress().String()
	tx := types.Transaction{Type: types.TransactionTypeMinerChangeAccount, Source: source, Time: time.Now().String(), ChainId: "8888"}
	var miner types.Miner
	miner.Id = common.FromHex("0x5fdcc98ef4dced468dd26c766dccad27737bace6edd770dab9311360207cc9b5")
	miner.Account = common.FromHex("0x38780174572fb5b4735df1b7c69aee77ff6e9f49")
	data, _ := json.Marshal(miner)
	tx.Data = string(data)
	tx.Hash = tx.GenHash()

	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
	fmt.Println(source)
	fmt.Println(privateKey.GetHexString())
	//sendTxToGate([]string{tx.ToTxJson().ToString()})
}
func parseFile() []piece {
	bytes, err := ioutil.ReadFile(minerApplyInfoFile)
	//bytes, err := ioutil.ReadFile(testnetMinerApplyInfoFile)
	if err != nil {
		panic("read account info  file error:" + err.Error())
	}
	minerInfoSummary := string(bytes)
	records := strings.Split(minerInfoSummary, "\n")

	result := make([]piece, 0)
	for _, record := range records {
		if record != "" {
			index := strings.Index(record, "Miner apply info:")
			record = string([]byte(record)[index+17:])
			elements := strings.Split(record, "|")

			piece := piece{}
			piece.id = elements[0]

			minerInfo := types.Miner{}
			err := json.Unmarshal([]byte(elements[1]), &minerInfo)
			if err != nil {
				fmt.Printf("marshal miner info error:%v\n", err)
			}
			piece.minerInfo = minerInfo
			result = append(result, piece)
		}
	}
	return result
}

func genMinerApplyTx(data string) string {
	privateKeyStr := "0x040a0c4baa2e0b927a2b1f6f93b317c320d4aa3a5b54c0a83f5872c23155dcf1455fb015a7699d4ef8491cc4c7a770e580ab1362a0e3af9f784dd2485cfc9ba7c1e7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"
	privateKey := common.BytesToSecKey(common.FromHex(privateKeyStr))

	source := privateKey.GetPubKey().GetAddress().String()
	tx := types.Transaction{Type: types.TransactionTypeMinerApply, Source: source, Time: time.Now().String(), ChainId: "9500"}

	tx.Data = string(data)
	tx.Hash = tx.GenHash()

	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
	return tx.ToTxJson().ToString()
}

func genMinerAddTx(data string) string {
	privateKeyStr := "0x040a0c4baa2e0b927a2b1f6f93b317c320d4aa3a5b54c0a83f5872c23155dcf1455fb015a7699d4ef8491cc4c7a770e580ab1362a0e3af9f784dd2485cfc9ba7c1e7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"
	privateKey := common.BytesToSecKey(common.FromHex(privateKeyStr))

	source := privateKey.GetPubKey().GetAddress().String()

	//source := "0x38780174572fb5b4735df1b7c69aee77ff6e9f49"
	tx := types.Transaction{Type: types.TransactionTypeMinerAdd, Source: source, Time: time.Now().String(),ChainId: "8888"}

	tx.Data = string(data)
	tx.Hash = tx.GenHash()

	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
	return tx.ToTxJson().ToString()
}

func sendTxToGate(txList []string) {
	d := websocket.Dialer{ReadBufferSize: 1024 * 1024 * 16, WriteBufferSize: 1024 * 1024 * 16}
	url := url.URL{Scheme: "wss", Host: "mainnet.rangersprotocol.com/pubhub", Path: "/api/writer"}
	//url := url.URL{Scheme: "ws", Host: "161.117.252.255", Path: "/api/writer/1"}

	conn, _, err := d.Dial("wss://mainnet.rangersprotocol.com/pubhub/api/writer", nil)
	if err != nil {
		panic("Dial to" + url.String() + " err:" + err.Error())
	}

	for _, tx := range txList {
		error1 := conn.WriteMessage(websocket.TextMessage, []byte(tx))
		if error1 != nil {
			panic("WriteMessage" + url.String() + " err:" + err.Error())
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func TestMinerRefundBase64(t *testing.T) {
	s, err := base64.StdEncoding.DecodeString("JevvUFyYiszl6wg3fPRA1zUBo8TKiphhjJy7Hy0nfcU=")
	if nil != err {
		t.Fatal(err)
	}

	fmt.Println(common.ToHex(s))
}

func TestMinerRefundHex(t *testing.T) {
	s := common.FromHex("0x2a70a4483ad6690a4c75548a9741c57f27d243f4c96a67d43c2c3e111ce72fe9")
	fmt.Println(base64.StdEncoding.EncodeToString(s))
}
