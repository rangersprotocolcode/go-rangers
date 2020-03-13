package cli

import (
	"bytes"
	"encoding/json"
	"log"
	"sync"
	"x/src/common"
	"x/src/consensus/base"
	"x/src/consensus/vrf"
	"x/src/core"
	"x/src/middleware/types"
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
	privKeyStr, walletAddress = priv.GetHexString(), address.GetHexString()

	var miner types.Miner
	miner.Id = address.Bytes()
	miner.PublicKey = pub.ToBytes()

	secretSeed := base.RandFromBytes(address.Bytes())
	vrfPK, _, _ := vrf.VRFGenerateKey(bytes.NewReader(secretSeed.Bytes()))
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
