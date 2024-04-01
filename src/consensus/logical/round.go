package logical

import (
	"com.tuntun.rangers/node/src/consensus/access"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/log"
	"fmt"
)

type Error struct {
	cause    error
	task     string
	round    int
	victim   string
	culprits []string
}

func NewError(err error, task string, round int, victim string, culprits []string) *Error {
	return &Error{cause: err, task: task, round: round, victim: victim, culprits: culprits}
}

func (err *Error) Error() string {
	if err == nil || err.cause == nil {
		return "Error is nil"
	}
	if err.culprits != nil && len(err.culprits) > 0 {
		return fmt.Sprintf("task %s, party %v, round %d, culprits %s: %s",
			err.task, err.victim, err.round, err.culprits, err.cause.Error())
	}
	return fmt.Sprintf("task %s, party %v, round %d: %s",
		err.task, err.victim, err.round, err.cause.Error())
}

func (err *Error) Unwrap() error { return err.cause }

func (err *Error) Cause() error { return err.cause }

type Round interface {
	Start() *Error
	Update(msg model.ConsensusMessage) *Error
	RoundNumber() int
	CanAccept(msg model.ConsensusMessage) int
	CanProceed() bool
	NextRound() Round
}

type (
	baseRound struct {
		ok             []bool // `ok` tracks parties which have been verified by Update()
		processed      map[string]byte
		futureMessages map[string]model.ConsensusMessage
		started        bool
		number         int
		logger         log.Logger
	}

	round1 struct {
		*baseRound
		belongGroups *access.JoinedGroupStorage
		globalGroups *access.GroupAccessor
		minerReader  *access.MinerPoolReader
		blockchain   core.BlockChain
		changedId    chan string
		canProcessed bool
		mi           groupsig.ID
		netServer    net.NetworkServer
	}
	round2 struct {
		*round1
	}
	round3 struct {
		*round2
	}
)

func (round *baseRound) RoundNumber() int {
	return round.number
}

func (round *baseRound) check() {

}

//type finalization struct {
//}
//
//func (round *finalization) Start() *Error {
//	return nil
//}
//func (round *finalization) Update(msg model.ConsensusMessage) *Error {
//	return nil
//}
//func (round *finalization) RoundNumber() int {
//	return -1
//}
//
//func (round *finalization) CanAccept(msg model.ConsensusMessage) int {
//	return -1
//}
//func (round *finalization) CanProceed() bool {
//	return true
//}
//func (round *finalization) NextRound() Round {
//	return nil
//}
