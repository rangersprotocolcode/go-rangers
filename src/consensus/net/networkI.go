package net

import (
	"x/src/consensus/model"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/middleware/types"
)

type MessageProcessor interface {
	Ready() bool

	GetMinerID() groupsig.ID

	ExistInGroup(gHash common.Hash) bool

	OnMessageGroupInit(msg *model.ConsensusGroupRawMessage)

	OnMessageSharePiece(msg *model.ConsensusSharePieceMessage)

	OnMessageSignPK(msg *model.ConsensusSignPubKeyMessage)

	OnMessageSignPKReq(msg *model.ConsensusSignPubkeyReqMessage)

	OnMessageGroupInited(msg *model.ConsensusGroupInitedMessage)

	OnMessageCast(msg *model.ConsensusCastMessage)

	OnMessageVerify(msg *model.ConsensusVerifyMessage)

	OnMessageNewTransactions(txs []common.Hash)

	OnMessageBlock(msg *model.ConsensusBlockMessage)

	OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage)

	OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage)

	OnMessageCreateGroupPing(msg *model.CreateGroupPingMessage)

	OnMessageCreateGroupPong(msg *model.CreateGroupPongMessage)

	OnMessageSharePieceReq(msg *model.ReqSharePieceMessage)
	OnMessageSharePieceResponse(msg *model.ResponseSharePieceMessage)
}

type GroupBrief struct {
	Gid    groupsig.ID
	MemIds []groupsig.ID
}

type NetworkServer interface {
	SendGroupInitMessage(grm *model.ConsensusGroupRawMessage)

	SendKeySharePiece(spm *model.ConsensusSharePieceMessage)

	SendSignPubKey(spkm *model.ConsensusSignPubKeyMessage)

	BroadcastGroupInfo(cgm *model.ConsensusGroupInitedMessage)

	SendCastVerify(ccm *model.ConsensusCastMessage, group *GroupBrief, body []*types.Transaction)

	SendVerifiedCast(cvm *model.ConsensusVerifyMessage, receiver groupsig.ID)

	BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *GroupBrief)

	SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage)

	SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage, parentGid groupsig.ID)

	BuildGroupNet(groupIdentifier string, mems []groupsig.ID)

	ReleaseGroupNet(groupIdentifier string)

	AnswerSignPkMessage(msg *model.ConsensusSignPubKeyMessage, receiver groupsig.ID)

	AskSignPkMessage(msg *model.ConsensusSignPubkeyReqMessage, receiver groupsig.ID)

	SendGroupPingMessage(msg *model.CreateGroupPingMessage, receiver groupsig.ID)

	SendGroupPongMessage(msg *model.CreateGroupPongMessage, group *GroupBrief)
	ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID)
	ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID)
}
