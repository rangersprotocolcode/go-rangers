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

package network

import (
	"com.tuntun.rocket/node/src/middleware/log"
)

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

	//-----------同步---------------------------------
	TopBlockInfoMsg uint32 = 12

	BlockChainPieceReqMsg uint32 = 13

	BlockChainPieceMsg uint32 = 14

	ReqBlockMsg uint32 = 15

	BlockResponseMsg uint32 = 16

	ReqGroupMsg uint32 = 19

	GroupResponseMsg uint32 = 20

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

	TxReceived uint32 = 99
)

type MsgDigest []byte

type Network interface {
	Send(id string, msg Message)

	SpreadToGroup(groupId string, msg Message)

	Broadcast(msg Message)

	SendToJSONRPC(msg []byte, sessionId string, requestId uint64)

	SendToClientReader(id string, msg []byte, nonce uint64)

	SendToClientWriter(id string, msg []byte, nonce uint64)

	Init(logger log.Logger, gateAddr, outerGateAddr string, selfMinerId []byte, consensusHandler MsgHandler, isSending bool)

	JoinGroupNet(groupId string)

	QuitGroupNet(groupId string)

	SendToStranger(strangerId []byte, msg Message)
}

func GetNetInstance() Network {
	return &instance
}

type MsgHandler interface {
	Handle(sourceId string, msg Message) error
}
