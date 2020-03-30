package cli

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"time"
	"x/src/common"
	"x/src/consensus"
	"x/src/consensus/logical/group_create"
	"x/src/consensus/model"
	cnet "x/src/consensus/net"
	"x/src/core"
	"x/src/middleware"
	"x/src/middleware/db"
	"x/src/middleware/log"
	"x/src/middleware/types"
	"x/src/network"
	"x/src/service"
	"x/src/statemachine"
)

const (
	GXVersion = "0.0.7.3"
	// Section 默认section配置
	Section = "gx"

	instanceSection = "instance"

	indexKey = "index"
)

type GX struct {
	init    bool
	account Account
}

func NewGX() *GX {
	return &GX{}
}

func (gx *GX) Run() {
	// control+c 信号
	ctrlC := signals()
	quitChan := make(chan bool)
	go gx.handleExit(ctrlC, quitChan)

	app := kingpin.New("gx", "A blockchain layer 2 application.")
	app.HelpFlag.Short('h')

	configFile := app.Flag("config", "Config file").Default("tas.ini").String()
	_ = app.Flag("metrics", "enable metrics").Bool()
	_ = app.Flag("dashboard", "enable metrics dashboard").Bool()
	pprofPort := app.Flag("pprof", "enable pprof").Default("23333").Uint()

	//控制台
	consoleCmd := app.Command("console", "start RocketProtocol console")
	showRequest := consoleCmd.Flag("show", "show the request json").Short('v').Bool()
	remoteHost := consoleCmd.Flag("host", "the node host address to connect").Short('i').String()
	remotePort := consoleCmd.Flag("port", "the node host port to connect").Short('p').Default("8101").Int()
	rpcPort := consoleCmd.Flag("rpcport", "RocketProtocol console will listen at the port for wallet service").Short('r').Default("0").Int()
	//版本号
	versionCmd := app.Command("version", "show RocketProtocol version")
	// mine
	mineCmd := app.Command("miner", "miner start")
	// rpc解析
	rpc := mineCmd.Flag("rpc", "start rpc server").Bool()
	addrRpc := mineCmd.Flag("rpcaddr", "rpc host").Short('r').Default("0.0.0.0").IP()
	portRpc := mineCmd.Flag("rpcport", "rpc port").Short('p').Default("8088").Uint()
	instanceIndex := mineCmd.Flag("instance", "instance index").Short('i').Default("0").Int()
	env := mineCmd.Flag("env", "the environment application run in").String()

	//自定义网关
	gateAddr := mineCmd.Flag("gateaddr", "the gate addr").String()
	command, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}

	common.InitConf(*configFile)
	walletManager = newWallets()
	common.GlobalConf.SetInt(instanceSection, indexKey, *instanceIndex)
	common.DefaultLogger = log.GetLoggerByIndex(log.DefaultConfig, common.GlobalConf.GetString(instanceSection, indexKey, ""))

	fmt.Println("Welcome to be a rocketProtocol miner!")
	switch command {
	case versionCmd.FullCommand():
		fmt.Println("GX Version:", GXVersion)
		os.Exit(0)
	case consoleCmd.FullCommand():
		err := ConsoleInit(*remoteHost, *remotePort, *showRequest, *rpcPort)
		if err != nil {
			fmt.Errorf(err.Error())
		}
	case mineCmd.FullCommand():
		go func() {
			http.ListenAndServe(fmt.Sprintf(":%d", *pprofPort), nil)
			runtime.SetBlockProfileRate(1)
			runtime.SetMutexProfileFraction(1)
		}()
		gx.initMiner(*instanceIndex, *env, *gateAddr)
		if *rpc {
			err = StartRPC(addrRpc.String(), *portRpc, gx.account.Sk)
			if err != nil {
				common.DefaultLogger.Infof(err.Error())
				return
			}
		}
	}
	<-quitChan
}

