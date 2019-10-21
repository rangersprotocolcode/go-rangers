package statemachine

import (
	"io/ioutil"
	"net/http"
	"encoding/json"
	"fmt"
	"net/url"
	"x/src/middleware/types"
	"time"
	"net"
	"x/src/common"
	"strings"
)

var Docker *DockerManager

type DockerManager struct {
	Mapping     map[string]PortInt // key 为appId， value为端口号
	AuthMapping map[string]string  // key 为appId， value为authCode
	Config      YAMLConfig
	Filename    string
	httpClient  *http.Client
}

func DockerInit(filename string, port uint) {

	if nil != Docker {
		return
	}

	Docker = &DockerManager{
		Filename: filename,
	}
	Docker.init(port)

	Docker.httpClient = createHTTPClient()
}

const (
	maxIdleConns        int = 10
	maxIdleConnsPerHost int = 10
	idleConnTimeout     int = 90
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

func (d *DockerManager) init(layer2Port uint) {
	d.Mapping, d.AuthMapping = d.Config.InitFromFile(d.Filename, layer2Port)
}

//todo 这里入参需要改，改为payload,transfer
func (d *DockerManager) Process(name string, kind string, nonce string, payload string, tx *types.Transaction) *types.OutputMessage {
	prefix := d.getUrlPrefix(name)
	if 0 == len(prefix) {
		return nil
	}

	path := fmt.Sprintf("%sprocess", prefix)
	values := url.Values{}
	values["payload"] = []string{payload}
	values["transfer"] = []string{"Test transfer info"}

	resp, err := d.httpClient.PostForm(path, values)
	if err != nil {
		common.DefaultLogger.Debugf("Docker process post error.Path:%s,values:%v,error:%s", path, values, values, err.Error())
		return nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		common.DefaultLogger.Debugf("Docker process read response error:%s", err.Error())
		return nil
	}

	output := types.OutputMessage{}
	if err = json.Unmarshal(body, &output); nil != err {
		common.DefaultLogger.Debugf("Docker process result unmarshal error:%s", err.Error())
		return nil
	}

	return &output
}

func (d *DockerManager) Nonce(name string) int {
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

func (d *DockerManager) getUrlPrefix(name string) string {
	port := d.Mapping[name]
	if 0 == port {
		return ""
	}

	//call local container
	return fmt.Sprintf("http://0.0.0.0:%d/api/v1/", port)
}

func (d *DockerManager) GetType(gameId string) string {
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
func (d *DockerManager) IsGame(address string) bool {
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
func (d *DockerManager) ValidateAppId(appId, authCode string) bool {
	if 0 == len(appId) || 0 == len(authCode) {
		return false
	}

	expect := d.AuthMapping[appId]
	if 0 == len(expect) {
		return false
	}

	return expect == authCode
}
