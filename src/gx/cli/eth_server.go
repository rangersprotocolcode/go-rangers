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
	"com.tuntun.rangers/node/src/eth_rpc"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

func NewETHServer() *ETHServer {
	server := ETHServer{}
	return &server
}

type ETHServer struct {
}

func (server *ETHServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// preflight
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Content-Type", "application/json")
	if 0 == strings.Compare(strings.ToUpper(r.Method), "OPTIONS") {
		w.WriteHeader(200)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		common.DefaultLogger.Errorf("fail to get postdata: %s, %s", r.URL.Path, err.Error())
		return
	}

	common.DefaultLogger.Debugf("receiving: %s, %s, %s from %s", r.Method, r.URL.Path, utility.BytesToStr(body), r.RemoteAddr)
	if r.Method != http.MethodPost {
		common.DefaultLogger.Errorf("wrong: %s, %s", r.Method, r.URL.Path)
		return
	}

	switch strings.ToLower(r.URL.Path) {
	case "/api/jsonrpc":
		server.processJSONRPC(w, r, body)
		break
	default:
		common.DefaultLogger.Errorf("wrong: %s, %s", r.Method, r.URL.Path)
	}
}

func (server *ETHServer) processJSONRPC(w http.ResponseWriter, r *http.Request, body []byte) {
	handler := eth_rpc.GetEthMsgHandler()

	var message notify.ETHRPCPiece
	err := json.Unmarshal(body, &message)
	if err == nil {
		common.DefaultLogger.Debugf("single data: %s, %s", r.URL.Path, utility.BytesToStr(body))
		res := handler.ProcessSingleRequest(message, 0)
		responseJson, err := json.Marshal(res)
		if err != nil {
			common.DefaultLogger.Errorf("fail to process postdata: %s, %s", r.URL.Path, err.Error())
			return
		}

		common.DefaultLogger.Debugf("single data response: %s, %s", r.URL.Path, utility.BytesToStr(responseJson))
		w.Write(responseJson)
		return
	}

	var messages []notify.ETHRPCPiece
	err = json.Unmarshal(body, &messages)
	if err == nil {
		common.DefaultLogger.Debugf("batch data: %s, %s", r.URL.Path, utility.BytesToStr(body))
		res := handler.ProcessBatchRequest(messages, 0)
		responseJson, err := json.Marshal(res)
		if err != nil {
			common.DefaultLogger.Errorf("fail to process batch postdata: %s, %s", r.URL.Path, err.Error())
			return
		}

		common.DefaultLogger.Debugf("batch data response: %s, %s", r.URL.Path, utility.BytesToStr(responseJson))
		w.Write(responseJson)
		return
	}

	common.DefaultLogger.Errorf("wrong post data: %s, %s", r.URL.Path, err.Error())
}
