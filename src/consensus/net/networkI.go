package net

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware/types"
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

	OnMessageCast(msg *model.ConsensusCastMessage)

	OnMessageVerify(msg *model.ConsensusVerifyMessage)

	OnMessageNewTransactions(txs []common.Hashes)
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

	BroadcastNewBlock(cbm *model.ConsensusBlockMessage)

	JoinGroupNet(groupId string)

	ReleaseGroupNet(groupIdentifier string)

	ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID)

	ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID)

	AskSignPkMessage(msg *model.SignPubkeyReqMessage, receiver groupsig.ID)

	AnswerSignPkMessage(msg *model.SignPubKeyMessage, receiver groupsig.ID)
}
