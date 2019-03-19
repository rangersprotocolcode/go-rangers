package network

import (
	"testing"
	"fmt"
	"net"
)

import (
	"time"
	"context"

	"x/src/common"

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

	seedPrivateKey := "0x041629de511d8f53d5a0ccf1676021708a15c6d85fad6765e33bae95eb15f6f9b10e0360813c63fc2cccfdd2e7a8ddbfe7eb84fe50555383d0323475622c5216f3e01cfb4e1c156a795fa6d525fb481727dabcbf066b3a153daf835f5570599a79"
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
