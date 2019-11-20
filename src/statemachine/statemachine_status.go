// 状态管理
package statemachine

const (
	preparing    = "preparing(开始创建)"
	failToCreate = "failToCreate(创建失败，请检查配置文件)"
	prepared     = "prepared(创建成功，初始化中)"
	ready        = "ready(正常服务)"
)

// 设置statemacine的状态
func (s *StateMachine) setStatus(status string) {
	s.Status = status
}

func (s *StateMachine) ready() {
	s.setStatus(ready)
}

func (s *StateMachine) prepared() {
	s.setStatus(prepared)
}

func (s *StateMachine) failed() {
	s.setStatus(failToCreate)
}