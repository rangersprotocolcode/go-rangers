package cli

import (
	"encoding/json"
	"log"
	"sync"
	"x/src/common"
	"x/src/core"
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
func (ws *wallets) newWallet() (privKeyStr, walletAddress string) {
	mutex.Lock()
	defer mutex.Unlock()
	priv := common.GenerateKey("")
	pub := priv.GetPubKey()
	address := pub.GetAddress()
	privKeyStr, walletAddress = pub.GetHexString(), address.GetHexString()
	// 加入本地钱包
	//*ws = append(*ws, wallet{privKeyStr, walletAddress})
	//ws.store()
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