func (gx *GX) initMiner(instanceIndex int, env, gateAddr string) {
	common.InstanceIndex = instanceIndex
	common.GlobalConf.SetInt(instanceSection, indexKey, instanceIndex)
	databaseValue := "d" + strconv.Itoa(instanceIndex)
	common.GlobalConf.SetString(db.ConfigSec, db.DefaultDatabase, databaseValue)
	joinedGroupDatabaseValue := "jgs" + strconv.Itoa(instanceIndex)
	common.GlobalConf.SetString(db.ConfigSec, db.DefaultJoinedGroupDatabaseKey, joinedGroupDatabaseValue)

	middleware.InitMiddleware()

	privateKey := common.GlobalConf.GetString(Section, "privateKey", "")
	gx.getAccountInfo(privateKey)
	fmt.Println("Your Miner Address:", gx.account.Address)

	minerInfo := model.NewSelfMinerInfo(gx.account.Miner.ID[:])
	common.GlobalConf.SetString(Section, "miner", minerInfo.ID.GetHexString())
	minerId := "0x" + common.Bytes2Hex(gx.account.Miner.ID[:])
	network.InitNetwork(cnet.MessageHandler, minerId, env, gateAddr)
	service.InitService()

	err := core.InitCore(consensus.NewConsensusHelper(minerInfo.ID))
	if err != nil {
		panic("Init miner core init error:" + err.Error())
	}

	//todo: 刷新requestId
	statemachine.InitSTMManager(common.GlobalConf.GetString("docker", "config", ""), minerId)

	ok := consensus.ConsensusInit(minerInfo, common.GlobalConf)
	if !ok {
		panic("Init miner consensus init error!")

	}
	gx.dumpAccountInfo(minerInfo)

	//consensus.Proc.BeginGenesisGroupMember()
	group_create.GroupCreateProcessor.BeginGenesisGroupMember()

	ok = consensus.StartMiner()
	if !ok {
		panic("Init miner start miner error!")
	}
	syncChainInfo()
	gx.init = true
}

func (gx *GX) getAccountInfo(sk string) {
	if 0 == len(sk) {
		privateKey := common.GenerateKey("")
		sk = privateKey.GetHexString()
	}

	gx.account = getAccountByPrivateKey(sk)
}

func syncChainInfo() {
	fmt.Println("Syncing block and group info from RocketProtocol net.Waiting...")
	core.InitGroupSyncer()
	core.InitBlockSyncer()
	go func() {
		timer := time.NewTimer(time.Second * 10)
		for {
			<-timer.C
			if core.BlockSyncer.IsInit() {
				break
			} else {
				var candicateHeight uint64
				if core.BlockSyncer != nil {
					core.BlockSyncer.Lock.Lock("trySync")
					_, _, candicateHeight, _ = core.BlockSyncer.GetCandidateForSync()
					core.BlockSyncer.Lock.Unlock("trySync")
				}
				localBlockHeight := core.GetBlockChain().Height()
				jsonObject := types.NewJSONObject()
				jsonObject.Put("candidateHeight", candicateHeight)
				jsonObject.Put("localHeight", localBlockHeight)
				middleware.HeightLogger.Debugf(jsonObject.TOJSONString())
				fmt.Printf("Sync candidate block height:%d,local block height:%d\n", candicateHeight, localBlockHeight)
				timer.Reset(time.Second * 5)
			}
		}
		fmt.Println("Sync data finished!")
		fmt.Println("Start Mining...")
	}()
}

func (gx *GX) dumpAccountInfo(minerDO model.SelfMinerInfo) {
	if nil != common.DefaultLogger {
		common.DefaultLogger.Infof("SecKey: %s", gx.account.Sk)
		common.DefaultLogger.Infof("PubKey: %s", gx.account.Pk)
		common.DefaultLogger.Infof("Miner SecKey: %s", minerDO.SecKey.GetHexString())
		common.DefaultLogger.Infof("Miner PubKey: %s", minerDO.PubKey.GetHexString())
		common.DefaultLogger.Infof("VRF PrivateKey: %s", minerDO.VrfSK.GetHexString())
		common.DefaultLogger.Infof("VRF PubKey: %s", minerDO.VrfPK.GetHexString())
		common.DefaultLogger.Infof("Miner ID: %s", minerDO.ID.GetHexString())
	}

}

func (gx *GX) handleExit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	if core.GetBlockChain() == nil {
		return
	}
	fmt.Println("exiting...")
	core.GetBlockChain().Close()
	log.Close()
	consensus.StopMiner()
	if gx.init {
		quit <- true
	} else {
		os.Exit(0)
	}
}
