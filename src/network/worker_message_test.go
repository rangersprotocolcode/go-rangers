package network

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

const (
	msgSizeThreshold = 4 * 1024 * 1024 * 8

	worker1LogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/worker1.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	p2p1LogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/p2p1.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	worker2LogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/worker2.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	p2p2LogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/p2p2.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
)

func TestWorker1(t *testing.T) {
	url := ""
	var worker1 WorkerConn
	p2pLogger = log.GetLoggerByIndex(p2p1LogConfig, strconv.Itoa(common.InstanceIndex))
	bizLogger = log.GetLoggerByIndex(p2p1LogConfig, strconv.Itoa(common.InstanceIndex))
	worker1Logger := log.GetLoggerByIndex(worker1LogConfig, strconv.Itoa(common.InstanceIndex))
	var rcvNonce = 0

	worker1.Init(url, []byte("1"), nil, worker1Logger)
	worker1.doRcv = func(wsHeader wsHeader, body []byte) {
		rcvNonce++
		method := wsHeader.method
		if !bytes.Equal(method, methodCodeSend) && !bytes.Equal(method, methodCodeBroadcast) && !bytes.Equal(method, methodCodeSendToGroup) && !bytes.Equal(method, methodSendToManager) {
			worker1Logger.Error("received wrong method, wsHeader: %v,body:%v", wsHeader, body)
			return
		}

		if bytes.Equal(method, methodSendToManager) {
			body = body[netIdSize:]
		}

		from := strconv.FormatUint(wsHeader.sourceId, 10)
		message, error := unMarshalMessage(body)
		if error != nil {
			worker1Logger.Errorf("Proto unmarshal node message error: %s", error.Error())
			return
		}
		assert.Equal(t, message.Code, uint32(len(message.Body)))
		worker1Logger.Debugf("nonce:%d,rcv from %s,size:%d", rcvNonce, from, message.Code)
		for _, item := range message.Body {
			assert.Equal(t, item, uint8(6))
		}
	}

	for i := 0; ; i++ {
		var msg []byte
		if sendBigMessage() {
			msg = genBigMessage()
			worker1Logger.Debugf("send big msg %d,size:%d,", i, uint32(len(msg)))
		} else {
			msg = genSmallMessage()
			worker1Logger.Debugf("send small msg %d,size:%d,", i, uint32(len(msg)))
		}
		message := Message{Code: uint32(len(msg)), Body: msg}
		worker1.SendToEveryone(message)
		time.Sleep(time.Second * 3)
	}

}

func TestWorker2(t *testing.T) {
	url := ""

	var worker1 WorkerConn
	p2pLogger = log.GetLoggerByIndex(p2p2LogConfig, strconv.Itoa(common.InstanceIndex))
	bizLogger = log.GetLoggerByIndex(p2p2LogConfig, strconv.Itoa(common.InstanceIndex))
	worker2Logger := log.GetLoggerByIndex(worker2LogConfig, strconv.Itoa(common.InstanceIndex))
	var rcvNonce = 0

	worker1.Init(url, []byte("2"), nil, worker2Logger)
	worker1.doRcv = func(wsHeader wsHeader, body []byte) {
		rcvNonce++
		method := wsHeader.method
		if !bytes.Equal(method, methodCodeSend) && !bytes.Equal(method, methodCodeBroadcast) && !bytes.Equal(method, methodCodeSendToGroup) && !bytes.Equal(method, methodSendToManager) {
			worker2Logger.Error("received wrong method, wsHeader: %v,body:%v", wsHeader, body)
			return
		}

		if bytes.Equal(method, methodSendToManager) {
			body = body[netIdSize:]
		}

		from := strconv.FormatUint(wsHeader.sourceId, 10)
		message, error := unMarshalMessage(body)
		if error != nil {
			worker2Logger.Errorf("Proto unmarshal node message error: %s", error.Error())
			return
		}
		assert.Equal(t, message.Code, uint32(len(message.Body)))
		worker2Logger.Debugf("nonce:%d,rcv from %s,size:%d", rcvNonce, from, message.Code)
		for _, item := range message.Body {
			assert.Equal(t, item, uint8(6))
		}
	}

	for i := 0; ; i++ {
		var msg []byte
		if sendBigMessage() {
			msg = genBigMessage()
			worker2Logger.Debugf("send big msg %d,size:%d,", i, uint32(len(msg)))
		} else {
			msg = genSmallMessage()
			worker2Logger.Debugf("send small msg %d,size:%d,", i, uint32(len(msg)))
		}
		message := Message{Code: uint32(len(msg)), Body: msg}
		worker1.SendToEveryone(message)
		time.Sleep(time.Second * 3)
	}

}

func sendBigMessage() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(100) > 50
}

func genBigMessage() []byte {
	size := rand.Intn(msgSizeThreshold)
	size += msgSizeThreshold
	msg := make([]byte, 0)
	for i := 0; i < size; i++ {
		msg = append(msg, 6)
	}
	return msg
}

func genSmallMessage() []byte {
	size := rand.Intn(1000)
	msg := make([]byte, 0)
	for i := 0; i < size; i++ {
		msg = append(msg, 6)
	}
	return msg
}
