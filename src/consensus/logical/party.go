package logical

import (
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/log"
	"errors"
	"fmt"
	"sync"
)

type Party interface {
	Start() *Error
	Update(msg model.ConsensusMessage)

	StoreMessage(msg model.ConsensusMessage)
	StoreMessages(msg []model.ConsensusMessage)
	GetFutureMessage() []model.ConsensusMessage

	FirstRound() Round

	setRound(Round) *Error
	round() Round
	advance()
	lock()
	unlock()

	String() string
}

type baseParty struct {
	Done    chan byte
	started bool

	mtx sync.Mutex
	rnd Round

	logger         log.Logger
	futureMessages []model.ConsensusMessage
}

func (p *baseParty) String() string {
	return fmt.Sprintf("round: %d", p.round().RoundNumber())
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
		p.logger.Errorf("finishied party, reject msg")
		return
	}

	switch p.round().CanAccept(msg) {
	case 0:
		if err := p.round().Update(msg); err != nil {
			p.unlock()
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
			p.round().Start()
		} else {
			// no more round, end this party
			p.Done <- 1
			return
		}
	}
}

func (p *baseParty) StoreMessage(msg model.ConsensusMessage) {
	p.futureMessages = append(p.futureMessages, msg)
}

func (p *baseParty) StoreMessages(msgs []model.ConsensusMessage) {
	for _, msg := range msgs {
		p.StoreMessage(msg)
	}
}

func (p *baseParty) GetFutureMessage() []model.ConsensusMessage {
	return p.futureMessages
}

type SignParty struct {
	baseParty
	ChangedId chan string
}

func (p *SignParty) Start() *Error {
	p.lock()
	defer p.unlock()

	if p.started {
		return NewError(errors.New("already started"), "start", 0, "", nil)
	}

	p.started = true
	p.setRound(p.FirstRound())

	return p.rnd.Start()
}

func (p *SignParty) FirstRound() Round {
	return &round1{&baseRound{futureMessages: &p.futureMessages}}
}
