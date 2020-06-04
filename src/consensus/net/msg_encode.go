package net

import (
	"x/src/middleware/pb"
	"x/src/consensus/model"
	"github.com/gogo/protobuf/proto"
	"x/src/middleware/types"
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

//func consensusGroupInitSummaryToPb(m *model.ConsensusGroupInitSummary) *middleware_pb.ConsensusGroupInitSummary {
//	message := middleware_pb.ConsensusGroupInitSummary{
//		Header: 		types.GroupToPbHeader(m.GHeader),
//		Signature:       m.Signature.Serialize(),
//	}
//	return &message
//}

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

//--------------------------------------------组铸币--------------------------------------------------------------------

func marshalConsensusCastMessage(m *model.ConsensusCastMessage) ([]byte, error) {
	bh := types.BlockHeaderToPb(&m.BH)
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

//----------------------------------------------------------------------------------------------------------------------

func signDataToPb(s *model.SignInfo) *middleware_pb.SignData {
	version := s.GetVersion()
	sign := middleware_pb.SignData{DataHash: s.GetDataHash().Bytes(), DataSign: s.GetSignature().Serialize(), SignMember: s.GetSignerID().Serialize(), Version: &version}
	return &sign
}

func sharePieceToPb(s *model.SharePiece) *middleware_pb.SharePiece {
	share := middleware_pb.SharePiece{Seckey: s.Share.Serialize(), Pubkey: s.Pub.Serialize()}
	return &share
}

//func staticGroupInfoToPb(s *model.StaticGroupSummary) *middleware_pb.StaticGroupSummary {
//	groupId := s.GroupID.Serialize()
//	groupPk := s.GroupPK.Serialize()
//
//	gis := consensusGroupInitSummaryToPb(&s.GIS)
//
//	groupInfo := middleware_pb.StaticGroupSummary{GroupID: groupId, GroupPK: groupPk, Gis: gis}
//	return &groupInfo
//}
//
//func pubKeyInfoToPb(p *model.PubKeyInfo) *middleware_pb.PubKeyInfo {
//	id := p.ID.Serialize()
//	pk := p.PK.Serialize()
//
//	pkInfo := middleware_pb.PubKeyInfo{ID: id, PublicKey: pk}
//	return &pkInfo
//}

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
