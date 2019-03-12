package network

import (
	"testing"
	"time"
	"context"
	"sync"

	"common"
	"middleware/log"

	inet "github.com/libp2p/go-libp2p-net"
)

func TestServerNet(t *testing.T) {
	clientId := "QmT8HAeTX7oZzUzNdL3iAqfiYsUEh5ZDyhKksMHxz7LWjU"
	mockSeedServer()

	go func() {
		seedId := "QmU51xws4zKRHdirwkLCeQPviVv3m74TQ9o76bcyzDGo23"
		clientServer := mockClientServer()
		for i := 0; i < 100; i++ {
			m := mockMessage()
			clientServer.SendMessage(m, seedId)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	time.Sleep(time.Second * 1)
	for i := 0; i < 100; i++ {
		m := mockMessage()
		Server.SendMessage(m, clientId)
		time.Sleep(100 * time.Millisecond)
	}
	log.Close()
}

func mockSeedServer() {
	common.InitConf("test.ini")
	seedPrivateKeyStr := "0x04d46485dfa6bb887daec6c35c707c4eaa58e2ea0cafbc8b40201b7759f611e3f27c7d3d3e5835d55e622b90a5d2f24172c80947f97544acd5cf8ed3f4d94f4243f3092f031b85e4675634bf60434a590e954c8051d42c53ced1744eaf32e47395"
	privateKey := *common.HexStringToSecKey(seedPrivateKeyStr)
	InitNetwork(privateKey, true)
}

func mockClientServer() server {
	common.InitConf("test.ini")
	//privateKey := common.GenerateKey("")
	//common.GlobalConf.SetString("network", "privateKey", privateKey.GetHexString())
	clientPrivateKeyStr := "0x040d5429d3ca995d8cee9696ae5351a3148295c3aec1d5377279b3ffa2d3ff3d47cc9ba665a59097e83e1d2da496635691cffc28f26fd92a1ff42579c8c3a654ba5928fff9b3fbeeff74ba4c242e3ee9d7323ed87e0a92e081f40490469372d02a"
	privateKey := *common.HexStringToSecKey(clientPrivateKeyStr)

	publicKey := privateKey.GetPubKey()
	id := getId(publicKey)
	ip := getLocalIp()
	port := getAvailablePort(false, basePort)
	logger.Debugf("Local ip:%s,listen port:%d\nID:%s", ip, port, idToString(id))

	ctx := context.Background()
	swarm := makeSwarm(ctx, id, ip, port, privateKey, publicKey)
	host := makeHost(swarm)
	dht := makeDHT(ctx, host)
	connectToSeed(ctx, host)
	host.SetStreamHandler(protocolID, swarmStreamHandler)
	clientServer := server{host: host, dht: dht, streams: make(map[string]inet.Stream), streamMapLock: sync.RWMutex{}}
	tryFindSeed(ctx)
	return clientServer
}
