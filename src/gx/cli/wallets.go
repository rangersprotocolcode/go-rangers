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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"log"
	"sync"
)

var walletManager wallets

type wallets []wallet

var mutex sync.Mutex

type wallet struct {
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
}

func (ws *wallets) store() {
	js, err := json.Marshal(ws)
	if err != nil {
		log.Println("store wallets error")
		return
	}
	common.GlobalConf.SetString(Section, "wallets", string(js))
}

func (ws *wallets) deleteWallet(key string) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, v := range *ws {
		if v.Address == key || v.PrivateKey == key {
			*ws = append((*ws)[:i], (*ws)[i+1:]...)
			break
		}
	}
	ws.store()
}

func (ws *wallets) newWallet() (privKeyStr, walletAddress, minerString string) {
	mutex.Lock()
	defer mutex.Unlock()
	priv := common.GenerateKey("")

	return ws.newWalletByPrivateKey(priv.GetHexString())
}

func (ws *wallets) newWalletByPrivateKey(privateKey string) (privKeyStr, walletAddress, minerString string) {
	priv := common.HexStringToSecKey(privateKey)
	pub := priv.GetPubKey()
	address := pub.GetAddress()
	privKeyStr, walletAddress = pub.GetHexString(), address.GetHexString()

	selfMinerInfo := model.NewSelfMinerInfo(*priv)

	var miner types.Miner
	miner.Id = selfMinerInfo.ID.Serialize()
	miner.PublicKey = selfMinerInfo.PubKey.Serialize()
	miner.VrfPublicKey = selfMinerInfo.VrfPK

	minerJson, _ := json.Marshal(miner)
	minerString = string(minerJson)
	return
}

func (ws *wallets) getBalance(addr []byte) string {
	account := common.BytesToAddress(addr)
	balance := core.GetBlockChain().GetBalance(account)
	return utility.BigIntToStr(balance)
}

func newWallets() wallets {
	var ws wallets
	s := common.GlobalConf.GetString(Section, "wallets", "")
	if s == "" {
		return ws
	}
	err := json.Unmarshal([]byte(s), &ws)
	if err != nil {
		log.Println(err)
	}
	return ws
}
