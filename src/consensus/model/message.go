package model

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/middleware/types"
	"strconv"
	"time"
)

type Hasher interface {
	GenHash() common.Hash
}

//数据签名结构
type SignInfo struct {
	dataHash  common.Hash        //哈希值
	signature groupsig.Signature //对HASH的签名
	signerID  groupsig.ID        //用户ID或组ID，看消息类型

	version int32
}

func NewSignInfo(sk groupsig.Seckey, id groupsig.ID, hasher Hasher) (SignInfo, bool) {
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

//用pk验证签名，验证通过返回true，否则false。
func (si SignInfo) VerifySign(pk groupsig.Pubkey) bool {
	if !si.signerID.IsValid() {
		return false
	}
	return groupsig.VerifySig(pk, si.dataHash.Bytes(), si.signature)
}

func (si SignInfo) IsEqual(rhs SignInfo) bool {
	return si.dataHash.Str() == rhs.dataHash.Str() && si.signerID.IsEqual(rhs.signerID) && si.signature.IsEqual(rhs.signature)
}

//GetID
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

//====================================父组建组共识消息================================
type CreateGroupPingMessage struct {
	FromGroupID groupsig.ID
	PingID      string
	BaseHeight  uint64

	SignInfo
}

func (msg *CreateGroupPingMessage) GenHash() common.Hash {
	buf := msg.FromGroupID.Serialize()
	buf = append(buf, []byte(msg.PingID)...)
	buf = append(buf, common.Uint64ToByte(msg.BaseHeight)...)
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

//ConsensusCreateGroupRawMessage
type ParentGroupConsensusMessage struct {
	GroupInitInfo GroupInitInfo //组初始化共识
	SignInfo
}

func (msg *ParentGroupConsensusMessage) GenHash() common.Hash {
	return msg.GroupInitInfo.GroupHash()
}

//ConsensusCreateGroupSignMessage
type ParentGroupConsensusSignMessage struct {
	GroupHash common.Hash
	Launcher  groupsig.ID

	SignInfo
}

func (msg *ParentGroupConsensusSignMessage) GenHash() common.Hash {
	return msg.GroupHash
}

//-----------------------------------------------------组创建消息-------------------------------
//收到父亲组的启动组初始化消息
//to do : 组成员ID列表在哪里提供
//ConsensusGroupRawMessage
type GroupInitMessage struct {
	GroupInitInfo GroupInitInfo //组初始化共识
	SignInfo
}

func (msg *GroupInitMessage) GenHash() common.Hash {
	return msg.GroupInitInfo.GroupHash()
}

func (msg *GroupInitMessage) MemberExist(id groupsig.ID) bool {
	return msg.GroupInitInfo.MemberExists(id)
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type SharePieceMessage struct {
	GroupHash      common.Hash //组初始化共识（ConsensusGroupInitSummary）的哈希
	GroupMemberNum int32

	ReceiverId groupsig.ID //接收者（矿工）的ID
	Share      SharePiece  //消息明文（由传输层用接收者公钥对消息进行加密和解密）

	SignInfo
}

func (msg *SharePieceMessage) GenHash() common.Hash {
	buf := msg.GroupHash.Bytes()
	buf = append(buf, msg.ReceiverId.Serialize()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

//向组内成员发送签名公钥消息（所有成员相同）
type SignPubKeyMessage struct {
	GroupHash      common.Hash
	GroupID        groupsig.ID     //组id
	SignPK         groupsig.Pubkey //组成员签名公钥
	GroupMemberNum int32

	SignInfo
}

func (msg *SignPubKeyMessage) GenHash() common.Hash {
	buf := msg.GroupHash.Bytes()
	buf = append(buf, msg.GroupID.Serialize()...)
	buf = append(buf, msg.SignPK.Serialize()...)
	return base.Data2CommonHash(buf)
}

//向组外广播该组已经初始化完成(组外节点要收到门限个消息相同，才进行上链)
type GroupInitedMessage struct {
	GroupHash common.Hash
	GroupID   groupsig.ID     //组ID(可以由组公钥生成)
	GroupPK   groupsig.Pubkey //组公钥

	MemberMask []byte //组成员mask，值为1的位表名该candidate在组成员列表中,根据该mask表和candidate集合可恢复出组成员列表
	MemberNum  int32

	CreateHeight    uint64 //组开始创建时的高度
	ParentGroupSign groupsig.Signature

	SignInfo
}

func (msg *GroupInitedMessage) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(msg.GroupHash.Bytes())
	buf.Write(msg.GroupID.Serialize())
	buf.Write(msg.GroupPK.Serialize())
	buf.Write(common.Uint64ToByte(msg.CreateHeight))
	buf.Write(msg.ParentGroupSign.Serialize())
	buf.Write(msg.MemberMask)
	return base.Data2CommonHash(buf.Bytes())
}

//-------------------------------------------缺失请求消息-------------------------------

type ReqSharePieceMessage struct {
	GroupHash common.Hash
	SignInfo
}

func (msg *ReqSharePieceMessage) GenHash() common.Hash {
	return msg.GroupHash
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type ResponseSharePieceMessage struct {
	GroupHash common.Hash //组初始化共识（ConsensusGroupInitSummary）的哈希 就是group hash
	Share     SharePiece  //消息明文（由传输层用接收者公钥对消息进行加密和解密）

	SignInfo
}

func (msg *ResponseSharePieceMessage) GenHash() common.Hash {
	buf := msg.GroupHash.Bytes()
	//buf = append(buf, msg.GHash.Bytes()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

//请求签名公钥
type SignPubkeyReqMessage struct {
	GroupID groupsig.ID
	SignInfo
}

func (m *SignPubkeyReqMessage) GenHash() common.Hash {
	return base.Data2CommonHash(m.GroupID.Serialize())
}

//---------------------------------------------------------铸块消息族--------------------------
//铸块消息族的SI用组成员签名公钥验签

//成为当前处理组消息 - 由第一个发现当前组成为铸块组的成员发出
type ConsensusCurrentMessage struct {
	GroupID     []byte      //铸块组
	PreHash     common.Hash //上一块哈希
	PreTime     time.Time   //上一块完成时间
	BlockHeight uint64      //铸块高度

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

//func (msg *ConsensusCastMessage) GenRandomSign(skey groupsig.Seckey, preRandom []byte)  {
//	sig := groupsig.Sign(skey, preRandom)
//    msg.BH.Random = sig.Serialize()
//}

func (msg *ConsensusCastMessage) VerifyRandomSign(pkey groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.BH.Random)
	if sig == nil || sig.IsNil() {
		return false
	}
	return groupsig.VerifySig(pkey, preRandom, *sig)
}

//出块消息 - 由成为KING的组成员发出
//type ConsensusCastMessage struct {
//	ConsensusBlockMessageBase
//}

//验证消息 - 由组内的验证人发出（对KING的出块进行验证）
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

//func (msg *ConsensusVerifyMessage) VerifyRandomSign(pkey groupsig.Pubkey, preRandom []byte) bool {
//	sig := msg.RandomSign
//	if sig.IsNil() {
//		return false
//	}
//	return groupsig.VerifySig(pkey, preRandom, sig)
//}

//铸块成功消息 - 该组成功完成了一个铸块，由组内任意一个收集到k个签名的成员发出
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

//==============================奖励交易==============================
type CastRewardTransSignMessage struct {
	ReqHash   common.Hash
	BlockHash common.Hash

	//不序列化
	GroupID  groupsig.ID
	Launcher groupsig.ID

	SignInfo
}

func (msg *CastRewardTransSignMessage) GenHash() common.Hash {
	return msg.ReqHash
}

//type ISignedMessage interface {
//	GenSign(ski SecKeyInfo, hasher Hasher) bool
//	VerifySign(pk groupsig.Pubkey) bool
//}

//type BaseSignedMessage struct {
//	SI SignData
//}
//
//
//func (sign *BaseSignedMessage) GenSign(ski SecKeyInfo, hasher Hasher) bool {
//	if !ski.IsValid() {
//		return false
//	}
//	sign.SI = GenSignData(hasher.GenHash(), ski.GetID(), ski.SK)
//	return true
//}
//
//func (sign *BaseSignedMessage) VerifySign(pk groupsig.Pubkey) (ok bool) {
//	if !sign.SI.GetID().IsValid() {
//		return false
//	}
//	ok = sign.SI.VerifySign(pk)
//	if !ok {
//		fmt.Printf("verifySign fail, pk=%v, id=%v, sign=%v, data=%v\n", pk.GetHexString(), sign.SI.SignMember.GetHexString(), sign.SI.DataSign.GetHexString(), sign.SI.DataHash.Hex())
//	}
//	return
//}
