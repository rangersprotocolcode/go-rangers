package statemachine

import (
	"io/ioutil"
	"net/http"
	"encoding/json"
	"fmt"
	"x/src/middleware/types"
	"time"
	"net"
	"github.com/docker/docker/client"
	"context"
	"x/src/common"
	"x/src/middleware/log"
	"sync"
)

const (
	maxIdleConns        = 10
	maxIdleConnsPerHost = 10
	idleConnTimeout     = 90
)

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(idleConnTimeout) * time.Second,
		},
	}

	return client
}

var STMManger *StateMachineManager

type StateMachineManager struct {
	// stm config
	Config   YAMLConfig
	Filename string // 配置文件名称

	// stm entities
	StateMachines map[string]*StateMachine

	// tool for connecting stm
	httpClient  *http.Client
	Mapping     map[string]PortInt // key 为appId， value为端口号
	AuthMapping map[string]string  // key 为appId， value为authCode
	lock        sync.RWMutex

	// docker client
	cli *client.Client  //cli:  用于访问 docker 守护进程
	ctx context.Context //ctx:  传递本次操作的上下文信息

	layer2Port uint // layer2 节点本机端口，用于给状态机提供服务

	logger log.Logger
}

// docker
func InitSTMManager(filename string, layer2Port uint) *StateMachineManager {
	if nil != STMManger {
		return STMManger
	}

	STMManger = &StateMachineManager{
		Filename:      filename,
		StateMachines: make(map[string]*StateMachine),
	}
	STMManger.httpClient = createHTTPClient()
	STMManger.ctx = context.Background()
	STMManger.cli, _ = client.NewClientWithOpts(client.FromEnv)
	STMManger.cli.NegotiateAPIVersion(STMManger.ctx)

	STMManger.logger = log.GetLoggerByIndex(log.STMLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	STMManger.Mapping = make(map[string]PortInt)
	STMManger.AuthMapping = make(map[string]string)
	STMManger.layer2Port = layer2Port

	STMManger.init()

	return STMManger
}

func (d *StateMachineManager) init() {
	d.Config.InitFromFile(d.Filename)
	d.runStateMachines()
}

func (d *StateMachineManager) runStateMachines() {
	if 0 == len(d.Config.Services) {
		return
	}

	//todo : 根据Priority排序
	for _, service := range d.Config.Services {
		// 异步启动
		go d.runStateMachine(service)

	}
}

func (d *StateMachineManager) Nonce(name string) int {
	prefix := d.getUrlPrefix(name)
	if 0 == len(prefix) {
		return -1
	}

	url := fmt.Sprintf("%snonce", prefix)
	resp, err := d.httpClient.Get(url)
	if err != nil {
		// handle error
		return -2
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
		return -3
	}

	nonce := types.Nonce{}
	if err = json.Unmarshal(body, &nonce); err != nil {
		return -4
	}

	return nonce.Nonce
}

func (d *StateMachineManager) getUrlPrefix(name string) string {
	d.lock.RLock()
	defer d.lock.RUnlock()

	port, ok := d.Mapping[name]
	if !ok {
		d.logger.Errorf("fail to find appId: %s", name)
		return ""
	}

	//call local container
	return fmt.Sprintf("http://0.0.0.0:%d/api/v1/", port)
}

// 检查authCode是否合法
func (d *StateMachineManager) ValidateAppId(appId, authCode string) bool {
	if 0 == len(appId) || 0 == len(authCode) {
		return false
	}

	d.lock.RLock()
	defer d.lock.RUnlock()

	expect := d.AuthMapping[appId]
	if 0 == len(expect) {
		d.logger.Debugf("Validate wrong")
		return false
	}
	if expect != authCode {
		d.logger.Errorf("Validate authCode error! appid:%s,authCode:%s,expect:%s", appId, authCode, expect)
	}
	return expect == authCode
}

// 获取当前STM状态
func (s *StateMachineManager) GetStmStatus() map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	result := make(map[string]string)
	for appId, stm := range s.StateMachines {
		result[appId] = stm.Status
	}

	return result
}
