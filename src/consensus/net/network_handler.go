package net

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

type ConsensusHandler struct {
	groupCreateMessageProcessor GroupCreateMessageProcessor
	miningMessageProcessor      MiningMessageProcessor
}

var MessageHandler = new(ConsensusHandler)

func (c *ConsensusHandler) Init(groupCreateMessageProcessor GroupCreateMessageProcessor, miningMessageProcessor MiningMessageProcessor) {
	c.groupCreateMessageProcessor = groupCreateMessageProcessor
	c.miningMessageProcessor = miningMessageProcessor
	InitStateMachines()
}

//func (c *ConsensusHandler) Processor() MessageProcessor {
//	return c.processor
//}

func (c *ConsensusHandler) ready() bool {
	return c.groupCreateMessageProcessor != nil && c.miningMessageProcessor != nil && c.miningMessageProcessor.Ready()
}

func (c *ConsensusHandler) Handle(sourceId string, msg network.Message) error {
	code := msg.Code
	body := msg.Body

	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	if !c.ready() {
		log.Printf("message ingored because processor not ready. code=%v\n", code)
		return fmt.Errorf("processor not ready yet")
	}
	switch code {
	case network.GroupInitMsg:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return e
		}

		//belongGroup := m.GInfo.MemberExists(c.processor.GetMinerID())

		//var machines *StateMachines
		//if belongGroup {
		//	machines = &GroupInsideMachines
		//} else {
		//	machines = &GroupOutsideMachines
		//}
		GroupInsideMachines.GetMachine(m.GroupInitInfo.GroupHash().Hex(), len(m.GroupInitInfo.GroupMembers)).Transform(NewStateMsg(code, m, sourceId))
	case network.KeyPieceMsg:
		m, e := unMarshalConsensusSharePieceMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GroupHash.Hex(), int(m.GroupMemberNum)).Transform(NewStateMsg(code, m, sourceId))
		logger.Infof("SharepieceMsg receive from:%v, gHash:%v", sourceId, m.GroupHash.Hex())
	case network.SignPubkeyMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GroupHash.Hex(), int(m.GroupMemberNum)).Transform(NewStateMsg(code, m, sourceId))
		logger.Infof("SignPubKeyMsg receive from:%v, gHash:%v, groupId:%v", sourceId, m.GroupHash.Hex(), m.GroupID.GetHexString())
	case network.GroupInitDoneMsg:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
			return e
		}
		logger.Infof("Rcv GroupInitDoneMsg from:%s,gHash:%s, groupId:%v", sourceId, m.GroupHash.Hex(), m.GroupID.GetHexString())

		//belongGroup := c.processor.ExistInGroup(m.GHash)
		//var machines *StateMachines
		//if belongGroup {
		//	machines = &GroupInsideMachines
		//} else {
		//	machines = &GroupOutsideMachines
		//}
		GroupInsideMachines.GetMachine(m.GroupHash.Hex(), int(m.MemberNum)).Transform(NewStateMsg(code, m, sourceId))

	case network.CurrentGroupCastMsg:

	case network.CastVerifyMsg:
		m, e := unMarshalConsensusCastMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCastMessage because of unmarshal error%s", e.Error())
			return e
		}

		id := utility.GetGoroutineId()
		middleware.PerfLogger.Infof("start Verify msg, id: %d, cost: %v, height: %v, hash: %v, msg size: %d", id, time.Since(m.BH.CurTime), m.BH.Height, m.BH.Hash.String(), len(body))
		c.miningMessageProcessor.OnMessageCast(m)
		middleware.PerfLogger.Infof("fin Verify msg, id: %d, cost: %v, height: %v, hash: %v, msg size: %d", id, time.Since(m.BH.CurTime), m.BH.Height, m.BH.Hash.String(), len(body))

	case network.VerifiedCastMsg:
		m, e := unMarshalConsensusVerifyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusVerifyMessage because of unmarshal error%s", e.Error())
			return e
		}

		id := utility.GetGoroutineId()
		middleware.PerfLogger.Infof("start Verified msg, id: %d, hash: %v, msg size: %d", id, m.BlockHash.String(), len(body))
		c.miningMessageProcessor.OnMessageVerify(m)
		middleware.PerfLogger.Infof("fin Verified msg, id: %d, hash: %v, msg size: %d", id, m.BlockHash.String(), len(body))

	case network.CreateGroupaRaw:
		m, e := unMarshalConsensusCreateGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupRawMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.groupCreateMessageProcessor.OnMessageParentGroupConsensus(m)
		return nil
	case network.CreateGroupSign:
		m, e := unMarshalConsensusCreateGroupSignMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.groupCreateMessageProcessor.OnMessageParentGroupConsensusSign(m)
		return nil
	case network.AskSignPkMsg:
		m, e := unMarshalConsensusSignPKReqMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalConsensusSignPKReqMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.groupCreateMessageProcessor.OnMessageSignPKReq(m)
	case network.AnswerSignPkMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.groupCreateMessageProcessor.OnMessageSignPK(m)

	case network.GroupPing:
		m, e := unMarshalCreateGroupPingMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalCreateGroupPingMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.groupCreateMessageProcessor.OnMessageCreateGroupPing(m)
	case network.GroupPong:
		m, e := unMarshalCreateGroupPongMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalCreateGroupPongMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.groupCreateMessageProcessor.OnMessageCreateGroupPong(m)

	case network.ReqSharePiece:
		m, e := unMarshalSharePieceReqMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalSharePieceReqMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.groupCreateMessageProcessor.OnMessageSharePieceReq(m)

	case network.ResponseSharePiece:
		m, e := unMarshalSharePieceResponseMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalSharePieceResponseMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.groupCreateMessageProcessor.OnMessageSharePieceResponse(m)
	}

	return nil
}
