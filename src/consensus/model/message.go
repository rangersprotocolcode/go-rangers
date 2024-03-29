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

package model

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/base"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"strconv"
	"time"
)

type SignInfo struct {
	dataHash  common.Hash
	signature groupsig.Signature
	signerID  groupsig.ID

	version int32
}

func NewSignInfo(sk groupsig.Seckey, id groupsig.ID, hasher common.Hasher) (SignInfo, bool) {
	result := SignInfo{}
	if !sk.IsValid() || !id.IsValid() {
		return result, false
	}

	hash := hasher.GenHash()
	result.dataHash = hash
	result.signerID = id
	result.signature = groupsig.Sign(sk, hash.Bytes())
	result.version = common.ConsensusVersion
	return result, true
}

func MakeSignInfo(dataHash common.Hash, signature groupsig.Signature, signerID groupsig.ID, version int32) SignInfo {
	result := SignInfo{}
	result.dataHash = dataHash
	result.signature = signature
	result.signerID = signerID
	result.version = version
	return result
}

func (si SignInfo) VerifySign(pk groupsig.Pubkey) bool {
	if !si.signerID.IsValid() {
		return false
	}
	return groupsig.VerifySig(pk, si.dataHash.Bytes(), si.signature)
}

func (si SignInfo) IsEqual(rhs SignInfo) bool {
	return si.dataHash.Str() == rhs.dataHash.Str() && si.signerID.IsEqual(rhs.signerID) && si.signature.IsEqual(rhs.signature)
}

func (si SignInfo) GetSignerID() groupsig.ID {
	return si.signerID
}

func (si SignInfo) GetDataHash() common.Hash {
	return si.dataHash
}

func (si SignInfo) GetSignature() groupsig.Signature {
	return si.signature
}

func (si SignInfo) GetVersion() int32 {
	return si.version
}

type ConsensusMessage interface {
	GenHash() common.Hash
}

type CreateGroupPingMessage struct {
	FromGroupID groupsig.ID
	PingID      string
	BaseHeight  uint64

	SignInfo
}

func (msg *CreateGroupPingMessage) GenHash() common.Hash {
	buf := msg.FromGroupID.Serialize()
	buf = append(buf, []byte(msg.PingID)...)
	buf = append(buf, utility.UInt64ToByte(msg.BaseHeight)...)
	return base.Data2CommonHash(buf)
}

type CreateGroupPongMessage struct {
	PingID    string
	Timestamp time.Time

	SignInfo
}

func (msg *CreateGroupPongMessage) GenHash() common.Hash {
	buf := []byte(msg.PingID)
	tb, _ := msg.Timestamp.MarshalBinary()
	buf = append(buf, tb...)
	return base.Data2CommonHash(tb)
}

// ConsensusCreateGroupRawMessage
type ParentGroupConsensusMessage struct {
	GroupInitInfo GroupInitInfo
	SignInfo
}

func (msg *ParentGroupConsensusMessage) GenHash() common.Hash {
	return msg.GroupInitInfo.GroupHash()
}

// ConsensusCreateGroupSignMessage
type ParentGroupConsensusSignMessage struct {
	GroupHash common.Hash
	Launcher  groupsig.ID

	SignInfo
}

func (msg *ParentGroupConsensusSignMessage) GenHash() common.Hash {
	return msg.GroupHash
}

// ConsensusGroupRawMessage
type GroupInitMessage struct {
	GroupInitInfo GroupInitInfo
	SignInfo
}

func (msg *GroupInitMessage) GenHash() common.Hash {
	return msg.GroupInitInfo.GroupHash()
}

func (msg *GroupInitMessage) MemberExist(id groupsig.ID) bool {
	return msg.GroupInitInfo.MemberExists(id)
}

type SharePieceMessage struct {
	GroupHash      common.Hash
	GroupMemberNum int32

	ReceiverId groupsig.ID
	Share      SharePiece

	SignInfo
}

