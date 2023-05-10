package cli

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/eth_rpc"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/utility"
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
		res := handler.ProcessSingleRequest(message)
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
		res := handler.ProcessBatchRequest(messages)
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
