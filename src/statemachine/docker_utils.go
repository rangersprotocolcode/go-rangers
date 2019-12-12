package statemachine

import (
	"io/ioutil"
	"net/http"
	"encoding/json"
	"fmt"
	"x/src/middleware/types"
	dockerTypes "github.com/docker/docker/api/types"
	"time"
	"net"
	"github.com/docker/docker/client"
	"context"
	"x/src/common"
	"x/src/middleware/log"
	"sync"
	"strconv"
	"strings"
	"os"
	"x/src/middleware/notify"
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
	Config YAMLConfig

	// stm entities
	StateMachines map[string]StateMachine // key 为appId

	StorageRoot string

	// tool for connecting stm
	httpClient *http.Client

	Mapping     map[string]PortInt // key 为appId， value为端口号
	AuthMapping map[string]string  // key 为appId， value为authCode
	lock        sync.RWMutex

	// docker client
	cli *client.Client  //cli:  用于访问 docker 守护进程
	ctx context.Context //ctx:  传递本次操作的上下文信息

	logger log.Logger

	minerId string
}

// docker
func InitSTMManager(filename, minerId string) *StateMachineManager {
	if nil != STMManger {
		return STMManger
	}

	STMManger = &StateMachineManager{
		StateMachines: make(map[string]StateMachine),
	}
	STMManger.httpClient = createHTTPClient()
	STMManger.ctx = context.Background()
	STMManger.cli, _ = client.NewClientWithOpts(client.FromEnv)
	STMManger.cli.NegotiateAPIVersion(STMManger.ctx)

	STMManger.logger = log.GetLoggerByIndex(log.STMLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	STMManger.Mapping = make(map[string]PortInt)
	STMManger.AuthMapping = make(map[string]string)
	STMManger.minerId = minerId

	pwd, _ := os.Getwd()
	STMManger.StorageRoot = pwd + "/storage"
	STMManger.init(filename)

	// 订阅状态更新消息
	notify.BUS.Subscribe(notify.STMStorageReady, STMManger.updateSTMStorage)

	STMManger.logger.Infof("start success, minerId: %s", minerId)
	return STMManger
}

func (d *StateMachineManager) init(filename string) {
	d.buildConfig(filename)

	if 0 != len(d.Config.Services) {
		//todo : 根据Priority排序
		for _, service := range d.Config.Services {
			// 异步启动
			go d.loadStateMachine(service)
		}
	}
}

func (d *StateMachineManager) buildConfig(filename string) {
	// 加载配置文件
	// 配置文件的方式应该逐步废除
	if 0 != len(filename) {
		d.Config.InitFromFile(filename)
	}

	d.logger.Infof("get stm configs from file, %s", d.Config.TOJSONString())

	// 获取当前机器上已有的容器
	containers, _ := d.cli.ContainerList(d.ctx, dockerTypes.ContainerListOptions{All: true})
	if 0 == len(containers) {
		return
	}

	for _, container := range containers {
		name := container.Names[0]
		if !strings.HasPrefix(name, "/"+containerPrefix) {
			continue
		}

		index := -1
		for i, service := range d.Config.Services {
			if service.Image == container.Image {
				index = i
				break
			}
		}
		if -1 != index {
			d.Config.Services = append(d.Config.Services[:index], d.Config.Services[index+1:]...)
		}

		var config ContainerConfig
		nameSplited := strings.Split(name, "-")
		config.Game = nameSplited[1]
		config.Image = container.Image
		config.This = container
		d.Config.Services = append(d.Config.Services, config)
	}

	d.logger.Infof("get stm configs, merged by existing containers, %s", d.Config.TOJSONString())
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
		d.logger.Errorf("Validate authCode error! appId:%s,authCode:%s,expect:%s", appId, authCode, expect)
	}
	return expect == authCode
}

// 获取当前STM状态
func (s *StateMachineManager) GetStmStatus() map[string]map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	result := make(map[string]map[string]string)
	for appId, stm := range s.StateMachines {
		status := make(map[string]string, 3)
		status["status"] = stm.Status
		status["nonce"] = strconv.FormatUint(stm.RequestId, 10)
		status["storage"] = common.Bytes2Hex(stm.StorageStatus[:])
		result[appId] = status
	}

	s.logger.Errorf("GetStmStatus %v", result)
	return result
}
