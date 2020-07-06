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

	asynchronous  = "asynchronous"
	synchronizing = "synchronizing(同步状态中)"
	synchronized  = "synchronized(同步状态完成，等待重启)"
)

// 设置stateMachine的状态
func (s *StateMachine) setStatus(status string) {
	if status != s.Status {
		s.logger.Warnf("stm %s change status, from %s to %s", s.Game, s.Status, status)
		s.Status = status
	}
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

func (s *StateMachine) async() {
	s.setStatus(asynchronous)
}

func (s *StateMachine) isAsync() bool {
	return s.Status == asynchronous
}
