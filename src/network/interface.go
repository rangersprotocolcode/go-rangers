package network

const (
	//-----------组初始化---------------------------------

	GroupInitMsg uint32 = 1

	KeyPieceMsg uint32 = 2

	SignPubkeyMsg uint32 = 3

	GroupInitDoneMsg uint32 = 4

	//-----------组铸币---------------------------------
	CurrentGroupCastMsg uint32 = 5

	CastVerifyMsg uint32 = 6

	VerifiedCastMsg uint32 = 7

	NewBlockMsg uint32 = 8
	//--------------交易-----------------------------
	ReqTransactionMsg uint32 = 9

	TransactionGotMsg uint32 = 10

	TransactionBroadcastMsg uint32 = 11

	//-----------块同步---------------------------------
	BlockInfoNotifyMsg uint32 = 13

	ReqBlock uint32 = 14

	BlockResponseMsg uint32 = 15

	//-----------组同步---------------------------------
	GroupChainCountMsg uint32 = 16

	ReqGroupMsg uint32 = 17

	GroupMsg uint32 = 18
	//-----------块链调整---------------------------------
	ChainPieceInfoReq uint32 = 19

	ChainPieceInfo uint32 = 20

	ReqChainPieceBlock uint32 = 21

	ChainPieceBlock uint32 = 22
	//---------------------组创建确认-----------------------
	CreateGroupaRaw uint32 = 23

	CreateGroupSign uint32 = 24
	//---------------------轻节点状态同步-----------------------
	//ReqStateInfoMsg uint32 = 25
	//
	//StateInfoMsg uint32 = 26

	//==================铸块分红=========
	CastRewardSignReq uint32 = 27
	CastRewardSignGot uint32 = 28

	//==================Trace=========
	//RequestTraceMsg  uint32 = 29
	//ResponseTraceMsg uint32 = 30

	//------------------------------
	//NewBlockHeaderMsg uint32 = 31
	//
	//BlockBodyReqMsg uint32 = 32
	//
	//BlockBodyMsg               uint32 = 33
	FullNodeVirtualGroupId = "full_node_virtual_group_id"

	//===================请求组内成员签名公钥======
	AskSignPkMsg    uint32 = 34
	AnswerSignPkMsg uint32 = 35

	VerifiedCastMsg2 uint32 = 77

	//建组时ping pong
	GroupPing uint32 = 100
	GroupPong uint32 = 101

	//
	ReqSharePiece      uint32 = 102
	ResponseSharePiece uint32 = 103
)

type Conn struct {
	Id   string
	Ip   string
	Port string
}

type MsgDigest []byte

type Network interface {
	//Send message to the node which id represents.If self doesn't connect to the node,
	// resolve the kad net to find the node and then send the message
	Send(id string, msg Message)


	//Broadcast the message among the group which self belongs to
	SpreadAmongGroup(groupId string, msg Message)

	//send message to random memebers which in special group
	//groupMembers is nil here
	SpreadToRandomGroupMember(groupId string, groupMembers []string, msg Message)


	//Send message to neighbor nodes
	TransmitToNeighbor(msg Message)


	//Send the message to all nodes it connects to and the node which receive the message also broadcast the message to their neighbor once
	Broadcast(msg Message)

	//Return all connections self has
	ConnInfo() []Conn
}

func GetNetInstance() Network {
	return &Server
}

type MsgHandler interface {
	Handle(sourceId string, msg Message) error
}
