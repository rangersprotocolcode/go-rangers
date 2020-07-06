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

package cli

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/core"
	"com.tuntun.rocket/node/src/middleware/types"
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

//存储钱包账户
func (ws *wallets) store() {
	js, err := json.Marshal(ws)
	if err != nil {
		log.Println("store wallets error")
		// TODO 输出log
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

// newWallet 新建钱包并存储到config文件中
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

	var miner types.Miner
	miner.Id = address.Bytes()

	secretSeed := base.RandFromBytes(address.Bytes())
	minerSecKey := *groupsig.NewSeckeyFromRand(secretSeed)
	minerPubKey := *groupsig.GeneratePubkey(minerSecKey)
	vrfPK, _, _ := vrf.VRFGenerateKey(bytes.NewReader(secretSeed.Bytes()))

	miner.PublicKey = minerPubKey.Serialize()
	miner.VrfPublicKey = vrfPK

	minerJson, _ := json.Marshal(miner)
	minerString = string(minerJson)
	return
}

func (ws *wallets) getBalance(account string) (uint64, error) {
	if account == "" && len(walletManager) > 0 {
		account = walletManager[0].Address
	}
	balance := core.GetBlockChain().GetBalance(common.HexToAddress(account))

	return balance.Uint64(), nil
}

func newWallets() wallets {
	var ws wallets
	s := common.GlobalConf.GetString(Section, "wallets", "")
	if s == "" {
		return ws
	}
	err := json.Unmarshal([]byte(s), &ws)
	if err != nil {
		// TODO 输出log
		log.Println(err)
	}
	return ws
}
