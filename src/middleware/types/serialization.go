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

package types

import (
	middleware_pb "com.tuntun.rangers/node/src/middleware/pb"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"strconv"

	"com.tuntun.rangers/node/src/common"
	"math/big"
	"time"

	"com.tuntun.rangers/node/src/middleware/log"
	"encoding/json"
)

var logger log.Logger

func InitSerialzation() {
	logger = log.GetLoggerByIndex(log.MiddlewareLogConfig, strconv.Itoa(common.InstanceIndex))
}

func UnMarshalTransaction(b []byte) (Transaction, error) {
	t := new(middleware_pb.Transaction)
	error := proto.Unmarshal(b, t)
	if error != nil {
		logger.Errorf("Unmarshal transaction error:%s", error.Error())
		return Transaction{}, error
	}
	transaction := pbToTransaction(t)
	return transaction, nil
}

func UnMarshalTransactions(b []byte) ([]*Transaction, error) {
	ts := new(middleware_pb.TransactionSlice)
	error := proto.Unmarshal(b, ts)
	if error != nil {
		logger.Errorf("Unmarshal transactions error:%s", error.Error())
		return nil, error
	}

	result := PbToTransactions(ts.Transactions)
	return result, nil
}

func UnMarshalBlock(bytes []byte) (*Block, error) {
	b := new(middleware_pb.Block)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		logger.Errorf("Unmarshal Block error:%s", error.Error())
		return nil, error
	}
	block := PbToBlock(b)
	return block, nil
}

func UnMarshalBlockHeader(bytes []byte) (*BlockHeader, error) {
	b := new(middleware_pb.BlockHeader)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		logger.Errorf("Unmarshal Block error:%s", error.Error())
		return nil, error
	}
	header := PbToBlockHeader(b)
	return header, nil
}

func UnMarshalMember(b []byte) (*Member, error) {
	member := new(middleware_pb.Member)
	e := proto.Unmarshal(b, member)
	if e != nil {
		logger.Errorf("UnMarshalMember error:%s\n", e.Error())
		return nil, e
	}
	m := pbToMember(member)
	return m, nil
}

func UnMarshalGroup(b []byte) (*Group, error) {
	group := new(middleware_pb.Group)
	e := proto.Unmarshal(b, group)
	if e != nil {
		logger.Errorf("UnMarshalGroup error:%s\n", e.Error())
		return nil, e
	}
	g := PbToGroup(group)
	return g, nil
}

func MarshalTransaction(t *Transaction) ([]byte, error) {
	transaction := transactionToPb(t)
	return proto.Marshal(transaction)
}

func MarshalTransactions(txs []*Transaction) ([]byte, error) {
	transactions := TransactionsToPb(txs)
	transactionSlice := middleware_pb.TransactionSlice{Transactions: transactions}
	return proto.Marshal(&transactionSlice)
}

func MarshalBlock(b *Block) ([]byte, error) {
	block := BlockToPb(b)
	if block == nil {
		return nil, nil
	}
	return proto.Marshal(block)
}

func MarshalBlockHeader(b *BlockHeader) ([]byte, error) {
	block := BlockHeaderToPb(b)
	if block == nil {
		return nil, nil
	}
	return proto.Marshal(block)
}

func MarshalMember(m *Member) ([]byte, error) {
	member := memberToPb(m)
	return proto.Marshal(member)
}

func MarshalGroup(g *Group) ([]byte, error) {
	group := GroupToPb(g)
	return proto.Marshal(group)
}

func pbToTransaction(t *middleware_pb.Transaction) Transaction {
	if t == nil {
		return Transaction{}
	}

	var source, target, data, socketRequestId string
	var sign *common.Sign
	var subTransactions []UserData
	if t.Source != nil {
		source = string(t.Source)
	}
	if t.Target != nil {

		target = *t.Target
	}
	if t.Data != nil {

		data = *t.Data
	}

	if t.SocketRequestId != nil {
		socketRequestId = *t.SocketRequestId
	}

	if t.SubTransactions != nil {
		json.Unmarshal(t.SubTransactions, &subTransactions)
	}

	if t.Sign != nil && len(t.Sign) != 0 {
		if len(t.Sign) != 65 {
			fmt.Printf("Bad sign hash:%v, sign:%v\n", common.BytesToHash(t.Hash), t.Sign)
		}
		sign = common.BytesToSign(t.Sign)
	}

	transaction := Transaction{Data: data, Nonce: *t.Nonce, RequestId: *t.RequestId, Source: source,
		Target: target, Hash: common.BytesToHash(t.Hash),
		ExtraData: string(t.ExtraData), ExtraDataType: *t.ExtraDataType, Type: *t.Type, Sign: sign,
		Time: *t.Time, SocketRequestId: socketRequestId, SubTransactions: subTransactions, SubHash: common.BytesToHash(t.SubHash), ChainId: *t.ChainId}

	return transaction
}

