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
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/utility"
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
