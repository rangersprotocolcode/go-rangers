// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	cnet "com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/eth_rpc"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/network"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
	"com.tuntun.rangers/node/src/vm"
	"encoding/json"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"time"
)

const (
	GXVersion = "1.0.23"
	// Section 默认section配置
	Section = "gx"
)

type GX struct {
	init    bool
	account Account
}

func NewGX() *GX {
	return &GX{}
}

func (gx *GX) Run() {
	ctrlC := signals()
	quitChan := make(chan bool)
	go gx.handleExit(ctrlC, quitChan)

	app := kingpin.New("gx", "A blockchain layer 2 application.")
	app.HelpFlag.Short('h')

	configFile := app.Flag("config", "Config file").Default("rp.ini").String()

	_ = app.Flag("metrics", "enable metrics").Bool()
	_ = app.Flag("dashboard", "enable metrics dashboard").Bool()

	consoleCmd := app.Command("console", "start RangersProtocol console")
	showRequest := consoleCmd.Flag("show", "show the request json").Short('v').Bool()
	remoteHost := consoleCmd.Flag("host", "the node host address to connect").Short('i').String()
	remotePort := consoleCmd.Flag("port", "the node host port to connect").Short('p').Default("8101").Int()
	rpcPort := consoleCmd.Flag("rpcport", "RangersProtocol console will listen at the port for wallet service").Short('r').Default("0").Int()

	versionCmd := app.Command("version", "show RangersProtocol version")

	mineCmd := app.Command("miner", "miner start")

	rpc := mineCmd.Flag("rpc", "start rpc server").Default("true").Bool()
	addrRpc := mineCmd.Flag("rpcaddr", "rpc host").Short('r').Default("0.0.0.0").IP()
	portRpc := mineCmd.Flag("rpcport", "rpc port").Short('p').Default("8088").Uint()
	instanceIndex := mineCmd.Flag("instance", "instance index").Short('i').Default("0").Int()

	env := mineCmd.Flag("env", "the environment application run in").String()
	gateAddrPoint := mineCmd.Flag("gateaddr", "the gate addr").String()
	outerGateAddrPoint := mineCmd.Flag("outergateaddr", "the gate addr").String()
	txAddrPoint := mineCmd.Flag("tx", "the tx queue addr").String()

	//fullnode
	fullNodeCmd := app.Command("fullnode", "sync data full node")
	fullNodeJSONPRCHttpPort := fullNodeCmd.Flag("httpport", "jsonrpc http port").Short('p').Default("7988").Uint()
	fullNodeJSONPRCWSPort := fullNodeCmd.Flag("wsport", "jsonrpc ws port").Short('w').Default("7989").Uint()
	fullNodeEnv := fullNodeCmd.Flag("env", "the environment application run in").String()

	command, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}

	switch command {
	case versionCmd.FullCommand():
		fmt.Println("Version:", GXVersion)
		os.Exit(0)
	case consoleCmd.FullCommand():
		err := ConsoleInit(*remoteHost, *remotePort, *showRequest, *rpcPort)
		if err != nil {
			fmt.Errorf(err.Error())
		}
	case mineCmd.FullCommand():
		common.Init(*instanceIndex, *configFile, *env)
		// using default
		gateAddr := *gateAddrPoint
		if 0 == len(gateAddr) {
			gateAddr = common.LocalChainConfig.PHub
		}
		outerGateAddr := *outerGateAddrPoint
		txAddr := *txAddrPoint

		walletManager = newWallets()
		fmt.Println("Use config file: " + *configFile)
		fmt.Printf("Env:%s, Chain ID:%s, Network ID:%s, Tx: %s\n", *env, common.ChainId(utility.MaxUint64), common.NetworkId(), txAddr)
		gx.initMiner(*env, gateAddr, outerGateAddr, txAddr)

		if *rpc {
			err = StartRPC(addrRpc.String(), *portRpc, gx.account.Sk)
			if err != nil {
				common.DefaultLogger.Infof(err.Error())
				return
			}
		}
	case fullNodeCmd.FullCommand():
		gx.initFullNode(*fullNodeEnv, *configFile, *fullNodeJSONPRCHttpPort, *fullNodeJSONPRCWSPort)
		break
	}
	<-quitChan
}

