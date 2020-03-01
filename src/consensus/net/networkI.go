package net

import (
	"x/src/consensus/model"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/middleware/types"
)

type GroupCreateMessageProcessor interface {
	OnMessageCreateGroupPing(msg *model.CreateGroupPingMessage)

	OnMessageCreateGroupPong(msg *model.CreateGroupPongMessage)

	OnMessageParentGroupConsensus(msg *model.ParentGroupConsensusMessage)

	OnMessageParentGroupConsensusSign(msg *model.ParentGroupConsensusSignMessage)

	OnMessageGroupInit(msg *model.GroupInitMessage)

	OnMessageSharePiece(msg *model.SharePieceMessage)

	OnMessageSignPK(msg *model.SignPubKeyMessage)

	OnMessageGroupInited(msg *model.GroupInitedMessage)

	OnMessageSharePieceReq(msg *model.ReqSharePieceMessage)

	OnMessageSharePieceResponse(msg *model.ResponseSharePieceMessage)

	OnMessageSignPKReq(msg *model.SignPubkeyReqMessage)
}

type MiningMessageProcessor interface {
	Ready() bool

	//GetMinerID() groupsig.ID
	//
	//ExistInGroup(gHash common.Hash) bool

	OnMessageCast(msg *model.ConsensusCastMessage)

	OnMessageVerify(msg *model.ConsensusVerifyMessage)

	OnMessageNewTransactions(txs []common.Hashes)

	OnMessageBlock(msg *model.ConsensusBlockMessage)
}

type GroupBrief struct {
	Gid    groupsig.ID
	MemIds []groupsig.ID
}

type NetworkServer interface {
	SendGroupPingMessage(msg *model.CreateGroupPingMessage, receiver groupsig.ID)

	SendGroupPongMessage(msg *model.CreateGroupPongMessage, groupId string, belongGroup bool)

	SendCreateGroupRawMessage(msg *model.ParentGroupConsensusMessage, belongGroup bool)

	SendCreateGroupSignMessage(msg *model.ParentGroupConsensusSignMessage, parentGid groupsig.ID)

	SendGroupInitMessage(grm *model.GroupInitMessage)

	SendKeySharePiece(spm *model.SharePieceMessage)

	SendSignPubKey(spkm *model.SignPubKeyMessage)

	BroadcastGroupInfo(cgm *model.GroupInitedMessage)

	SendCastVerify(ccm *model.ConsensusCastMessage, group *GroupBrief, body []*types.Transaction)

	SendVerifiedCast(cvm *model.ConsensusVerifyMessage, receiver groupsig.ID)

	BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *GroupBrief)

	JoinGroupNet(groupId string)

	ReleaseGroupNet(groupIdentifier string)

	ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID)

	ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID)

	AskSignPkMessage(msg *model.SignPubkeyReqMessage, receiver groupsig.ID)

	AnswerSignPkMessage(msg *model.SignPubKeyMessage, receiver groupsig.ID)
}
