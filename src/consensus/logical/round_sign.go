package logical

import "com.tuntun.rangers/node/src/consensus/model"

func (r *round1) Start() *Error {
	r.started = true
	return nil
}
func (r *round1) Update(msg model.ConsensusMessage) *Error {
	return nil
}
func (r *round1) CanAccept(msg model.ConsensusMessage) int {
	return 0
}
func (r *round1) CanProceed() bool {
	return false
}
func (r *round1) NextRound() Round {
	return nil
}
