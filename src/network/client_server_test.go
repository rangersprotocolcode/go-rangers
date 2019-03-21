package network

import (
	"testing"
	"time"
	"context"
	"sync"

	"x/src/common"
	"x/src/middleware/log"

	inet "github.com/libp2p/go-libp2p-net"
)

func TestServerNet(t *testing.T) {
	defaultSeedAddr = "/ip4/192.168.3.210/tcp/1122"
	clientId := "0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76"
	mockSeedServer()

	go func() {
		seedId := "0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"
		clientServer := mockClientServer()
		for i := 0; i < 100; i++ {
			m := mockMessage()
			clientServer.Send(seedId, m)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	time.Sleep(time.Second * 1)
	for i := 0; i < 100; i++ {
		m := mockMessage()
		Server.Send(clientId, m)
		time.Sleep(100 * time.Millisecond)
	}
	log.Close()
}

func mockSeedServer() {
	common.InitConf("test.ini")
	seedPrivateKeyStr := "0x041629de511d8f53d5a0ccf1676021708a15c6d85fad6765e33bae95eb15f6f9b10e0360813c63fc2cccfdd2e7a8ddbfe7eb84fe50555383d0323475622c5216f3e01cfb4e1c156a795fa6d525fb481727dabcbf066b3a153daf835f5570599a79"
	privateKey := *common.HexStringToSecKey(seedPrivateKeyStr)
	InitNetwork(privateKey, true, nil)
}

func mockClientServer() server {
	common.InitConf("test.ini")
	//privateKey := common.GenerateKey("")
	//common.GlobalConf.SetString("network", "privateKey", privateKey.GetHexString())
	clientPrivateKeyStr := "0x04b9d93a1997592e2b165cd1ba5a06baee709f33bd7504179b8c229e9c695aaab741150a66fe03eaeb97b2b6d07b981df6b5d9f703bb2a055fa5919343f5ad414264de573e668cd3a4fdda46f785b473672714ecdb00b81f1e75f7cd697309f53d"
	privateKey := *common.HexStringToSecKey(clientPrivateKeyStr)

	publicKey := privateKey.GetPubKey()
	id := getId(publicKey)
	ip := getLocalIp()
	port := getAvailablePort(false, basePort)
	Logger.Debugf("Local ip:%s,listen port:%d\nID:%s", ip, port, idToString(id))

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
