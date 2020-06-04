package network

import "x/src/middleware/log"

const (
	//-----------组初始化---------------------------------

	GroupInitMsg uint32 = 1

	KeyPieceMsg uint32 = 2

	SignPubkeyMsg uint32 = 3

	GroupInitDoneMsg uint32 = 4

	//-----------组铸币---------------------------------
	CurrentGroupCastMsg uint32 = 5

	// 提案者发送候选块，待验证
	CastVerifyMsg uint32 = 6

	// 验证组内，验证完块后，发送签名片段
	VerifiedCastMsg uint32 = 36

	NewBlockMsg uint32 = 8
	//--------------交易-----------------------------
	ReqTransactionMsg uint32 = 9

	TransactionGotMsg uint32 = 10

	TransactionBroadcastMsg uint32 = 11

	//-----------块同步---------------------------------
	BlockInfoNotifyMsg uint32 = 12

	ReqBlock uint32 = 13

	BlockResponseMsg uint32 = 14

	//-----------组同步---------------------------------
	GroupChainCountMsg uint32 = 15

	ReqGroupMsg uint32 = 16

	GroupMsg uint32 = 17
	//-----------块链调整---------------------------------
	ChainPieceInfoReq uint32 = 18

	ChainPieceInfo uint32 = 19

	ReqChainPieceBlock uint32 = 20

	ChainPieceBlock uint32 = 21
	//---------------------组创建确认-----------------------
	CreateGroupaRaw uint32 = 22

	CreateGroupSign uint32 = 23

	//===================请求组内成员签名公钥======
	AskSignPkMsg    uint32 = 34
	AnswerSignPkMsg uint32 = 35

	//建组时ping pong
	GroupPing uint32 = 37
	GroupPong uint32 = 38

	ReqSharePiece      uint32 = 39
	ResponseSharePiece uint32 = 40

	//-----------stm状态通知---------------------------------
	STMStorageReady uint32 = 50
)

//与coin connector 通信的消息CODE
const (
	CoinProxyNotify uint32 = 1000
	WithDraw        uint32 = 1001
	AssetOnChain           = 1002
)

type MsgDigest []byte

type Network interface {
	Send(id string, msg Message)

	SpreadToGroup(groupId string, msg Message)

	Broadcast(msg Message)

	SendToClientReader(id string, msg []byte, nonce uint64)

	SendToClientWriter(id string, msg []byte, nonce uint64)

	SendToCoinConnector(msg []byte)

	Notify(isunicast bool, gameId string, userid string, msg string)

	Init(logger log.Logger, gateAddr, selfMinerId string, consensusHandler MsgHandler)

	JoinGroupNet(groupId string)

	QuitGroupNet(groupId string)

	SetNetId(netId []byte)

	SendToStranger(strangerId []byte, msg Message)
}

func GetNetInstance() Network {
	return &instance
}

type MsgHandler interface {
	Handle(sourceId string, msg Message) error
}
