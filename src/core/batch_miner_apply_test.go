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

type piece struct {
	id        string
	minerInfo types.Miner
}

func TestBatchMinerApply(t *testing.T) {
	pieceList := parseFile()
	txList := make([]string, 0)

	for i := 0; i < len(pieceList); i++ {
		if i < 4 {
			continue
		}
		piece := pieceList[i]
		if i < 10 {
			piece.minerInfo.Type = common.MinerTypeProposer
			piece.minerInfo.Stake = 5000000
		} else {
			piece.minerInfo.Type = common.MinerTypeValidator
			piece.minerInfo.Stake = 300000
		}
		minerApplyData, err := json.Marshal(piece.minerInfo)
		if err != nil {
			fmt.Printf("marshal miner info error:%v\n", err)
		}
		txStr := genMinerApplyTx(piece.id, string(minerApplyData))
		txList = append(txList, txStr)
	}

	sendTxToGate(txList)
}

func parseFile() []piece {
	bytes, err := ioutil.ReadFile(minerApplyInfoFile)
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

func genMinerApplyTx(target string, data string) string {
	source := "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"
	tx := types.Transaction{Type: 2, Source: source, Target: target, Time: time.Now().String()}

	tx.Data = string(data)
	tx.Hash = tx.GenHash()

	privateKeyStr := "0x040a0c4baa2e0b927a2b1f6f93b317c320d4aa3a5b54c0a83f5872c23155dcf1455fb015a7699d4ef8491cc4c7a770e580ab1362a0e3af9f784dd2485cfc9ba7c1e7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"
	privateKey := common.HexStringToSecKey(privateKeyStr)
	sign := privateKey.Sign(tx.Hash.Bytes())
	tx.Sign = &sign

	fmt.Printf("%s\n\n", tx.ToTxJson().ToString())
	return tx.ToTxJson().ToString()
}

func sendTxToGate(txList []string) {
	d := websocket.Dialer{ReadBufferSize: 1024 * 1024 * 16, WriteBufferSize: 1024 * 1024 * 16}
	url := url.URL{Scheme: "ws", Host: "gate.tuntunhz.com:10000", Path: "/api/writer/1"}

	conn, _, err := d.Dial(url.String(), nil)
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
