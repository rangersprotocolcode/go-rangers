// 状态管理
package statemachine

const (
	preparing    = "preparing(开始创建)"
	failToCreate = "failToCreate(创建失败，请检查配置文件)"
	prepared     = "prepared(创建成功，初始化中)"
	ready        = "ready(正常服务)"

	pause  = "paused(暂停)"
	stop   = "stopped(停止)"
	remove = "removed(已删除)"

	synchronizing = "synchronizing(同步状态中)"
	synchronized  = "synchronized(同步状态完成，等待重启)"
)

// 设置stateMachine的状态
func (s *StateMachine) setStatus(status string) {
	s.logger.Warnf("stm %s change status, from %s to %s", s.Game, s.Status, status)
	s.Status = status
}

func (s *StateMachine) ready() {
	// 刷新存储状态
	s.RefreshStorageStatus(s.RequestId)

	s.setStatus(ready)
}

func (s *StateMachine) isReady() bool {
	return s.Status == ready
}

func (s *StateMachine) prepared() {
	s.setStatus(prepared)
}

func (s *StateMachine) failed() {
	s.setStatus(failToCreate)
}

func (s *StateMachine) paused() {
	s.setStatus(pause)
}

func (s *StateMachine) stopped() {
	s.setStatus(stop)
}

func (s *StateMachine) removed() {
	s.setStatus(remove)
}

func (s *StateMachine) sync() {
	s.setStatus(synchronizing)
}

func (s *StateMachine) isSync() bool {
	return s.Status == synchronizing
}

func (s *StateMachine) synced() {
	// 刷新存储状态
	s.RefreshStorageStatus(s.RequestId)

	s.setStatus(synchronized)
}

func (s *StateMachine) isSynced() bool {
	return s.Status == synchronized
}
