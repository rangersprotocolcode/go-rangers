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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package net

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	middleware_pb "com.tuntun.rangers/node/src/middleware/pb"
	"com.tuntun.rangers/node/src/middleware/types"
	"github.com/gogo/protobuf/proto"
	"time"
)

func baseMessage(sign *middleware_pb.SignData) *model.SignInfo {
	return pbToSignData(sign)
}

func pbToGroupInfo(gi *middleware_pb.ConsensusGroupInitInfo) *model.GroupInitInfo {
	mems := make([]groupsig.ID, len(gi.Mems))
	for idx, mem := range gi.Mems {
		mems[idx] = groupsig.DeserializeID(mem)
	}
	groupHeader := types.PbToGroupHeader(gi.Header)
	return &model.GroupInitInfo{
		GroupHeader:     groupHeader,
		ParentGroupSign: *groupsig.DeserializeSign(gi.Signature),
		GroupMembers:    mems,
	}
}

func unMarshalConsensusGroupRawMessage(b []byte) (*model.GroupInitMessage, error) {
	message := new(middleware_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	m := model.GroupInitMessage{
		GroupInitInfo: *pbToGroupInfo(message.GInfo),
		SignInfo:      *baseMessage(message.Sign),
	}
	return &m, nil
}

func unMarshalConsensusSharePieceMessage(b []byte) (*model.SharePieceMessage, error) {
	m := new(middleware_pb.ConsensusSharePieceMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	gHash := common.BytesToHash(m.GHash)

	dest := groupsig.DeserializeID(m.Dest)

	share := pbToSharePiece(m.SharePiece)
	message := model.SharePieceMessage{
		GroupHash:      gHash,
		ReceiverId:     dest,
		Share:          *share,
		SignInfo:       *baseMessage(m.Sign),
		GroupMemberNum: *m.MemCnt,
	}
	return &message, nil
}

func unMarshalConsensusSignPubKeyMessage(b []byte) (*model.SignPubKeyMessage, error) {
	m := new(middleware_pb.ConsensusSignPubKeyMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e.Error())
		return nil, e
	}
	gisHash := common.BytesToHash(m.GHash)

	pk := groupsig.ByteToPublicKey(m.SignPK)

	base := baseMessage(m.SignData)
	message := model.SignPubKeyMessage{
		GroupHash:      gisHash,
		SignPK:         pk,
		GroupID:        groupsig.DeserializeID(m.GroupID),
		SignInfo:       *base,
		GroupMemberNum: *m.MemCnt,
	}
	return &message, nil
}

func unMarshalConsensusGroupInitedMessage(b []byte) (*model.GroupInitedMessage, error) {
	m := new(middleware_pb.ConsensusGroupInitedMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusGroupInitedMessage error:%s", e.Error())
		return nil, e
	}

	ch := uint64(0)
	if m.CreateHeight != nil {
		ch = *m.CreateHeight
	}
	var sign groupsig.Signature
	if len(m.ParentSign) > 0 {
		sign.Deserialize(m.ParentSign)
	}
	message := model.GroupInitedMessage{
		GroupHash:       common.BytesToHash(m.GHash),
		GroupID:         groupsig.DeserializeID(m.GroupID),
		GroupPK:         groupsig.ByteToPublicKey(m.GroupPK),
		CreateHeight:    ch,
		ParentGroupSign: sign,
		SignInfo:        *baseMessage(m.Sign),
		MemberNum:       *m.MemCnt,
		MemberMask:      m.MemMask,
	}
	return &message, nil
}

func unMarshalConsensusSignPKReqMessage(b []byte) (*model.SignPubkeyReqMessage, error) {
	m := new(middleware_pb.ConsensusSignPubkeyReqMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPKReqMessage error: %v", e.Error())
		return nil, e
	}
	message := &model.SignPubkeyReqMessage{
		GroupID:  groupsig.DeserializeID(m.GroupID),
		SignInfo: *baseMessage(m.SignData),
	}
	return message, nil
}

func unMarshalConsensusCurrentMessage(b []byte) (*model.ConsensusCurrentMessage, error) {
	m := new(middleware_pb.ConsensusCurrentMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCurrentMessage error:%s", e.Error())
		return nil, e
	}

	GroupID := m.GroupID
	PreHash := common.BytesToHash(m.PreHash)

	var PreTime time.Time
	PreTime.UnmarshalBinary(m.PreTime)

	BlockHeight := m.BlockHeight
	si := pbToSignData(m.Sign)
	//base := model.BaseSignedMessage{SI: *si}
	message := model.ConsensusCurrentMessage{GroupID: GroupID, PreHash: PreHash, PreTime: PreTime, BlockHeight: *BlockHeight, SignInfo: *si}
	return &message, nil
}

func UnMarshalConsensusCastMessage(b []byte) (*model.ConsensusCastMessage, error) {
	m := new(middleware_pb.ConsensusCastMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusCastMessage error:%s", e.Error())
		return nil, e
	}

	bh := types.PbToBlockHeader(m.Bh)

	hashs := make([]common.Hash, len(m.ProveHash))
	for i, h := range m.ProveHash {
		hashs[i] = common.BytesToHash(h)
	}

	return &model.ConsensusCastMessage{
		BH:        *bh,
		ProveHash: hashs,
		SignInfo:  *baseMessage(m.Sign),
		Id:        common.ToHex(common.Sha256(b)),
	}, nil
}

func UnMarshalConsensusVerifyMessage(b []byte) (*model.ConsensusVerifyMessage, error) {
	m := new(middleware_pb.ConsensusVerifyMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("unMarshalConsensusVerifyMessage error:%v", e.Error())
		return nil, e
	}
	return &model.ConsensusVerifyMessage{
		BlockHash:  common.BytesToHash(m.BlockHash),
		RandomSign: *groupsig.DeserializeSign(m.RandomSign),
		SignInfo:   *baseMessage(m.Sign),
		Id:         common.ToHex(common.Sha256(b)),
	}, nil
}

func unMarshalConsensusBlockMessage(b []byte) (*model.ConsensusBlockMessage, error) {
	m := new(middleware_pb.ConsensusBlockMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusBlockMessage error:%s", e.Error())
		return nil, e
	}
	block := types.PbToBlock(m.Block)
	message := model.ConsensusBlockMessage{Block: *block}
	return &message, nil
}

func pbToSignData(s *middleware_pb.SignData) *model.SignInfo {

	var sig groupsig.Signature
	e := sig.Deserialize(s.DataSign)
	if e != nil {
		logger.Errorf("[handler]groupsig.Signature Deserialize error:%s", e.Error())
		return nil
	}

	id := groupsig.ID{}
	e1 := id.Deserialize(s.SignMember)
	if e1 != nil {
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	v := int32(0)
	if s.Version != nil {
		v = *s.Version
	}
	sign := model.MakeSignInfo(common.BytesToHash(s.DataHash), sig, id, v)
	return &sign
}

func pbToSharePiece(s *middleware_pb.SharePiece) *model.SharePiece {
	var share groupsig.Seckey
	var pub groupsig.Pubkey

	e1 := share.Deserialize(s.Seckey)
	if e1 != nil {
		logger.Errorf("[handler]groupsig.Seckey Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pub.Deserialize(s.Pubkey)
	if e2 != nil {
		logger.Errorf("[handler]groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	sp := model.SharePiece{Share: share, Pub: pub}
	return &sp
}

func unMarshalConsensusCreateGroupRawMessage(b []byte) (*model.ParentGroupConsensusMessage, error) {
	message := new(middleware_pb.ConsensusCreateGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToGroupInfo(message.GInfo)

	m := model.ParentGroupConsensusMessage{
		GroupInitInfo: *gi,
		SignInfo:      *baseMessage(message.Sign),
	}
	return &m, nil
}

func unMarshalConsensusCreateGroupSignMessage(b []byte) (*model.ParentGroupConsensusSignMessage, error) {
	message := new(middleware_pb.ConsensusCreateGroupSignMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupSignMessage error:%s", e.Error())
		return nil, e
	}

	m := model.ParentGroupConsensusSignMessage{
		GroupHash: common.BytesToHash(message.GHash),
		SignInfo:  *baseMessage(message.Sign),
	}
	return &m, nil
}

func unMarshalCreateGroupPingMessage(b []byte) (*model.CreateGroupPingMessage, error) {
	message := new(middleware_pb.CreateGroupPingMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]unMarshalCreateGroupPingMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	//base := model.BaseSignedMessage{SI: *sign}

	m := &model.CreateGroupPingMessage{
		SignInfo:    *sign,
		FromGroupID: groupsig.DeserializeID(message.FromGroupID),
		PingID:      *message.PingID,
		BaseHeight:  *message.BaseHeight,
	}
	return m, nil
}

func unMarshalCreateGroupPongMessage(b []byte) (*model.CreateGroupPongMessage, error) {
	message := new(middleware_pb.CreateGroupPongMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]unMarshalCreateGroupPongMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	//base := model.BaseSignedMessage{SI: *sign}

	var ts time.Time
	ts.UnmarshalBinary(message.Ts)

	m := &model.CreateGroupPongMessage{
		SignInfo:  *sign,
		PingID:    *message.PingID,
		Timestamp: ts,
	}
	return m, nil
}

func unMarshalSharePieceReqMessage(b []byte) (*model.ReqSharePieceMessage, error) {
	message := new(middleware_pb.ReqSharePieceMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]unMarshalSharePieceReqMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	//base := model.BaseSignedMessage{SI: *sign}

	m := &model.ReqSharePieceMessage{
		SignInfo:  *sign,
		GroupHash: common.BytesToHash(message.GHash),
	}
	return m, nil
}

func unMarshalSharePieceResponseMessage(b []byte) (*model.ResponseSharePieceMessage, error) {
	message := new(middleware_pb.ResponseSharePieceMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]unMarshalResponseSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	//base := model.BaseSignedMessage{SI: *sign}

	m := &model.ResponseSharePieceMessage{
		SignInfo:  *sign,
		GroupHash: common.BytesToHash(message.GHash),
		Share:     *pbToSharePiece(message.SharePiece),
	}
	return m, nil
}
