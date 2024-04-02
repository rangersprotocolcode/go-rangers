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
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/pb"
	"com.tuntun.rangers/node/src/middleware/types"
	"github.com/gogo/protobuf/proto"
)

func marshalGroupInfo(gInfo *model.GroupInitInfo) *middleware_pb.ConsensusGroupInitInfo {
	mems := make([][]byte, gInfo.MemberSize())
	for i, mem := range gInfo.GroupMembers {
		mems[i] = mem.Serialize()
	}

	return &middleware_pb.ConsensusGroupInitInfo{
		Header:    types.GroupToPbHeader(gInfo.GroupHeader),
		Signature: gInfo.ParentGroupSign.Serialize(),
		Mems:      mems,
	}
}

func marshalConsensusGroupRawMessage(m *model.GroupInitMessage) ([]byte, error) {
	gi := marshalGroupInfo(&m.GroupInitInfo)

	sign := signDataToPb(&m.SignInfo)

	message := middleware_pb.ConsensusGroupRawMessage{
		GInfo: gi,
		Sign:  sign,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSharePieceMessage(m *model.SharePieceMessage) ([]byte, error) {
	share := sharePieceToPb(&m.Share)
	sign := signDataToPb(&m.SignInfo)

	message := middleware_pb.ConsensusSharePieceMessage{
		GHash:      m.GroupHash.Bytes(),
		Dest:       m.ReceiverId.Serialize(),
		SharePiece: share,
		Sign:       sign,
		MemCnt:     &m.GroupMemberNum,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSignPubKeyMessage(m *model.SignPubKeyMessage) ([]byte, error) {
	signData := signDataToPb(&m.SignInfo)

	message := middleware_pb.ConsensusSignPubKeyMessage{
		GHash:    m.GroupHash.Bytes(),
		SignPK:   m.SignPK.Serialize(),
		SignData: signData,
		GroupID:  m.GroupID.Serialize(),
		MemCnt:   &m.GroupMemberNum,
	}
	return proto.Marshal(&message)
}
func marshalConsensusGroupInitedMessage(m *model.GroupInitedMessage) ([]byte, error) {
	si := signDataToPb(&m.SignInfo)
	message := middleware_pb.ConsensusGroupInitedMessage{
		GHash:        m.GroupHash.Bytes(),
		GroupID:      m.GroupID.Serialize(),
		GroupPK:      m.GroupPK.Serialize(),
		CreateHeight: &m.CreateHeight,
		ParentSign:   m.ParentGroupSign.Serialize(),
		Sign:         si,
		MemCnt:       &m.MemberNum,
		MemMask:      m.MemberMask,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSignPubKeyReqMessage(m *model.SignPubkeyReqMessage) ([]byte, error) {
	signData := signDataToPb(&m.SignInfo)

	message := middleware_pb.ConsensusSignPubkeyReqMessage{
		GroupID:  m.GroupID.Serialize(),
		SignData: signData,
	}
	return proto.Marshal(&message)
}

func marshalConsensusCastMessage(m *model.ConsensusCastMessage) ([]byte, error) {
	bh := types.BlockHeaderToPb(m.BH)
	//groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SignInfo)

	hashs := make([][]byte, len(m.ProveHash))
	for i, h := range m.ProveHash {
		hashs[i] = h.Bytes()
	}

	message := middleware_pb.ConsensusCastMessage{Bh: bh, Sign: si, ProveHash: hashs}
	return proto.Marshal(&message)
}

func marshalConsensusVerifyMessage(m *model.ConsensusVerifyMessage) ([]byte, error) {
	message := &middleware_pb.ConsensusVerifyMessage{
		BlockHash:  m.BlockHash.Bytes(),
		RandomSign: m.RandomSign.Serialize(),
		Sign:       signDataToPb(&m.SignInfo),
	}
	return proto.Marshal(message)
}

func marshalConsensusBlockMessage(m *model.ConsensusBlockMessage) ([]byte, error) {
	block := types.BlockToPb(&m.Block)
	if block == nil {
		logger.Errorf("[peer]Block is nil while marshalConsensusBlockMessage")
	}
	message := middleware_pb.ConsensusBlockMessage{Block: block}
	return proto.Marshal(&message)
}

func signDataToPb(s *model.SignInfo) *middleware_pb.SignData {
	version := s.GetVersion()
	sign := middleware_pb.SignData{DataHash: s.GetDataHash().Bytes(), DataSign: s.GetSignature().Serialize(), SignMember: s.GetSignerID().Serialize(), Version: &version}
	return &sign
}

func sharePieceToPb(s *model.SharePiece) *middleware_pb.SharePiece {
	share := middleware_pb.SharePiece{Seckey: s.Share.Serialize(), Pubkey: s.Pub.Serialize()}
	return &share
}

func marshalConsensusCreateGroupRawMessage(msg *model.ParentGroupConsensusMessage) ([]byte, error) {
	gi := marshalGroupInfo(&msg.GroupInitInfo)

	sign := signDataToPb(&msg.SignInfo)

	message := middleware_pb.ConsensusCreateGroupRawMessage{GInfo: gi, Sign: sign}
	return proto.Marshal(&message)
}

func marshalConsensusCreateGroupSignMessage(msg *model.ParentGroupConsensusSignMessage) ([]byte, error) {
	sign := signDataToPb(&msg.SignInfo)

	message := middleware_pb.ConsensusCreateGroupSignMessage{GHash: msg.GroupHash.Bytes(), Sign: sign}
	return proto.Marshal(&message)
}

func marshalCreateGroupPingMessage(msg *model.CreateGroupPingMessage) ([]byte, error) {
	si := signDataToPb(&msg.SignInfo)
	message := &middleware_pb.CreateGroupPingMessage{
		Sign:        si,
		PingID:      &msg.PingID,
		FromGroupID: msg.FromGroupID.Serialize(),
		BaseHeight:  &msg.BaseHeight,
	}
	return proto.Marshal(message)
}

func marshalCreateGroupPongMessage(msg *model.CreateGroupPongMessage) ([]byte, error) {
	si := signDataToPb(&msg.SignInfo)
	tb, _ := msg.Timestamp.MarshalBinary()
	message := &middleware_pb.CreateGroupPongMessage{
		Sign:   si,
		PingID: &msg.PingID,
		Ts:     tb,
	}
	return proto.Marshal(message)
}

func marshalSharePieceReqMessage(msg *model.ReqSharePieceMessage) ([]byte, error) {
	si := signDataToPb(&msg.SignInfo)
	message := &middleware_pb.ReqSharePieceMessage{
		Sign:  si,
		GHash: msg.GroupHash.Bytes(),
	}
	return proto.Marshal(message)
}

func marshalSharePieceResponseMessage(msg *model.ResponseSharePieceMessage) ([]byte, error) {
	si := signDataToPb(&msg.SignInfo)
	message := &middleware_pb.ResponseSharePieceMessage{
		Sign:       si,
		GHash:      msg.GroupHash.Bytes(),
		SharePiece: sharePieceToPb(&msg.Share),
	}
	return proto.Marshal(message)
}