func PbToTransactions(txs []*middleware_pb.Transaction) []*Transaction {
	result := make([]*Transaction, 0)
	if txs == nil {
		return result
	}
	for _, t := range txs {
		transaction := pbToTransaction(t)
		result = append(result, &transaction)
	}
	return result
}

func PbToBlockHeader(h *middleware_pb.BlockHeader) *BlockHeader {
	if h == nil {
		return nil
	}
	hashBytes := h.Transactions
	hashes := make([]common.Hashes, 0, len(hashBytes))

	if hashBytes != nil {
		for _, hashByte := range hashBytes {
			hash := common.Hashes{}
			hash[0] = common.BytesToHash(hashByte.Hash)
			hash[1] = common.BytesToHash(hashByte.SubHash)
			hashes = append(hashes, hash)

		}
	}

	hashBytes2 := h.EvictedTxs
	hashes2 := make([]common.Hash, 0)

	if hashBytes2 != nil {
		for _, hashByte := range hashBytes2.Hashes {
			hash := common.BytesToHash(hashByte)
			hashes2 = append(hashes2, hash)
		}
	}

	var preTime time.Time
	e1 := preTime.UnmarshalBinary(h.PreTime)
	if e1 != nil {
		logger.Errorf("pbToBlockHeader preTime UnmarshalBinary error:%s", e1.Error())
		return nil
	}

	var curTime time.Time
	curTime.UnmarshalBinary(h.CurTime)
	e2 := curTime.UnmarshalBinary(h.CurTime)
	if e2 != nil {
		logger.Errorf("pbToBlockHeader curTime UnmarshalBinary error:%s", e2.Error())
		return nil
	}

	pv := &big.Int{}
	var proveValue *big.Int
	if h.ProveValue != nil {
		proveValue = pv.SetBytes(h.ProveValue)
	} else {
		proveValue = nil
	}
	//log.Printf("PbToBlockHeader height:%d StateTree Hash:%s",*h.Height,common.Bytes2Hex(h.StateTree))
	header := BlockHeader{Hash: common.BytesToHash(h.Hash), Height: *h.Height, PreHash: common.BytesToHash(h.PreHash), PreTime: preTime,
		ProveValue: proveValue, CurTime: curTime, Castor: h.Castor, GroupId: h.GroupId, Signature: h.Signature,
		Nonce: *h.Nonce, Transactions: hashes, TxTree: common.BytesToHash(h.TxTree), ReceiptTree: common.BytesToHash(h.ReceiptTree), StateTree: common.BytesToHash(h.StateTree),
		ExtraData: h.ExtraData, TotalQN: *h.TotalQN, Random: h.Random, EvictedTxs: hashes2}

	if nil != h.RequestIds {
		json.Unmarshal(h.RequestIds, &header.RequestIds)
	}
	return &header
}

func GroupRequestInfoToPB(CurrentTopGroupId []byte, ExistGroupIds [][]byte) *middleware_pb.GroupRequestInfo {
	return &middleware_pb.GroupRequestInfo{CurrentTopGroupId: CurrentTopGroupId, ExistGroupIds: &middleware_pb.GroupIdSlice{GroupIds: ExistGroupIds}}
}

func PbToBlock(b *middleware_pb.Block) *Block {
	if b == nil {
		return nil
	}
	h := PbToBlockHeader(b.Header)
	txs := PbToTransactions(b.Transactions)
	block := Block{Header: h, Transactions: txs}
	return &block
}

func PbToGroupHeader(g *middleware_pb.GroupHeader) *GroupHeader {
	var beginTime time.Time
	beginTime.UnmarshalBinary(g.BeginTime)
	header := GroupHeader{
		Hash:            common.BytesToHash(g.Hash),
		Parent:          g.Parent,
		PreGroup:        g.PreGroup,
		BeginTime:       beginTime,
		MemberRoot:      common.BytesToHash(g.MemberRoot),
		CreateHeight:    *g.CreateHeight,
		Extends:         *g.Extends,
		CreateBlockHash: g.CreateBlockHash,
	}
	return &header
}

func PbToGroup(g *middleware_pb.Group) *Group {
	if g == nil {
		return nil
	}
	group := Group{
		Header:      PbToGroupHeader(g.Header),
		Id:          g.Id,
		Members:     g.Members,
		PubKey:      g.PubKey,
		Signature:   g.Signature,
		GroupHeight: *g.GroupHeight,
	}
	return &group
}

func PbToGroups(g *middleware_pb.GroupSlice) []*Group {
	result := make([]*Group, 0)
	for _, group := range g.Groups {
		result = append(result, PbToGroup(group))
	}
	return result
}

func pbToMember(m *middleware_pb.Member) *Member {
	member := Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}

