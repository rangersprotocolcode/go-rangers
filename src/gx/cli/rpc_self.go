package cli

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/core"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func NewSelfServer(privateKey string) *SelfServer {
	server := SelfServer{privateKey: privateKey}

	_, _, self := walletManager.newWalletByPrivateKey(server.privateKey)
	server.minerInfo = self

	var miner types.Miner
	json.Unmarshal(utility.StrToBytes(self), &miner)
	server.id = miner.Id

	return &server
}

type SelfServer struct {
	privateKey string
	minerInfo  string
	id         types.HexBytes
}

func (server *SelfServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		common.DefaultLogger.Errorf("fail to get postdata: %s, %s", r.URL.Path, err.Error())
		return
	}

	common.DefaultLogger.Debugf("receiving: %s, %s, %s from %s", r.Method, r.URL.Path, utility.BytesToStr(body), r.RemoteAddr)

	switch strings.ToLower(r.URL.Path) {
	case "/api/miner":
		server.processSelf(w, r, body)
		break
	case "/api/height":
		server.processHeight(w, r, body)
		break
	case "/api/account":
		server.processAccount(w, r, body)
		break
	default:
		common.DefaultLogger.Errorf("wrong: %s, %s", r.Method, r.URL.Path)
	}
}

// 本地矿工信息
func (server *SelfServer) processSelf(w http.ResponseWriter, r *http.Request, body []byte) {
	w.Write(utility.StrToBytes(server.minerInfo))
}

// 本地块高
func (server *SelfServer) processHeight(w http.ResponseWriter, r *http.Request, body []byte) {
	height := core.GetBlockChain().Height()
	w.Write(utility.StrToBytes(strconv.FormatUint(height, 10)))
}

// 收益账户
func (server *SelfServer) processAccount(w http.ResponseWriter, r *http.Request, body []byte) {
	miner := service.MinerManagerImpl.GetMiner(server.id, nil)
	if nil == miner {
		return
	}

	w.Write(utility.StrToBytes(common.ToHex(miner.Account)))
}
