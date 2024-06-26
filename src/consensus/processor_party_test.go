package consensus

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/network"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/vm"
	"os"
	"testing"
	"time"
)

const (
	CastVerifyMsg           = "0x0af0020a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d910011a208c85ba47af540e1b19239f5b42c8f133dc1733fe63c8c3c5745cab608b980c6d220f010000000edd9d418000000000ffff2a5071aeb0a7d1986d020139b500493b7555c3d25f7f0331395ae499fd2f0af475a72335d6e9d364c18d550c8a30b5b1d93308e98ab5c93a49aa6c9ca864c003509b68883106797ab5c4d315cd3d8d79110130033a0f010000000edd9db64319a98dc001e042207f88b4f2d36a83640ce5d782a0a20cc2b233de3df2d8a358bf0e7b29e9586a124a20d588982453c1f6278564b44667573ec3eb6ce326a63e33e89b091cfe3de2a88758006a200000000000000000000000000000000000000000000000000000000000000000722000000000000000000000000000000000000000000000000000000000000000007a20e538bb98782b404a27b0bad9bfca651f79b0a0a3053c760405d3607bc413ccfd9a0100a201027b7d1a88010a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d9124040a3ebbe9c609934b216b4e5ce36c797d04bf3c10b54c516cb02636510c4103e30e641714fb23c184b90a0550cce6edaef7309cdcb5c4880b41f17a567ff595e1a207f88b4f2d36a83640ce5d782a0a20cc2b233de3df2d8a358bf0e7b29e9586a122001"
	ConsensusVerifyMessage1 = "0x0a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d912405831f190a857e3657c7b44608e502d952229179291dc059e4f4b0be49c7b037a31d05027f9045ea5302b8f0ef0872949bea34e9167d207427a83ab2b1b46d64f1a88010a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d9124086c3930b135b8f277ecdc0a60de9afb83358453fa4de46d3013ce2b619cab87e2185df67f8b223dbdc65dd26475519769cd957a42faa9634f638925680d1f9211a205437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d1232001"
	ConsensusVerifyMessage2 = "0x0a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d912402f0d536e38c45db81d3eb9b472b3fea6888c87b3c0b92a286e7d729f99f828514049c071f94290318dc4a9bd180b78218598fd4219ea5c1def23cc4f15017ff71a88010a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d912407c864adfc60a9fd662f805dceb37a7b100533a1f2d7b7d5c0edb35110905972c1ff7d1958efdd26fd71c9d0af4fc5ffdaea796177d4f5f3a6dd8ea2bfdd31e411a202a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b772001"
	ConsensusVerifyMessage3 = "0x0a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d91240491ae370df06cce6de58cad9e2cf9f830341ebc226d465b81cb89d333c12997e427e837d95c027e871fc51f937fe3ef9b8d74717b84f7682e9933101c44541551a88010a20d5ba4ac8998ddb4984837af97245e71ba68af8f71f1241d652bcc325c02277d912406df944b158f1607bc3c0dacc600ef09fd0d1336c7bd6469872f6e3c65488326282f9851252b5815a93822f88ef724e1d8ae746684c0adc5b9ece84142ec7e63f1a20b1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c931602001"
)

func TestProcessor_OnMessageCastV2(t *testing.T) {
	preTest()

	go func() {
		cvm2, _ := net.UnMarshalConsensusVerifyMessage(common.FromHex(ConsensusVerifyMessage2))
		Proc.OnMessageVerify(cvm2)
	}()

	go func() {
		ccm, _ := net.UnMarshalConsensusCastMessage(common.FromHex(CastVerifyMsg))
		Proc.OnMessageCast(ccm)
	}()

	go func() {
		ccm, _ := net.UnMarshalConsensusCastMessage(common.FromHex(CastVerifyMsg))
		Proc.OnMessageCast(ccm)
	}()

	go func() {
		cvm1, _ := net.UnMarshalConsensusVerifyMessage(common.FromHex(ConsensusVerifyMessage1))
		Proc.OnMessageVerify(cvm1)
	}()

	go func() {
		cvm3, _ := net.UnMarshalConsensusVerifyMessage(common.FromHex(ConsensusVerifyMessage3))
		Proc.OnMessageVerify(cvm3)
	}()

	time.Sleep(19 * time.Hour)
}

func preTest() {
	func() {
		os.RemoveAll("0.ini")
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
	}()

	common.Init(0, "0.ini", "dev")
	pk := "0x043b1e81d607ab0cd11fa5050437f15d3fbc074d422640686f8c6f4473473c63ebc4e43f0a5faa75216b3d62c0326bf11c33f0c8847d9dc68bded297e28d7cc0a88114f89b10266958149b3d9dc3fb85601e30332636e44d85f00ed2de368e0db8"
	common.GlobalConf.SetString("gx", "signSecKey", "0x2fff2e03051a8e86918f3525836855761dedf7307e5cd82d668e80600a586617")
	privateKey := common.BytesToSecKey(common.FromHex(pk))
	sk := common.HexStringToSecKey(privateKey.GetHexString())
	minerInfo := model.NewSelfMinerInfo(*sk)

	middleware.InitMiddleware()
	service.InitService()
	network.InitNetwork(net.MessageHandler, minerInfo.ID.Serialize(), "dev", "ws://192.168.2.15:1017", "ws://192.168.2.19/pubhub", false)
	vm.InitVM()
	core.InitCore(NewConsensusHelper(minerInfo.ID), *sk, minerInfo.ID.GetHexString())

	InitConsensus(minerInfo, common.GlobalConf)
	group_create.GroupCreateProcessor.BeginGenesisGroupMember()
	Proc.Start()
}
