package network

import (
	"fmt"
	"testing"
	"context"
	"time"
	"math/rand"
	"sync"
	"bufio"

	"x/src/middleware/log"
	"x/src/common"
	"x/src/utility"

	"github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
)

func TestSendMessage(t *testing.T) {
	defer log.Close()

	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = UnmarshalEcdsaPublicKey

	common.InitConf("test.ini")

	ctx := context.Background()

	seedPrivateKey := "0x04d46485dfa6bb887daec6c35c707c4eaa58e2ea0cafbc8b40201b7759f611e3f27c7d3d3e5835d55e622b90a5d2f24172c80947f97544acd5cf8ed3f4d94f4243f3092f031b85e4675634bf60434a590e954c8051d42c53ced1744eaf32e47395"
	seedDht, seedHost, seedId := mockDHT(seedPrivateKey, true)
	fmt.Printf("Mock seed dht success! seedId is:%s\n\n", idToString(seedId))

	node1Dht, node1Host, node1Id := mockDHT("", false)
	fmt.Printf("Mock dht node1 success! node1 id is:%s\n\n", idToString(node1Id))

	node2Dht, node2Host, node2Id := mockDHT("", false)
	fmt.Printf("Mock dht node2 success! node2 id is:%s\n\n", idToString(node2Id))

	if node1Dht != nil && node2Dht != nil && seedDht != nil {
		connectToSeed(ctx, node1Host)
		connectToSeed(ctx, node2Host)
	}
	node2Server := server{host: node2Host, dht: node2Dht, streams: make(map[string]inet.Stream), streamMapLock: sync.RWMutex{}}

	seedServer := server{host: seedHost, dht: seedDht, streams: make(map[string]inet.Stream), streamMapLock: sync.RWMutex{}}

	node1Server := server{host: node1Host, dht: node1Dht, streams: make(map[string]inet.Stream), streamMapLock: sync.RWMutex{}}
	node1Server.host.SetStreamHandler(protocolID, testSteamHandler)

	message := mockMessage()
	seedServer.Send(idToString(node1Id), message)
	fmt.Printf("Send message code %d,msg len:%d\n", message.Code, len(message.Body))

	time.Sleep(50 * time.Millisecond)
	dumpConn(seedServer, node1Server, node2Server)
}

func testSteamHandler(stream inet.Stream) {
	defer stream.Close()
	id := idToString(stream.Conn().RemotePeer())
	reader := bufio.NewReader(stream)

	headerBytes := make([]byte, 3)
	h, e1 := reader.Read(headerBytes)
	if e1 != nil {
		fmt.Printf("steam read 3 from %s error:%s!", id, e1.Error())
		return
	}
	if h != 3 {
		fmt.Printf("Stream  should read %d byte, but received %d bytes", 3, h)
		return
	}
	//校验 header
	if !(headerBytes[0] == header[0] && headerBytes[1] == header[1] && headerBytes[2] == header[2]) {
		logger.Errorf("validate header error from %s! ", id)
		return
	}

	pkgLengthBytes := make([]byte, packageLengthSize)
	n, err := reader.Read(pkgLengthBytes)
	if err != nil {
		fmt.Printf("Stream  read4 error:%s", err.Error())
		return
	}
	if n != 4 {
		fmt.Printf("Stream  should read %d byte, but received %d bytes", 4, n)
		return
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	b := make([]byte, pkgLength)
	e := readMessageBody(reader, b, 0)
	if e != nil {
		fmt.Printf("Stream  readMessageBody error:%s", e.Error())
	}

	message, e := unMarshalMessage(b)
	if e != nil {
		fmt.Printf("Unmarshal message error!" + e.Error())
		return
	}
	fmt.Printf("Reviced message code %d,msg len:%d\n", message.Code, len(message.Body))
}

func mockMessage() Message {
	var code = rand.Uint32()

	r := rand.Intn(1000000)
	body := make([]byte, r)
	for i := 0; i < r; i++ {
		body[i] = 8
	}
	m := Message{Code: code, Body: body}
	return m
}

func dumpConn(seedServer server, node1Server server, node2Server server) {
	conns := seedServer.ConnInfo()
	for _, conn := range conns {
		fmt.Printf("seed server's conn:%s,%s,%s\n", conn.Id, conn.Ip, conn.Port)
	}

	conn1 := node1Server.ConnInfo()
	for _, conn := range conn1 {
		fmt.Printf("node1 server's conn:%s,%s,%s\n", conn.Id, conn.Ip, conn.Port)
	}

	conn2 := node2Server.ConnInfo()
	for _, conn := range conn2 {
		fmt.Printf("node2 server's conn:%s,%s,%s\n", conn.Id, conn.Ip, conn.Port)
	}
}
