package cli

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"io/ioutil"
	"net/http"
	"strings"
)

func NewSelfServer(privateKey string) *SelfServer {
	server := SelfServer{privateKey: privateKey}
	return &server
}

type SelfServer struct {
	privateKey string
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
	case "/api/self":
		server.processSelf(w, r, body)
		break
	default:
		common.DefaultLogger.Errorf("wrong: %s, %s", r.Method, r.URL.Path)
	}
}

func (server *SelfServer) processSelf(w http.ResponseWriter, r *http.Request, body []byte) {
	_, _, self := walletManager.newWalletByPrivateKey(server.privateKey)
	w.Write(utility.StrToBytes(self))
}
