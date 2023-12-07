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

package notify

import (
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/json"
	"strconv"
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

// -------------------------------------sync---------------------
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

// --------------------------------------------------------------------------------------------------------------------
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

// ---------------------------------------------------------------------------------------------------------------------
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
	Tx        types.Transaction
	UserId    string
	Nonce     uint64
	GateNonce uint64
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
	result["gateNonce"] = strconv.FormatUint(m.GateNonce, 10)
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

type ETHRPCWrongMessage struct {
	Rid uint64
	Sid string
}

func (m *ETHRPCWrongMessage) GetRaw() []byte {
	return nil
}
func (m *ETHRPCWrongMessage) GetData() interface{} {
	return m
}

type ETHRPCMessage struct {
	Message ETHRPCPiece `json:"jsonrpc"`

	GateNonce uint64 `json:"request_id"`
	SessionId string `json:"session_id"`
}

func (m *ETHRPCMessage) GetRaw() []byte {
	return nil
}
func (m *ETHRPCMessage) GetData() interface{} {
	return m
}

type ETHRPCBatchMessage struct {
	Message []ETHRPCPiece `json:"jsonrpc"`

	GateNonce uint64 `json:"request_id"`
	SessionId string `json:"session_id"`
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

// ------------------------------------------notify message----------------------------------------------------------------------
type VMEventNotifyMessage struct {
	Logs []*types.Log
}

func (m *VMEventNotifyMessage) GetRaw() []byte {
	// never use it
	return nil
}
func (m *VMEventNotifyMessage) GetData() interface{} {
	return m.Logs
}

type BlockHeaderNotifyMessage struct {
	BlockHeader *types.BlockHeader
}

func (m *BlockHeaderNotifyMessage) GetRaw() []byte {
	// never use it
	return nil
}
func (m *BlockHeaderNotifyMessage) GetData() interface{} {
	return m.BlockHeader
}

type VMRemovedEventNotifyMessage struct {
	Logs []*types.Log
}

func (m *VMRemovedEventNotifyMessage) GetRaw() []byte {
	// never use it
	return nil
}
func (m *VMRemovedEventNotifyMessage) GetData() interface{} {
	return m.Logs
}
