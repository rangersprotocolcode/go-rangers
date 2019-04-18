package network

import (
	"x/src/common"
	"x/src/middleware/log"
	"net/url"
	"github.com/gorilla/websocket"
	"sync"
)

const (
	gateAddr           = "192.168.3.64"
	protocolHeaderSize = 28
)

var Logger log.Logger

func InitNetwork(selfMinerId string, consensusHandler MsgHandler) {
	Logger = log.GetLoggerByIndex(log.P2PLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	url := url.URL{Scheme: "ws", Host: gateAddr, Path: "/service"}
	Logger.Debugf("connecting to %s", url.String())

	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		panic("Dial to" + url.String() + " err:" + err.Error())
	}
	Server = server{conn: conn, consensusHandler: consensusHandler, lock: sync.RWMutex{}}
	go Server.receiveMessage()

	getNetMemberInfo("")
	joinGroup(selfMinerId)
}

func joinGroup(selfMinerId string) {
	var groupId = ""
	for _, group := range netMemberInfo.VerifyGroupList {
		for _, member := range group.Members {
			if selfMinerId == member {
				groupId = group.GroupId
				break
			}
		}
	}
	if groupId == "" {
		return
	}
	Server.joinGroup(groupId)
}