func transactionToPb(t *Transaction) *middleware_pb.Transaction {
	if t == nil {
		return nil
	}
	var (
		target *string
		data   *string
		source []byte
		sign   []byte
	)
	if len(t.Target) != 0 {
		target = &t.Target
	}
	if len(t.Data) != 0 {
		data = &t.Data
	}
	if len(t.Source) != 0 {
		source = []byte(t.Source)
	}

	if t.Sign != nil {
		sign = t.Sign.Bytes()
		if len(sign) != 65 {
			logger.Errorf("Bad sign len:%d", len(sign))
		}
	}

	subTx, _ := json.Marshal(t.SubTransactions)
	transaction := middleware_pb.Transaction{Data: data, Nonce: &t.Nonce, RequestId: &t.RequestId, Source: source,
		Target: target, Hash: t.Hash.Bytes(),
		ExtraData: []byte(t.ExtraData), ExtraDataType: &t.ExtraDataType, Type: &t.Type, Sign: sign,
		Time: &t.Time, SubTransactions: subTx, SubHash: t.SubHash.Bytes(), ChainId: &t.ChainId}
	return &transaction
}

func TransactionsToPb(txs []*Transaction) []*middleware_pb.Transaction {
	if txs == nil {
		return nil
	}
	transactions := make([]*middleware_pb.Transaction, 0)
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

func BlockHeaderToPb(h *BlockHeader) *middleware_pb.BlockHeader {
	hashes := h.Transactions
	txHashes := make([]*middleware_pb.TransactionHash, 0)

	if hashes != nil {
		for _, hash := range hashes {
			txHash := &middleware_pb.TransactionHash{}
			txHash.Hash = hash[0].Bytes()
			txHash.SubHash = hash[1].Bytes()
			txHashes = append(txHashes, txHash)
		}
	}

	hashes2 := h.EvictedTxs
	hashBytes2 := make([][]byte, 0)

	if hashes2 != nil {
		for _, hash := range hashes2 {
			hashBytes2 = append(hashBytes2, hash.Bytes())
		}
	}
	evictedTxs := middleware_pb.Hashes{Hashes: hashBytes2}
	preTime, e1 := h.PreTime.MarshalBinary()
	if e1 != nil {
		logger.Errorf("BlockHeaderToPb marshal pre time error:%s\n", e1.Error())
		return nil
	}

	curTime, e2 := h.CurTime.MarshalBinary()
	if e2 != nil {
		logger.Errorf("BlockHeaderToPb marshal cur time error:%s", e2.Error())
		return nil
	}

	var proveValueByte []byte
	if h.ProveValue != nil {
		proveValueByte = h.ProveValue.Bytes()
	} else {
		proveValueByte = nil
	}

	header := middleware_pb.BlockHeader{Hash: h.Hash.Bytes(), Height: &h.Height, PreHash: h.PreHash.Bytes(), PreTime: preTime,
		ProveValue: proveValueByte, CurTime: curTime, Castor: h.Castor, GroupId: h.GroupId, Signature: h.Signature,
		Nonce: &h.Nonce, Transactions: txHashes, TxTree: h.TxTree.Bytes(), ReceiptTree: h.ReceiptTree.Bytes(), StateTree: h.StateTree.Bytes(),
		ExtraData: h.ExtraData, TotalQN: &h.TotalQN, Random: h.Random, EvictedTxs: &evictedTxs}

	header.RequestIds, _ = json.Marshal(h.RequestIds)
	return &header
}

func BlockToPb(b *Block) *middleware_pb.Block {
	if b == nil {
		return nil
	}
	header := BlockHeaderToPb(b.Header)
	transactions := TransactionsToPb(b.Transactions)
	block := middleware_pb.Block{Header: header, Transactions: transactions}
	return &block
}

func GroupToPbHeader(g *GroupHeader) *middleware_pb.GroupHeader {
	beginTime, _ := g.BeginTime.MarshalBinary()
	header := middleware_pb.GroupHeader{
		Hash:            g.Hash.Bytes(),
		Parent:          g.Parent,
		PreGroup:        g.PreGroup,
		BeginTime:       beginTime,
		MemberRoot:      g.MemberRoot.Bytes(),
		CreateHeight:    &g.CreateHeight,
		Extends:         &g.Extends,
		CreateBlockHash: g.CreateBlockHash,
	}
	return &header
}

func GroupToPb(g *Group) *middleware_pb.Group {
	//members := make([]*middleware_pb.Member, 0)
	//for _, m := range g.Members {
	//	member := memberToPb(&m)
	//	members = append(members, member)
	//}
	if g == nil {
		return nil
	}
	group := middleware_pb.Group{
		Header:      GroupToPbHeader(g.Header),
		Id:          g.Id,
		Members:     g.Members,
		PubKey:      g.PubKey,
		Signature:   g.Signature,
		GroupHeight: &g.GroupHeight,
	}
	return &group
}

func memberToPb(m *Member) *middleware_pb.Member {
	member := middleware_pb.Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}
