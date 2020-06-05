package consensus

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/access"
	"com.tuntun.rocket/node/src/consensus/logical"
	"com.tuntun.rocket/node/src/consensus/logical/group_create"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/net"
)

///////////////////////////////////////////////////////////////////////////////
//共识模块提供给主框架的接口

//所有私钥，公钥，地址，ID的对外格式均为“0xa19d...854e”的加前缀十六进制格式

var Proc logical.Processor

//共识初始化
//mid: 矿工ID
//返回：true初始化成功，可以启动铸块。内部会和链进行交互，进行初始数据加载和预处理。失败返回false。
func ConsensusInit(mi model.SelfMinerInfo, conf common.ConfManager) bool {
	logical.InitConsensus()
	joinedGroupStorage := initJoinedGroupStorage()

	group_create.GroupCreateProcessor.Init(mi, joinedGroupStorage)
	ret := Proc.Init(mi, conf, joinedGroupStorage)
	net.MessageHandler.Init(&group_create.GroupCreateProcessor, &Proc)
	
	return ret
}

//启动矿工进程，参与铸块。
//成功返回true，失败返回false。
func StartMiner() bool {
	return Proc.Start()
}

//结束矿工进程，不再参与铸块。
func StopMiner() {
	Proc.Stop()
	Proc.Finalize()
	return
}

func initJoinedGroupStorage() *access.JoinedGroupStorage {
	return access.NewJoinedGroupStorage()
}
