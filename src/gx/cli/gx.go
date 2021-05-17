// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus"
	"com.tuntun.rocket/node/src/consensus/logical/group_create"
	"com.tuntun.rocket/node/src/consensus/model"
	cnet "com.tuntun.rocket/node/src/consensus/net"
	"com.tuntun.rocket/node/src/core"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/statemachine"
	"com.tuntun.rocket/node/src/vm"
	"encoding/json"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"
)

const (
	GXVersion = "0.0.8"
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

	configFile := app.Flag("config", "Config file").Default("rp.ini").String()
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
	rpc := mineCmd.Flag("rpc", "start rpc server").Default("true").Bool()
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

	fmt.Println("Use config file: " + *configFile)
	common.InitConf(*configFile)

	instance := 0
	if 0 != *instanceIndex {
		instance = *instanceIndex
		common.GlobalConf.SetInt(instanceSection, indexKey, *instanceIndex)
	} else {
		instance = common.GlobalConf.GetInt(instanceSection, indexKey, 0)
	}

	common.DefaultLogger = log.GetLoggerByIndex(log.DefaultConfig, common.GlobalConf.GetString(instanceSection, indexKey, ""))

	walletManager = newWallets()
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
		gx.initMiner(instance, *env, *gateAddr)
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
	databaseValue := "chain"
	common.GlobalConf.SetString(db.ConfigSec, db.DefaultDatabase, databaseValue)
	joinedGroupDatabaseValue := "jgs"
	common.GlobalConf.SetString(db.ConfigSec, db.DefaultJoinedGroupDatabaseKey, joinedGroupDatabaseValue)

	middleware.InitMiddleware()

	privateKey := common.GlobalConf.GetString(Section, "privateKey", "")
	gx.getAccountInfo(privateKey)
	fmt.Println("Your Miner Address:", gx.account.Address)

	sk := common.HexStringToSecKey(gx.account.Sk)
	minerInfo := model.NewSelfMinerInfo(*sk)
	common.GlobalConf.SetString(Section, "miner", minerInfo.ID.GetHexString())

	network.InitNetwork(cnet.MessageHandler, minerInfo.ID.Serialize(), env, gateAddr)
	service.InitService()
	vm.InitVM()

	err := core.InitCore(consensus.NewConsensusHelper(minerInfo.ID), *sk, minerInfo.ID.GetHexString())
	if err != nil {
		panic("Init miner core init error:" + err.Error())
	}

	//todo: 刷新requestId
	statemachine.InitSTMManager(common.GlobalConf.GetString("docker", "config", ""), common.ToHex(gx.account.Miner.ID[:]))

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
	syncChainInfo(*sk, minerInfo.ID.GetHexString())
	gx.init = true
}

func (gx *GX) getAccountInfo(sk string) {
	if 0 == len(sk) {
		privateKey := common.GenerateKey("")
		sk = privateKey.GetHexString()
		common.GlobalConf.SetString(Section, "privateKey", sk)
	}

	gx.account = getAccountByPrivateKey(sk)
}

func syncChainInfo(privateKey common.PrivateKey, id string) {
	fmt.Println("Syncing block and group info from RocketProtocol net.Waiting...")
	go func() {
		timer := time.NewTimer(time.Second * 10)
		for {
			<-timer.C

			var candidateHeight uint64
			if core.SyncProcessor != nil {
				candidate := core.SyncProcessor.GetCandidateInfo()
				candidateHeight = candidate.Height
			}
			localBlockHeight := core.GetBlockChain().Height()
			jsonObject := types.NewJSONObject()
			jsonObject.Put("candidateHeight", candidateHeight)
			jsonObject.Put("localHeight", localBlockHeight)
			if candidateHeight > 0 {
				middleware.HeightLogger.Debugf(jsonObject.TOJSONString())
			}
			timer.Reset(time.Second * 5)
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

		miner := types.Miner{}
		miner.Id = minerDO.ID.Serialize()
		miner.PublicKey = minerDO.PubKey.Serialize()
		miner.VrfPublicKey = minerDO.VrfPK
		minerBytes, _ := json.Marshal(miner)
		common.DefaultLogger.Infof("Miner apply info:%s|%s", minerDO.ID.GetHexString(), string(minerBytes))
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
