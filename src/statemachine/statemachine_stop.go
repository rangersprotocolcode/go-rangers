// statemachine的暂停、停止、删除容器
package statemachine

import "github.com/docker/docker/api/types"

// 暂停容器
func (s *StateMachine) Pause() {
	err := s.cli.ContainerPause(s.ctx, s.Id)
	if err != nil {
		s.logger.Errorf("fail to pause stm, %s. err: %s", s.TOJSONString(), err.Error())
	}

	s.logger.Warnf("pause stm, %s. err: %s", s.TOJSONString(), err.Error())
	s.paused()
}

func (s *StateMachine) UnPause() {
	err := s.cli.ContainerUnpause(s.ctx, s.Id)
	if err != nil {
		s.logger.Errorf("fail to unpause stm, %s. err: %s", s.TOJSONString(), err.Error())
	}

	s.logger.Warnf("unpause stm, %s. err: %s", s.TOJSONString(), err.Error())
	s.ready()
}

func (s *StateMachine) Stop() {
	err := s.cli.ContainerStop(s.ctx, s.Id, nil)
	if err != nil {
		s.logger.Errorf("fail to stop stm, %s. err: %s", s.TOJSONString(), err.Error())
	}

	s.logger.Warnf("stop stm, %s. err: %s", s.TOJSONString(), err.Error())
	s.stopped()

}

func (s *StateMachine) Remove() {
	err := s.cli.ContainerRemove(s.ctx, s.Id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: false})
	if err != nil {
		s.logger.Errorf("fail to Remove stm, %s. err: %s", s.TOJSONString(), err.Error())
	}

	s.logger.Warnf("Remove stm, %s. err: %s", s.TOJSONString(), err.Error())
	s.removed()
}

func (s *StateMachine) Update() {
	s.Stop()
	s.Remove()

}
