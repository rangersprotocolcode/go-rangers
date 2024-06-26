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

package network

const (
	GroupInitMsg uint32 = 1

	KeyPieceMsg uint32 = 2

	SignPubkeyMsg uint32 = 3

	GroupInitDoneMsg uint32 = 4

	CurrentGroupCastMsg uint32 = 5

	CastVerifyMsg uint32 = 6

	VerifiedCastMsg uint32 = 36

	NewBlockMsg uint32 = 8

	ReqTransactionMsg uint32 = 9

	TransactionGotMsg uint32 = 10

	TopBlockInfoMsg uint32 = 12

	BlockChainPieceReqMsg uint32 = 13

	BlockChainPieceMsg uint32 = 14

	ReqBlockMsg uint32 = 15

	BlockResponseMsg uint32 = 16

	ReqGroupMsg uint32 = 19

	GroupResponseMsg uint32 = 20

	CreateGroupaRaw uint32 = 22

	CreateGroupSign uint32 = 23

	AskSignPkMsg    uint32 = 34
	AnswerSignPkMsg uint32 = 35

	GroupPing uint32 = 37
	GroupPong uint32 = 38

	ReqSharePiece      uint32 = 39
	ResponseSharePiece uint32 = 40
)

type MsgDigest []byte

type Network interface {
	// Send id is given by gateway
	Send(id string, msg Message)

	SpreadToGroup(groupId string, msg Message)

	Broadcast(msg Message)

	SendToJSONRPC(msg []byte, sessionId string, requestId uint64)

	SendToClientReader(id string, msg []byte, nonce uint64)

	SendToClientWriter(id string, msg []byte, nonce uint64)

	Init(gateAddr, outerGateAddr string, selfMinerId []byte, consensusHandler MsgHandler, isSending bool)

	InitTx(tx string)

	JoinGroupNet(groupId string)

	QuitGroupNet(groupId string)

	// SendToStranger strangerId equals to minerId
	SendToStranger(strangerId []byte, msg Message)
}

func GetNetInstance() Network {
	return &instance
}

type MsgHandler interface {
	Handle(sourceId string, msg Message) error
}
