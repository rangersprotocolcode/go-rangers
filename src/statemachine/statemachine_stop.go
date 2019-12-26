// statemachine的暂停、停止、删除容器
package statemachine

import (
	"github.com/docker/docker/api/types"
	"os"
)

// 暂停容器
func (s *StateMachine) Pause() {
	err := s.cli.ContainerPause(s.ctx, s.This.ID)
	if err != nil {
		s.logger.Errorf("fail to pause stm, %s. err: %s", s.TOJSONString(), err.Error())
	}

	s.logger.Warnf("pause stm, %s. err: %s", s.TOJSONString(), err.Error())
	s.paused()
}

func (s *StateMachine) UnPause() {
	err := s.cli.ContainerUnpause(s.ctx, s.This.ID)
	if err != nil {
		s.logger.Errorf("fail to unpause stm, %s. err: %s", s.TOJSONString(), err.Error())
	}

	s.logger.Warnf("unpause stm, %s. err: %s", s.TOJSONString(), err.Error())
	s.ready()
}

func (s *StateMachine) Stop() bool {
	err := s.cli.ContainerStop(s.ctx, s.This.ID, nil)
	if err != nil {
		s.logger.Errorf("fail to stop stm, %s. err: %s", s.TOJSONString(), err.Error())
		return false
	}

	s.logger.Warnf("stopped stm, %s", s.TOJSONString())
	s.stopped()

	return true
}

func (s *StateMachine) Remove() bool {
	err := s.cli.ContainerRemove(s.ctx, s.This.ID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: false})
	if err != nil {
		s.logger.Errorf("fail to remove stm, %s. err: %s", s.TOJSONString(), err.Error())
		return false
	}

	_, err = s.cli.ImageRemove(s.ctx, s.This.Image, types.ImageRemoveOptions{Force: true,})
	if err != nil {
		s.logger.Errorf("fail to remove stm, %s. err: %s", s.TOJSONString(), err.Error())
		return false
	}

	s.logger.Warnf("removed stm, %s", s.TOJSONString())
	s.removed()
	return true
}

func (s *StateMachine) Clear() {
	os.Remove(s.storageGame)
}

func (s *StateMachine) StopHeartbeat(){
	s.heartBeat = false
}
