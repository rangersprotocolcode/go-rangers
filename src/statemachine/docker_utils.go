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

	// docker client
	cli *client.Client
	ctx context.Context
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

	Docker.init(port)

	return Docker
}

func (d *StateMachineManager) init(layer2Port uint) {
	d.Config.InitFromFile(d.Filename)
	d.Mapping, d.AuthMapping = d.runStateMachines(layer2Port)
}

//RunContainers: 从配置运行容器
//cli:  用于访问 docker 守护进程
//ctx:  传递本次操作的上下文信息
func (d *StateMachineManager) runStateMachines(port uint) (map[string]PortInt, map[string]string) {
	if 0 == len(d.Config.Services) {
		return nil, nil
	}

	mapping := make(map[string]PortInt)
	authCodeMapping := make(map[string]string)

	//todo : 根据Priority排序
	for _, service := range d.Config.Services {
		stateMachine := buildStateMachine(service, d.cli, d.ctx)

		name, ports := stateMachine.Run()
		if name == "" || ports == nil {
			continue
		}

		d.StateMachines[name] = stateMachine
		mapping[name] = ports[0].Host
		authCode := d.generateAuthcode()
		authCodeMapping[name] = authCode

		//需要等到docker镜像启动完成
		d.callInit(ports[0].Host, port, authCode)
	}

	return mapping, authCodeMapping
}

func (d *StateMachineManager) callInit(dockerPortInt PortInt, layer2Port uint, authCode string) {
	path := fmt.Sprintf("http://0.0.0.0:%d/api/v1/%s", dockerPortInt, "init")
	values := url.Values{}
	values["url"] = []string{fmt.Sprintf("http://%s:%d", "172.17.0.1", layer2Port)}
	values["authCode"] = []string{authCode}
	if nil != common.DefaultLogger {
		common.DefaultLogger.Errorf("Send post req:path:%s,values:%v", path, values)
	}

	for {
		resp, err := http.PostForm(path, values)
		if err != nil {
			if nil != common.DefaultLogger {
				common.DefaultLogger.Infof(err.Error())
			}

			time.Sleep(200 * time.Millisecond)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if common.DefaultLogger != nil {
				common.DefaultLogger.Error(fmt.Sprintf("start success: %s", string(body)))
			}

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
	port := d.Mapping[name]
	if 0 == port {
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
	common.DefaultLogger.Debugf("Validate app id! Appid:%s,authCode:%s", appId, authCode)
	if 0 == len(appId) || 0 == len(authCode) {
		return false
	}

	expect := d.AuthMapping[appId]
	if 0 == len(expect) {
		common.DefaultLogger.Debugf("Validate wrong")
		return false
	}

	return expect == authCode
}
