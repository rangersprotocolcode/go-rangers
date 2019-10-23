package network

import (
	"x/src/common"
	"x/src/middleware/log"
	"net/url"
	"github.com/gorilla/websocket"
)

const (
	protocolHeaderSize = 28
	channelSize        = 100
	bufferSize         = 1024 * 1024 * 32

	gateAddrProduction = "47.96.99.105:8848"
	gateAddrDaily      = "47.96.99.105:8848"
)

var gateAddr string
var Logger log.Logger

func InitNetwork(selfMinerId string, consensusHandler MsgHandler, env string, gate string) {
	Logger = log.GetLoggerByIndex(log.P2PLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	if len(gate) != 0 {
		gateAddr = gate
	} else {
		if env == "production" {
			gateAddr = gateAddrProduction
		} else {
			gateAddr = gateAddrDaily
		}
	}
	url := url.URL{Scheme: "ws", Host: gateAddr, Path: "/srv/worker"}
	Logger.Debugf("connecting to %s", url.String())

	conn, _, err := newDialer().Dial(url.String(), nil)
	if err != nil {
		panic("Dial to" + url.String() + " err:" + err.Error())
	}
	Server = server{conn: conn, consensusHandler: consensusHandler, sendChan: make(chan []byte, channelSize), rcvChan: make(chan []byte, channelSize)}
	go Server.receiveMessage()
	go Server.loop()

	getNetMemberInfo("")
	joinGroup(selfMinerId)
}

func newDialer() *websocket.Dialer {
	d := websocket.Dialer{ReadBufferSize: bufferSize, WriteBufferSize: bufferSize,}
	return &d
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
