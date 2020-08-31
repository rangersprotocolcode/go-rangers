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

type BlockReqMessage struct {
	HeightByte []byte
	Peer       string
}

func (m *BlockReqMessage) GetRaw() []byte {
	return m.HeightByte
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

type BlockOnChainSuccMessage struct {
	Block types.Block
}

func (m *BlockOnChainSuccMessage) GetRaw() []byte {
	return []byte{}
}
func (m *BlockOnChainSuccMessage) GetData() interface{} {
	return m.Block
}

type BlockInfoNotifyMessage struct {
	BlockInfo []byte
	Peer      string
}

func (m *BlockInfoNotifyMessage) GetRaw() []byte {
	return m.BlockInfo
}

func (m *BlockInfoNotifyMessage) GetData() interface{} {
	return m
}

//------------------------------------------------fork------------------------------------------------------------------
type ChainPieceInfoReqMessage struct {
	HeightByte []byte
	Peer       string
}

func (m *ChainPieceInfoReqMessage) GetRaw() []byte {
	return nil
}
func (m *ChainPieceInfoReqMessage) GetData() interface{} {
	return m
}

type ChainPieceInfoMessage struct {
	ChainPieceInfoByte []byte
	Peer               string
}

func (m *ChainPieceInfoMessage) GetRaw() []byte {
	return m.ChainPieceInfoByte
}
func (m *ChainPieceInfoMessage) GetData() interface{} {
	return m
}

type ChainPieceBlockReqMessage struct {
	ReqHeightByte []byte
	Peer          string
}

func (m *ChainPieceBlockReqMessage) GetRaw() []byte {
	return m.ReqHeightByte
}
func (m *ChainPieceBlockReqMessage) GetData() interface{} {
	return m
}

type ChainPieceBlockMessage struct {
	ChainPieceBlockMsgByte []byte
	Peer                   string
}

func (m *ChainPieceBlockMessage) GetRaw() []byte {
	return m.ChainPieceBlockMsgByte
}
func (m *ChainPieceBlockMessage) GetData() interface{} {
	return m
}

//type BlockHeaderNotifyMessage struct {
//	HeaderByte []byte
//
//	Peer string
//}
//
//func (m *BlockHeaderNotifyMessage) GetRaw() []byte {
//	return nil
//}
//
//func (m *BlockHeaderNotifyMessage) GetData() interface{} {
//	return m
//}
//
//type BlockBodyReqMessage struct {
//	BlockHashByte []byte
//
//	Peer string
//}
//
//func (m *BlockBodyReqMessage) GetRaw() []byte {
//	return nil
//}
//
//func (m *BlockBodyReqMessage) GetData() interface{} {
//	return m
//}
//
//type BlockBodyNotifyMessage struct {
//	BodyByte []byte
//
//	Peer string
//}
//
//func (m *BlockBodyNotifyMessage) GetRaw() []byte {
//	return nil
//}
//func (m *BlockBodyNotifyMessage) GetData() interface{} {
//	return m
//}

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

type GroupReqMessage struct {
	GroupIdByte []byte
	Peer        string
}

func (m *GroupReqMessage) GetRaw() []byte {
	return m.GroupIdByte
}
func (m *GroupReqMessage) GetData() interface{} {
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

//type StateInfoReqMessage struct {
//	StateInfoReqByte []byte
//
//	Peer string
//}
//
//func (m *StateInfoReqMessage) GetRaw() []byte {
//	return nil
//}
//func (m *StateInfoReqMessage) GetData() interface{} {
//	return m
//}
//
//type StateInfoMessage struct {
//	StateInfoByte []byte
//
//	Peer string
//}
//
//func (m *StateInfoMessage) GetRaw() []byte {
//	return nil
//}
//func (m *StateInfoMessage) GetData() interface{} {
//	return m
//}

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

type CoinProxyNotifyMessage struct {
	Tx types.Transaction
}

func (m *CoinProxyNotifyMessage) GetRaw() []byte {
	// never use it
	return nil
}
func (m *CoinProxyNotifyMessage) GetData() interface{} {
	return m
}

type STMStorageReadyMessage struct {
	FileName []byte
}

func (s *STMStorageReadyMessage) GetRaw() []byte {
	return s.FileName
}
func (s *STMStorageReadyMessage) GetData() interface{} {
	return s.FileName
}

type NonceNotifyMessage struct {
	Nonce uint64
}

func (m *NonceNotifyMessage) GetRaw() []byte {
	// never use it
	return nil
}
func (m *NonceNotifyMessage) GetData() interface{} {
	return m
}
