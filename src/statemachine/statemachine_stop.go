// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

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
