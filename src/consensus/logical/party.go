package logical

import (
	"com.tuntun.rangers/node/src/consensus/access"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/log"
	"errors"
	"fmt"
	"sync"
)

type Party interface {
	Start() *Error
	Cancel()
	Update(msg model.ConsensusMessage)

	StoreMessage(msg model.ConsensusMessage)
	GetFutureMessage() map[string]model.ConsensusMessage

	FirstRound() Round

	setRound(Round) *Error
	round() Round
	advance()
	lock()
	unlock()

	String() string

	SetId(key string)
}

type baseParty struct {
	Done, CancelChan chan byte
	Err              chan error

	id      string
	started bool

	mtx sync.Mutex
	rnd Round

	logger         log.Logger
	futureMessages map[string]model.ConsensusMessage
}

func (p *baseParty) String() string {
	return fmt.Sprintf("round: %d", p.round().RoundNumber())
}

func (p *baseParty) SetId(key string) {
	p.id = key
}

func (p *baseParty) Cancel() {
	go func() {
		p.CancelChan <- 0
	}()
}

func (p *baseParty) setRound(round Round) *Error {
	if p.rnd != nil {
		return NewError(errors.New("a round is already set on this party"), "setRound", p.rnd.RoundNumber(), "", nil)
	}
	p.rnd = round
	return nil
}

func (p *baseParty) round() Round {
	return p.rnd
}

func (p *baseParty) advance() {
	p.rnd = p.rnd.NextRound()
}

func (p *baseParty) lock() {
	p.mtx.Lock()
}

func (p *baseParty) unlock() {
	p.mtx.Unlock()
}

func (p *baseParty) Update(msg model.ConsensusMessage) {
	p.lock()
	defer p.unlock()

	if nil == p.round() {
		p.logger.Warnf("finished party: %s, reject msg: %s", p.id, msg.GetMessageID())
		return
	}

	switch p.round().CanAccept(msg) {
	case 0:
		if err := p.round().Update(msg); err != nil {
			p.Err <- err
			return
		}
	case 1:
		p.StoreMessage(msg)
	default:
		p.logger.Warnf("")
	}

	for {
		// need more message
		// waiting
		if !p.round().CanProceed() {
			return
		}

		// go to next round
		p.advance()
		if p.round() != nil {
			if err := p.round().Start(); err != nil {
				p.Err <- err
				return
			}
		} else {
			// no more round, end this party
			p.Done <- 1
			return
		}
	}
}

func (p *baseParty) StoreMessage(msg model.ConsensusMessage) {
	p.logger.Debugf("party: %s, store future message: %s", p.id, msg.GetMessageID())
	p.futureMessages[msg.GetMessageID()] = msg
}

func (p *baseParty) GetFutureMessage() map[string]model.ConsensusMessage {
	p.lock()
	defer p.unlock()

	return p.futureMessages
}

type SignParty struct {
	baseParty
	blockchain   core.BlockChain
	minerReader  *access.MinerPoolReader
	globalGroups *access.GroupAccessor
	belongGroups *access.JoinedGroupStorage
	ChangedId    chan string
	mi           groupsig.ID
	netServer    net.NetworkServer
}

func (p *SignParty) Start() *Error {
	p.lock()
	defer p.unlock()

	if p.started {
		return NewError(errors.New("already started"), "start", 0, "", nil)
	}

	p.started = true
	p.ChangedId = make(chan string)
	p.setRound(p.FirstRound())

	return p.rnd.Start()
}

func (p *SignParty) FirstRound() Round {
	return &round1{baseRound: &baseRound{futureMessages: p.futureMessages, errChan: p.Err, logger: p.logger},
		belongGroups: p.belongGroups, blockchain: p.blockchain,
		minerReader: p.minerReader, globalGroups: p.globalGroups,
		changedId: p.ChangedId, mi: p.mi, netServer: p.netServer}
}
