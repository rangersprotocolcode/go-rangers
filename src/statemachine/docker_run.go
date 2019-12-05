package statemachine

import (
	"fmt"
	"net/url"
	"net/http"
	"time"
	"io/ioutil"
	"math/rand"
	"strconv"
)

// 加载STM
func (s *StateMachineManager) runStateMachine(service ContainerConfig) {
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
	stateMachine := buildStateMachine(service, s.cli, s.ctx, s.logger, s.httpClient)
	s.StateMachines[service.Game] = &stateMachine
	s.lock.Unlock()

	appId, ports := stateMachine.Run()
	if appId == "" || ports == nil {
		return
	}

	// 调用stm init接口
	authCode := s.generateAuthcode()
	s.callInit(ports[0].Host, stateMachine.wsServer.GetURL(), authCode)
	stateMachine.ready()
	stateMachine.heartbeat()

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
			s.logger.Errorf("start success: %s", string(body))

			return
		}
	}
}

func (s *StateMachineManager) generateAuthcode() string {
	rand.Seed(int64(time.Now().UnixNano()))
	return strconv.Itoa(rand.Int())
}