func (gx *GX) initMiner(env, gateAddr, outerGateAddr, tx string) {
	common.DefaultLogger.Infof("start initMiner")
	defer func() {
		common.DefaultLogger.Infof("end initMiner")
	}()

	privateKey := common.GlobalConf.GetString(Section, "privateKey", "")
	gx.getAccountInfo(privateKey)
	fmt.Println("Your Miner Address:", gx.account.Address)
	sk := common.HexStringToSecKey(gx.account.Sk)
	minerInfo := model.NewSelfMinerInfo(*sk)
	common.GlobalConf.SetString(Section, "miner", minerInfo.ID.GetHexString())
	gx.dumpAccountInfo(minerInfo)

	account.Init()

	middleware.InitMiddleware()

	service.InitService()

	network.InitNetwork(cnet.MessageHandler, minerInfo.ID.Serialize(), env, gateAddr, outerGateAddr, 0 != len(outerGateAddr) && 0 != len(tx))

	vm.InitVM()

	err := core.InitCore(consensus.NewConsensusHelper(minerInfo.ID), *sk, minerInfo.ID.GetHexString())
	if err != nil {
		panic("Init miner core init error:" + err.Error())
	}

	if common.IsSub() {
		accountDB := middleware.AccountDBManagerInstance.GetLatestStateDB()
		if nil == accountDB {
			panic("wrong accountDB")
		}

		if 3 == service.GetSubChainStatus(accountDB) {
			fmt.Println("subChain ended")
			return
		}
	}

	if !common.IsSub() {
		network.GetNetInstance().InitTx(tx)
	}

	ok := consensus.InitConsensus(minerInfo, common.GlobalConf)
	if !ok {
		panic("Init miner consensus init error!")

	}

	group_create.GroupCreateProcessor.BeginGenesisGroupMember()

	ok = consensus.StartMiner()
	if !ok {
		panic("Init miner start miner error!")
	}

	syncChainInfo()

	if common.IsSub() {
		checkStatus()
	}

	eth_rpc.InitEthMsgHandler()
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

func syncChainInfo() {
	start := time.Now()
	common.DefaultLogger.Infof("start syncChainInfo")
	defer func() {
		common.DefaultLogger.Infof("end syncChainInfo, cost: %s", time.Now().Sub(start).String())
	}()

	core.StartSync()

	go func() {
		timer := time.NewTicker(time.Second * 10)
		output := true
		for {
			<-timer.C

			var candidateHeight = core.GetCandidateHeight()
			topBlock := core.GetBlockChain().TopBlock()
			jsonObject := types.NewJSONObject()
			jsonObject.Put("chainId", common.ChainId(utility.MaxUint64))
			jsonObject.Put("instanceNum", common.InstanceIndex)
			jsonObject.Put("candidateHeight", candidateHeight)
			if topBlock != nil {
				jsonObject.Put("localHeight", topBlock.Height)
				jsonObject.Put("topBlockHash", topBlock.Hash.String())

				if output && candidateHeight > 0 && topBlock.Height >= candidateHeight {
					fmt.Println("Sync data finished!")
					fmt.Println("Start Mining...")
					output = false
				}
			}
			middleware.MonitorLogger.Infof("|height|%s", jsonObject.TOJSONString())
		}
	}()
}

func checkStatus() {
	go func() {
		for range time.Tick(10 * time.Second) {
			accountDB := middleware.AccountDBManagerInstance.GetLatestStateDB()
			if nil == accountDB {
				continue
			}

			if 3 == service.GetSubChainStatus(accountDB) {
				fmt.Println("subChain ended")

				consensus.StopMiner()
				if core.GetBlockChain() != nil {
					core.GetBlockChain().Close()
				}
				service.Close()
				middleware.Close()
				log.Close()

				os.Exit(0)
				return
			}
		}
	}()

}

func (gx *GX) dumpAccountInfo(minerDO model.SelfMinerInfo) {
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
	miner.VrfPublicKey = minerDO.VrfPK.GetBytes()
	minerBytes, _ := json.Marshal(miner)
	common.DefaultLogger.Infof("Miner apply info:%s|%s", minerDO.ID.GetHexString(), string(minerBytes))

}

func (gx *GX) handleExit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	fmt.Println("exiting...")
	consensus.StopMiner()

	if core.GetBlockChain() != nil {
		core.GetBlockChain().Close()
	}

	service.Close()
	middleware.Close()
	log.Close()
	if gx.init {
		quit <- true
	} else {
		os.Exit(0)
	}
}

func (gx *GX) initFullNode(env string, configFile string, jsonRPCHttpPort uint, jsonRPCWSPort uint) {
	common.Init(0, configFile, env)
	common.SetFullNode(true)
	fmt.Printf("Start full node mode.\n")
	fmt.Printf("Chain ID:%s,Network ID:%s\n", common.ChainId(utility.MaxUint64), common.NetworkId())

	outerGateAddr := "" //do not connect outerGate
	tx := ""            //do not connect tx
	//dsn := ""           //do not connect dsn
	gateAddr := common.LocalChainConfig.PHub
	gx.initMiner(env, gateAddr, outerGateAddr, tx)

	err := StartJSONRPCHttp(jsonRPCHttpPort)
	if err != nil {
		panic(err)
	}

	err = StartJSONRPCWS(jsonRPCWSPort)
	if err != nil {
		panic(err)
	}
	gx.init = true
}
