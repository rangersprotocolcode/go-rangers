package consensus

import (
	"x/src/consensus/logical"
	"x/src/consensus/model"
	"x/src/consensus/net"
	"x/src/common"
)

///////////////////////////////////////////////////////////////////////////////
//共识模块提供给主框架的接口

//所有私钥，公钥，地址，ID的对外格式均为“0xa19d...854e”的加前缀十六进制格式

var Proc logical.Processor


//共识初始化
//mid: 矿工ID
//返回：true初始化成功，可以启动铸块。内部会和链进行交互，进行初始数据加载和预处理。失败返回false。
func ConsensusInit(mi model.SelfMinerDO, conf common.ConfManager) bool {
	logical.InitConsensus()
	//groupsig.Init(1)
	ret := Proc.Init(mi, conf)
	net.MessageHandler.Init(&Proc)
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
