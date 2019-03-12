package network

import (
	"net"
	"strconv"
	"math/rand"
	"context"
	"time"

	"common"
	"middleware/log"

	"github.com/libp2p/go-libp2p-swarm"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/go-libp2p-host"
	lnet "github.com/libp2p/go-libp2p-net"
	ma "github.com/multiformats/go-multiaddr"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
)

const (
	seedId = "seed_id"

	seedAddress = "seed_address"

	basePort = 1122

	baseSection = "network"

	defaultSeedId = "QmU51xws4zKRHdirwkLCeQPviVv3m74TQ9o76bcyzDGo23"

	defaultSeedAddr = "/ip4/10.0.0.66/tcp/1122"
)

var logger log.Logger

func InitNetwork(privateKey common.PrivateKey, isSuper bool) {
	logger = log.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))

	publicKey := privateKey.GetPubKey()
	id := getId(publicKey)
	ip := getLocalIp()
	port := getAvailablePort(isSuper, basePort)
	logger.Debugf("Local ip:%s,listen port:%d\nID:%s", ip, port, idToString(id))

	ctx := context.Background()
	swarm := makeSwarm(ctx, id, ip, port, privateKey, publicKey)
	host := makeHost(swarm)
	dht := makeDHT(ctx, host)
	if !isSuper {
		connectToSeed(ctx, host)
	}
	initServer(host, dht)
	tryFindSeed(ctx)
}

func makeSwarm(ctx context.Context, id peer.ID, ip string, port int, privateKey common.PrivateKey, publicKey common.PublicKey) lnet.Network {
	addr, e1 := ma.NewMultiaddr(genMulAddrStr(ip, port))
	if e1 != nil {
		panic("New multi addr error:" + e1.Error())
	}
	listenAddrs := []ma.Multiaddr{addr}

	peerStore := pstore.NewPeerstore()
	p1 := &Pubkey{PublicKey: publicKey}
	p2 := &Privkey{PrivateKey: privateKey}
	peerStore.AddPubKey(id, p1)
	peerStore.AddPrivKey(id, p2)

	peerStore.AddAddrs(id, listenAddrs, pstore.PermanentAddrTTL)
	//bwc  is a bandwidth metrics collector, This is used to track incoming and outgoing bandwidth on connections managed by this swarm.
	// It is optional, and passing nil will simply result in no metrics for connections being available.
	swarm, e2 := swarm.NewNetwork(ctx, listenAddrs, id, peerStore, nil)
	if e2 != nil {
		panic("New swarm error!\n" + e2.Error())
	}
	return swarm
}

func makeHost(n lnet.Network) (host.Host) {
	opt := basichost.HostOpts{}
	opt.NegotiationTimeout = -1
	host := basichost.New(n)
	return host
}

func connectToSeed(ctx context.Context, host host.Host) {
	seedId, seedAddrStr := getSeedInfo()
	seedMultiAddr, e := ma.NewMultiaddr(seedAddrStr)
	if e != nil {
		panic("New multi addr error:" + e.Error())
	}
	seedPeerInfo := pstore.PeerInfo{ID: seedId, Addrs: []ma.Multiaddr{seedMultiAddr}}
	host.Peerstore().AddAddrs(seedPeerInfo.ID, seedPeerInfo.Addrs, pstore.PermanentAddrTTL)

	e = host.Connect(ctx, seedPeerInfo)
	if e != nil {
		logger.Errorf("Connect to seed error:", e.Error())
		for i := 1; i <= 3; i++ {
			time.Sleep(time.Second * 5)
			logger.Infof("Try to connect to seed:no %d\n", i)
			e := host.Connect(ctx, seedPeerInfo)
			if e == nil {
				break
			}
		}
	}
}

func makeDHT(ctx context.Context, host host.Host) (*dht.IpfsDHT) {
	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)

	cfg := dht.DefaultBootstrapConfig
	cfg.Queries = 3
	cfg.Period = time.Duration(10 * time.Second)
	cfg.Timeout = time.Second * 30
	process, e := kadDht.BootstrapWithConfig(cfg)
	if e != nil {
		process.Close()
		panic("KadDht bootstrap error!" + e.Error())
	}
	return kadDht
}

func tryFindSeed(ctx context.Context) {
	seedId, _ := getSeedInfo()
	if seedId != Server.host.ID() {
		for {
			info, e := Server.dht.FindPeer(ctx, seedId)
			if e != nil {
				logger.Infof("Find seed id %s error:%s", idToString(seedId), e.Error())
				time.Sleep(5 * time.Second)
			} else if idToString(info.ID) == "" {
				logger.Infof("Can not find seed node,finding....")
				time.Sleep(5 * time.Second)
			} else {
				logger.Infof("Welcome to join X Network!")
				break
			}
		}
	}
}

func getSeedInfo() (peer.ID, string) {
	seedIdStr := common.GlobalConf.GetString(baseSection, seedId, defaultSeedId)
	seedAddr := common.GlobalConf.GetString(baseSection, seedAddress, defaultSeedAddr)
	return strToId(seedIdStr), seedAddr
}

func strToId(i string) peer.ID {
	id, e := peer.IDB58Decode(i)
	if e != nil {
		panic("string to id error:" + e.Error())
	}
	return id
}

//"/ip4/127.0.0.1/tcp/1234"
func genMulAddrStr(ip string, port int) string {
	return "/ip4/" + ip + "/tcp/" + strconv.Itoa(port)
}

func getLocalIp() string {
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func getAvailablePort(isSuper bool, port int) int {
	if isSuper {
		return basePort
	}
	rand.Seed(time.Now().UnixNano())
	port += rand.Intn(1000)
	return port
}