func (msg *SharePieceMessage) GenHash() common.Hash {
	buf := msg.GroupHash.Bytes()
	buf = append(buf, msg.ReceiverId.Serialize()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

type SignPubKeyMessage struct {
	GroupHash      common.Hash
	GroupID        groupsig.ID
	SignPK         groupsig.Pubkey
	GroupMemberNum int32

	SignInfo
}

func (msg *SignPubKeyMessage) GenHash() common.Hash {
	buf := msg.GroupHash.Bytes()
	buf = append(buf, msg.GroupID.Serialize()...)
	buf = append(buf, msg.SignPK.Serialize()...)
	return base.Data2CommonHash(buf)
}

type GroupInitedMessage struct {
	GroupHash common.Hash
	GroupID   groupsig.ID
	GroupPK   groupsig.Pubkey

	MemberMask []byte
	MemberNum  int32

	CreateHeight    uint64
	ParentGroupSign groupsig.Signature

	SignInfo
}

func (msg *GroupInitedMessage) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(msg.GroupHash.Bytes())
	buf.Write(msg.GroupID.Serialize())
	buf.Write(msg.GroupPK.Serialize())
	buf.Write(utility.UInt64ToByte(msg.CreateHeight))
	buf.Write(msg.ParentGroupSign.Serialize())
	buf.Write(msg.MemberMask)
	return base.Data2CommonHash(buf.Bytes())
}

type ReqSharePieceMessage struct {
	GroupHash common.Hash
	SignInfo
}

func (msg *ReqSharePieceMessage) GenHash() common.Hash {
	return msg.GroupHash
}

type ResponseSharePieceMessage struct {
	GroupHash common.Hash
	Share     SharePiece

	SignInfo
}

func (msg *ResponseSharePieceMessage) GenHash() common.Hash {
	buf := msg.GroupHash.Bytes()
	//buf = append(buf, msg.GHash.Bytes()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

type SignPubkeyReqMessage struct {
	GroupID groupsig.ID
	SignInfo
}

func (m *SignPubkeyReqMessage) GenHash() common.Hash {
	return base.Data2CommonHash(m.GroupID.Serialize())
}

type ConsensusCurrentMessage struct {
	GroupID     []byte
	PreHash     common.Hash
	PreTime     time.Time
	BlockHeight uint64

	SignInfo
}

func (msg *ConsensusCurrentMessage) GenHash() common.Hash {
	buf := msg.PreHash.Str()
	buf += string(msg.GroupID[:])
	buf += msg.PreTime.String()
	buf += strconv.FormatUint(msg.BlockHeight, 10)
	return base.Data2CommonHash([]byte(buf))
}

type ConsensusCastMessage struct {
	BH        types.BlockHeader
	ProveHash []common.Hash

	SignInfo
}

func (msg *ConsensusCastMessage) GenHash() common.Hash {
	//buf := bytes.Buffer{}
	//buf.Write(msg.BH.GenHash().Bytes())
	//for _, h := range msg.ProveHash {
	//	buf.Write(h.Bytes())
	//}
	//return base.Data2CommonHash(buf.Bytes())
	return msg.BH.GenHash()
}

func (msg *ConsensusCastMessage) VerifyRandomSign(pkey groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.BH.Random)
	if sig == nil || sig.IsNil() {
		return false
	}
	return groupsig.VerifySig(pkey, preRandom, *sig)
}

type ConsensusVerifyMessage struct {
	BlockHash  common.Hash
	RandomSign groupsig.Signature

	SignInfo
}

func (msg *ConsensusVerifyMessage) GenHash() common.Hash {
	//buf := bytes.Buffer{}
	//buf.Write(msg.BH.GenHash().Bytes())
	//for _, h := range msg.ProveHash {
	//	buf.Write(h.Bytes())
	//}
	//return base.Data2CommonHash(buf.Bytes())
	return msg.BlockHash
}

func (msg *ConsensusVerifyMessage) GenRandomSign(skey groupsig.Seckey, preRandom []byte) {
	sig := groupsig.Sign(skey, preRandom)
	msg.RandomSign = sig
}

type ConsensusBlockMessage struct {
	Block types.Block
}

func (msg *ConsensusBlockMessage) GenHash() common.Hash {
	buf := msg.Block.Header.GenHash().Bytes()
	buf = append(buf, msg.Block.Header.GroupId...)
	return base.Data2CommonHash(buf)
}

func (msg *ConsensusBlockMessage) VerifySig(gpk groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.Block.Header.Signature)
	if sig == nil {
		return false
	}
	b := groupsig.VerifySig(gpk, msg.Block.Header.Hash.Bytes(), *sig)
	if !b {
		return false
	}
	rsig := groupsig.DeserializeSign(msg.Block.Header.Random)
	if rsig == nil {
		return false
	}
	return groupsig.VerifySig(gpk, preRandom, *rsig)
}

type CastRewardTransSignMessage struct {
	ReqHash   common.Hash
	BlockHash common.Hash

	GroupID  groupsig.ID
	Launcher groupsig.ID

	SignInfo
}

func (msg *CastRewardTransSignMessage) GenHash() common.Hash {
	return msg.ReqHash
}
