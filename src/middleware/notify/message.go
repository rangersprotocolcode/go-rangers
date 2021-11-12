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

package notify

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"encoding/json"
)

type NewBlockMessage struct {
	BlockByte []byte
	Peer      string
}

func (m *NewBlockMessage) GetRaw() []byte {
	return m.BlockByte
}
func (m *NewBlockMessage) GetData() interface{} {
	return m
}

type BlockOnChainSuccMessage struct {
	Block types.Block
}

func (m *BlockOnChainSuccMessage) GetRaw() []byte {
	return []byte{}
}
func (m *BlockOnChainSuccMessage) GetData() interface{} {
	return m.Block
}

//-------------------------------------sync---------------------
type ChainInfoMessage struct {
	ChainInfo []byte
	Peer      string
}

func (m *ChainInfoMessage) GetRaw() []byte {
	return m.ChainInfo
}

func (m *ChainInfoMessage) GetData() interface{} {
	return m
}

type BlockChainPieceReqMessage struct {
	BlockChainPieceReq []byte
	Peer               string
}

func (m *BlockChainPieceReqMessage) GetRaw() []byte {
	return nil
}
func (m *BlockChainPieceReqMessage) GetData() interface{} {
	return m
}

type BlockChainPieceMessage struct {
	BlockChainPieceByte []byte
	Peer                string
}

func (m *BlockChainPieceMessage) GetRaw() []byte {
	return m.BlockChainPieceByte
}
func (m *BlockChainPieceMessage) GetData() interface{} {
	return m
}

type BlockReqMessage struct {
	ReqInfoByte []byte
	Peer        string
}

func (m *BlockReqMessage) GetRaw() []byte {
	return m.ReqInfoByte
}
func (m *BlockReqMessage) GetData() interface{} {
	return m
}

type BlockResponseMessage struct {
	BlockResponseByte []byte
	Peer              string
}

func (m *BlockResponseMessage) GetRaw() []byte {
	return m.BlockResponseByte
}
func (m *BlockResponseMessage) GetData() interface{} {
	return m
}

type GroupReqMessage struct {
	ReqInfoByte []byte
	Peer        string
}

func (m *GroupReqMessage) GetRaw() []byte {
	return m.ReqInfoByte
}
func (m *GroupReqMessage) GetData() interface{} {
	return m
}

type GroupResponseMessage struct {
	GroupResponseByte []byte
	Peer              string
}

func (m *GroupResponseMessage) GetRaw() []byte {
	return m.GroupResponseByte
}
func (m *GroupResponseMessage) GetData() interface{} {
	return m
}

//--------------------------------------------------------------------------------------------------------------------
type GroupMessage struct {
	Group types.Group
}

func (m *GroupMessage) GetRaw() []byte {
	return []byte{}
}
func (m *GroupMessage) GetData() interface{} {
	return m.Group
}

type GroupHeightMessage struct {
	HeightByte []byte
	Peer       string
}

func (m *GroupHeightMessage) GetRaw() []byte {
	return m.HeightByte
}
func (m *GroupHeightMessage) GetData() interface{} {
	return m
}

type GroupInfoMessage struct {
	GroupInfoByte []byte
	Peer          string
}

func (m *GroupInfoMessage) GetRaw() []byte {
	return m.GroupInfoByte
}
func (m *GroupInfoMessage) GetData() interface{} {
	return m
}

//---------------------------------------------------------------------------------------------------------------------
type TransactionBroadcastMessage struct {
	TransactionsByte []byte
	Peer             string
}

func (m *TransactionBroadcastMessage) GetRaw() []byte {
	return m.TransactionsByte
}
func (m *TransactionBroadcastMessage) GetData() interface{} {
	return m
}

type TransactionReqMessage struct {
	TransactionReqByte []byte
	Peer               string
}

func (m *TransactionReqMessage) GetRaw() []byte {
	return m.TransactionReqByte
}
func (m *TransactionReqMessage) GetData() interface{} {
	return m
}

type TransactionGotMessage struct {
	TransactionGotByte []byte
	Peer               string
}

func (m *TransactionGotMessage) GetRaw() []byte {
	return m.TransactionGotByte
}
func (m *TransactionGotMessage) GetData() interface{} {
	return m
}

type TransactionGotAddSuccMessage struct {
	Transactions []*types.Transaction
	Peer         string
}

func (m *TransactionGotAddSuccMessage) GetRaw() []byte {
	return nil
}
func (m *TransactionGotAddSuccMessage) GetData() interface{} {
	return m.Transactions
}

type ClientTransactionMessage struct {
	Tx     types.Transaction
	UserId string
	Nonce  uint64
}

func (m *ClientTransactionMessage) GetRaw() []byte {
	// never use it
	return nil
}
func (m *ClientTransactionMessage) GetData() interface{} {
	return m
}

func (m *ClientTransactionMessage) TOJSONString() string {
	result := make(map[string]string, 0)
	result["tx"] = m.Tx.ToTxJson().ToString()
	result["userId"] = m.UserId

	byte, _ := json.Marshal(result)
	return string(byte)
}

type NonceNotifyMessage struct {
	Nonce uint64
	Msg   string
}

func (m *NonceNotifyMessage) GetRaw() []byte {
	// never use it
	return nil
}
func (m *NonceNotifyMessage) GetData() interface{} {
	return m
}

type ETHRPCMessage struct {
	Message ETHRPCPiece `json:"jsonrpc"`

	RequestId uint64 `json:"request_id"`
	SessionId uint64 `json:"session_id"`
}

func (m *ETHRPCMessage) GetRaw() []byte {
	return nil
}
func (m *ETHRPCMessage) GetData() interface{} {
	return m
}

type ETHRPCBatchMessage struct {
	Message []ETHRPCPiece `json:"jsonrpc"`

	RequestId uint64 `json:"request_id"`
	SessionId uint64 `json:"session_id"`
}

func (m *ETHRPCBatchMessage) GetRaw() []byte {
	return nil
}
func (m *ETHRPCBatchMessage) GetData() interface{} {
	return m
}

type ETHRPCPiece struct {
	Id     interface{}     `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`

	Nonce uint64 `json:"nonce"`
}
