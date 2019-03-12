package network

import (
	"testing"
	"fmt"
	"net"
)

import (
	"time"
	"context"

	"common"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-host"
)

func TestDHT(t *testing.T) {
	common.InitConf("test.ini")
	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = UnmarshalEcdsaPublicKey
	ctx := context.Background()

	seedPrivateKey := "0x04d46485dfa6bb887daec6c35c707c4eaa58e2ea0cafbc8b40201b7759f611e3f27c7d3d3e5835d55e622b90a5d2f24172c80947f97544acd5cf8ed3f4d94f4243f3092f031b85e4675634bf60434a590e954c8051d42c53ced1744eaf32e47395"
	seedDht, _, seedId := mockDHT(seedPrivateKey, true)
	fmt.Printf("Mock seed dht success! seedId is:%s\n\n", idToString(seedId))

	node1Dht, node1Host, node1Id := mockDHT("", false)
	fmt.Printf("Mock dht node1 success! node1 id is:%s\n\n", idToString(node1Id))

	if node1Dht != nil && seedDht != nil {
		connectToSeed(ctx, node1Host)

		time.Sleep(time.Second * 1)
		r1 := seedDht.FindLocal(node1Id)
		fmt.Printf("Seed local find node1. node1 id is:%s\n", idToString(r1.ID))

		r3, err := seedDht.FindPeer(ctx, node1Id)
		if err != nil {
			fmt.Printf("Seed find node1 error:%s\n", err.Error())
		}
		fmt.Printf("Seed find node1 result is:%s\n", idToString(r3.ID))

		r2 := node1Dht.FindLocal(seedId)
		fmt.Printf("Node1 local find seed. seed id is:%s\n", idToString(r2.ID))

		r4, err1 := node1Dht.FindPeer(ctx, seedId)
		if err1 != nil {
			fmt.Printf("Node1 find seed error:%s\n", err1.Error())
		}
		fmt.Printf("Node1 find seed result is:%s\n", idToString(r4.ID))
	}
}

func mockDHT(privateKeyStr string, isSuper bool) (*dht.IpfsDHT, host.Host, peer.ID) {
	var privateKey common.PrivateKey
	if privateKeyStr == "" {
		privateKey = common.GenerateKey("")
	} else {
		privateKey = *common.HexStringToSecKey(privateKeyStr)
	}
	//fmt.Printf("privatekey:%s",privateKey.GetHexString())
	publicKey := privateKey.GetPubKey()
	id := getId(publicKey)
	ip := getLocalIp()
	port := getAvailablePort(isSuper, basePort)
	fmt.Printf("Local ip:%s,listen port:%d\nID:%s\n", ip, port, idToString(id))

	ctx := context.Background()
	swarm := makeSwarm(ctx, id, ip, port, privateKey, publicKey)
	host := makeHost(swarm)
	dht := makeDHT(ctx, host)
	return dht, host, id
}

func TestID(t *testing.T) {
	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	id := getId(publicKey)
	fmt.Println(idToString(id))
}

func TestUnmarshalEcdsaPublicKey(t *testing.T) {
	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = UnmarshalEcdsaPublicKey

	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	pub := Pubkey{PublicKey: publicKey}
	b1, e := pub.Bytes()
	if e != nil {
		fmt.Errorf("PublicKey to bytes error!\n")
	}

	bytes, e := crypto.MarshalPublicKey(&pub)
	if e != nil {
		fmt.Errorf("MarshalPublicKey Error\n")
	}
	pubKey, i := crypto.UnmarshalPublicKey(bytes)
	if i != nil {
		fmt.Errorf("UnmarshalPublicKey Error\n")
	}
	b2, i4 := pubKey.Bytes()
	if i4 != nil {
		fmt.Errorf("PubKey to bytes Error\n")

	}
	fmt.Printf("Origin public key length is :%d,marshal and unmaishal pub key length is:%d\n", len(b1), len(b2))
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	deadline, ok := ctx.Deadline()
	fmt.Print(deadline, ok)
}

func TestIp(t *testing.T) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Printf("TestIp error:%s", err.Error())
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println(ipnet.IP.String())
				break
			}
		}
	}
}
