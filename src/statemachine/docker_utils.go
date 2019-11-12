package statemachine

import (
	"io/ioutil"
	"net/http"
	"encoding/json"
	"fmt"
	"x/src/middleware/types"
	"time"
	"net"
	"strings"
	"github.com/docker/docker/client"
	"context"
	"net/url"
	"x/src/common"
	"math/rand"
	"strconv"
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

var Docker *StateMachineManager

type StateMachineManager struct {
	// stm config
	Config   YAMLConfig
	Filename string

	// stm entities
	StateMachines map[string]StateMachine

	// tool for connecting stm
	httpClient  *http.Client
	Mapping     map[string]PortInt // key 为appId， value为端口号
	AuthMapping map[string]string  // key 为appId， value为authCode
	lock        sync.RWMutex

	// docker client
	cli *client.Client
	ctx context.Context

	logger log.Logger
}

func DockerInit(filename string, port uint) *StateMachineManager {
	if nil != Docker {
		return Docker
	}

	Docker = &StateMachineManager{
		Filename:      filename,
		StateMachines: make(map[string]StateMachine),
	}
	Docker.httpClient = createHTTPClient()
	Docker.ctx = context.Background()
	Docker.cli, _ = client.NewClientWithOpts(client.FromEnv)
	Docker.cli.NegotiateAPIVersion(Docker.ctx)
	Docker.logger = log.GetLoggerByIndex(log.DockerLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	Docker.Mapping = make(map[string]PortInt)
	Docker.AuthMapping = make(map[string]string)

	Docker.init(port)

	return Docker
}

func (d *StateMachineManager) init(layer2Port uint) {
	d.Config.InitFromFile(d.Filename)
	d.runStateMachines(layer2Port)
}

//RunContainers: 从配置运行容器
//cli:  用于访问 docker 守护进程
//ctx:  传递本次操作的上下文信息
func (d *StateMachineManager) runStateMachines(layer2Port uint) {
	if 0 == len(d.Config.Services) {
		return
	}

	//todo : 根据Priority排序
	for _, service := range d.Config.Services {
		// 异步启动
		go d.runStateMachine(service, layer2Port)

	}
}

// 通过配置文件，加载
func (d *StateMachineManager) runStateMachine(service ContainerConfig, layer2Port uint) {
	stateMachine := buildStateMachine(service, d.cli, d.ctx, d.logger)
	name, ports := stateMachine.Run()
	if name == "" || ports == nil {
		return
	}

	d.lock.Lock()
	d.StateMachines[name] = stateMachine
	d.Mapping[name] = ports[0].Host
	authCode := d.generateAuthcode()
	d.AuthMapping[name] = authCode
	d.lock.Unlock()

	//需要等到docker镜像启动完成
	d.callInit(ports[0].Host, layer2Port, authCode)
}

func (d *StateMachineManager) callInit(dockerPortInt PortInt, layer2Port uint, authCode string) {
	path := fmt.Sprintf("http://0.0.0.0:%d/api/v1/%s", dockerPortInt, "init")
	values := url.Values{}
	values["url"] = []string{fmt.Sprintf("http://%s:%d", "172.17.0.1", layer2Port)}
	values["authCode"] = []string{authCode}
	d.logger.Infof("Send post req:path:%s,values:%v", path, values)

	for {
		resp, err := http.PostForm(path, values)
		if err != nil {
			d.logger.Infof(err.Error())
			time.Sleep(200 * time.Millisecond)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			d.logger.Error(fmt.Sprintf("start success: %s", string(body)))

			return
		}
	}
}

func (d *StateMachineManager) generateAuthcode() string {
	rand.Seed(int64(time.Now().UnixNano()))
	return strconv.Itoa(rand.Int())
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

func (d *StateMachineManager) GetType(gameId string) string {
	configs := d.Config.Services
	if 0 == len(configs) {
		return ""
	}

	for _, value := range configs {
		if 0 == strings.Compare(value.Game, gameId) {
			return value.Type
		}
	}

	return ""
}

// 判断是否是游戏地址
// todo: 这里只判断了本地运行的statemachine，会有漏洞
func (d *StateMachineManager) IsGame(address string) bool {
	configs := d.Config.Services
	if 0 == len(configs) {
		return false
	}

	for _, value := range configs {
		if 0 == strings.Compare(value.Game, address) {
			return true
		}
	}

	return false
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

