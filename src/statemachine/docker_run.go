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

// 通过配置文件，加载STM
func (s *StateMachineManager) runStateMachine(service ContainerConfig) {
	if 0 == len(service.Game) {
		s.logger.Errorf("fail to create stm with nil game. config: %s", service.TOJSONString())
		return
	}
	stateMachine := buildStateMachine(service, s.cli, s.ctx, s.logger)

	s.lock.Lock()
	s.StateMachines[service.Game] = &stateMachine
	s.lock.Unlock()

	appId, ports := stateMachine.Run()
	if appId == "" || ports == nil {
		return
	}

	// 调用stm init接口
	authCode := s.generateAuthcode()
	s.callInit(ports[0].Host, authCode)
	stateMachine.ready()

	s.lock.Lock()
	s.Mapping[appId] = ports[0].Host
	s.AuthMapping[appId] = authCode
	s.lock.Unlock()
}

func (s *StateMachineManager) callInit(dockerPortInt PortInt, authCode string) {
	path := fmt.Sprintf("http://0.0.0.0:%d/api/v1/%s", dockerPortInt, "init")
	values := url.Values{}
	values["url"] = []string{fmt.Sprintf("http://%s:%d", "172.17.0.1", s.layer2Port)}
	values["authCode"] = []string{authCode}
	s.logger.Infof("Send post req:path:%s,values:%v", path, values)

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
