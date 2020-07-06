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

package statemachine

import (
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// 加载STM
func (s *StateMachineManager) loadStateMachine(service ContainerConfig) {
	s.lock.Lock()
	if 0 == len(service.Game) {
		s.logger.Errorf("fail to create stm with nil game. config: %s", service.TOJSONString())
		s.lock.Unlock()
		return
	}
	stm, ok := s.StateMachines[service.Game]
	if ok && stm.isReady() {
		s.logger.Errorf("fail to create stm with same game. config: %s", service.TOJSONString())
		s.lock.Unlock()
		return
	}

	// 构建stm实例
	stateMachine := buildStateMachine(service, s.StorageRoot, s.cli, s.ctx, s.logger, s.httpClient)
	s.StateMachines[service.Game] = stateMachine
	s.lock.Unlock()

	s.runSTM(stateMachine, true)
}

// 启动stm并调用其init方法
func (s *StateMachineManager) runSTM(stm *StateMachine, heartbeat bool) {
	// refresh config from this
	if 0 != len(stm.This.ID) {
		stm.Image = stm.This.Image

		stm.storageRoot = s.StorageRoot
		stm.storageGame = fmt.Sprintf("%s/%s", stm.storageRoot, stm.Game)
		stm.refreshStoragePath()

		stm.refreshPort()
	}

	appId, ports := stm.Run()
	if appId == "" || ports == nil {
		s.logger.Errorf("fail to run stm, appId: %s", appId)
		return
	}

	// 调用stm init接口
	authCode := s.generateAuthcode()
	s.callInit(ports[0].Host, stm.wsServer.GetURL(), authCode)
	stm.ready()

	// 是否启动心跳
	if heartbeat {
		stm.heartbeat()
	}

	// 保存stm实例
	s.lock.Lock()
	s.Mapping[appId] = ports[0].Host
	s.AuthMapping[appId] = authCode
	s.lock.Unlock()
}

func (s *StateMachineManager) callInit(dockerPortInt PortInt, wsUrl, authCode string) {
	path := fmt.Sprintf("http://0.0.0.0:%d/api/v1/%s", dockerPortInt, "init")
	values := url.Values{}
	values["url"] = []string{wsUrl}
	values["authCode"] = []string{authCode}
	s.logger.Infof("send init req:path:%s,values:%v", path, values)

	// keeping waiting
	// todo: timeout
	for {
		resp, err := http.PostForm(path, values)
		if err != nil {
			s.logger.Debug(err.Error())
			time.Sleep(200 * time.Millisecond)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			s.logger.Errorf("call init success: %s", string(body))

			return
		}
	}
}

func (s *StateMachineManager) generateAuthcode() string {
	rand.Seed(utility.GetTime().UnixNano())
	return strconv.Itoa(rand.Int())
}
