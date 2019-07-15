package statemachine

import (
	"fmt"
	"encoding/json"
	"strings"
	"path/filepath"
	"os"
	"log"
	"io/ioutil"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"gopkg.in/yaml.v2"
	"github.com/docker/docker/api/types/container"
	"net/http"
	"net/url"
	"x/src/common"
	"sort"
)

//PortInt: 端口号类型
type PortInt uint16

//PortInt.String() : 将端口号转换为字符串
//  返回值: 字符串格式的端口号
func (pi PortInt) String() string {
	return fmt.Sprintf("%d", pi)
}

// NetworkConfig: To create network
// Name: 网络的名称
type NetworkConfig struct {
	Name string
}

// NetworkConfig.Name: 返回创建网络的名称
// 返回值:网络的名称
func (n *NetworkConfig) String() string {
	return n.Name
}

//    Priority      设定启动顺序
//    Game          游戏id
//    Name 			设定容器名
//    Image			string,设定镜像名
//    Detached 		bool,设定是否后台运行(不输出初始化日志记录),
//                  true 表示不输出初始化记录
//                  false 表示
//    WorkDir		设定容器工作目录
//    CMD           设定容器运行的命令
//    Net           配置容器的网络信息
//    Ports 		配置容器端口供外部访问
//                  支持挂载多端口
//    Volumes		配置挂载卷信息
//    AutoRemove    设定容器运行完毕后是否删除该容器
//                  true 表示自动删除
//                  false 表示不删除
type ContainerConfig struct {
	Priority   uint   `yaml:"priority"`
	Game       string `yaml:"game"`
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	Detached   bool   `yaml:"detached"`
	WorkDir    string `yaml:"work_dir"`
	CMD        string `yaml:"cmd"`
	Net        string `yaml:"net"`
	Ports      Ports  `yaml:"ports"`
	Volumes    Vols   `yaml:"volumes"`
	AutoRemove bool   `yaml:"auto_remove"`
	Import     string `yaml:"import"`
}

//将配置信息转换为 json 数据用于输出
//返回值: JSON 格式数据
//用于排查问题
func (c *ContainerConfig) JSONStr() string {
	res, e := json.Marshal(c)
	if e != nil {
		return ""
	} else {
		return string(res)
	}
}

//ContainerConfig.RunContainer: 从配置运行容器
//cli:  用于访问 docker 守护进程
//ctx:  传递本次操作的上下文信息
//net:  网络配置
func (c *ContainerConfig) RunContainer(cli *client.Client, ctx context.Context, containers []types.Container) (string, Ports) {
	// c.name 如果不申明，则默认为c.game
	if 0 == len(c.Name) {
		c.Name = c.Game
	}

	container := c.checkStatus(containers)
	if nil == container {
		common.DefaultLogger.Infof("Contain is nil.Create container!")
		return c.runContainer(cli, ctx)
	}

	common.DefaultLogger.Infof("Contain id:%s,state:%s", container.ID, container.Status)

	state := strings.ToLower(container.State)
	if "running" == container.State || strings.HasPrefix(state,"up"){
		var p uint16 = 0
		for _, port := range container.Ports {
			if port.PublicPort > p {
				p = port.PublicPort
			}
		}
		return c.Game, c.makePorts(p)
	}
	if "created" == container.State || strings.HasPrefix(state,"created"){
		cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true})

		// refresh container status
		containers, _ = cli.ContainerList(ctx, types.ContainerListOptions{All: true})
		return c.RunContainer(cli, ctx, containers)
	}

	if "exited" == container.State || strings.HasPrefix(state,"exited"){
		if err := cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); nil != err {
			panic(err)
		}

		// refresh container status
		containers, _ = cli.ContainerList(ctx, types.ContainerListOptions{All: true})

		return c.RunContainer(cli, ctx, containers)
	}

	if "paused" == container.State || strings.HasPrefix(state,"paused"){
		if err := cli.ContainerUnpause(ctx, container.ID); nil != err {
			panic(err)
		}

		var p uint16 = 0
		for _, port := range container.Ports {
			if port.PublicPort > p {
				p = port.PublicPort
			}
		}
		return c.Game, c.makePorts(p)
	}

	return "", nil
}

func (c *ContainerConfig) makePorts(port uint16) Ports {
	ports := make(Ports, 1)
	ports[0] = Port{Host: PortInt(port)}

	return ports
}

func (c *ContainerConfig) checkStatus(containers []types.Container) *types.Container {
	if nil == containers || 0 == (len(containers)) {
		return nil
	}

	for _, container := range containers {
		if container.Names[0] == fmt.Sprintf("/%s", c.Name) {
			return &container
		}
	}

	return nil
}

func (c *ContainerConfig) runContainer(cli *client.Client, ctx context.Context) (string, Ports) {
	//todo: load image if necessary
	//if 0 != len(c.Import) {
	//	file, err := os.Open(c.Import)
	//	if nil != err {
	//		readerCloser, _ := cli.ImageImport(ctx, types.ImageImportSource{Source: file, SourceName: c.Import}, "", types.ImageImportOptions{})
	//		readerCloser.Close()
	//	}
	//	//todo set image to ContainerConfig
	//}

	if 0 == len(c.Image) || 0 == len(c.Name) {
		common.DefaultLogger.Infof("skip to start image")
		return "", nil
	}

	//replace pwd to current abs dir
	c.Volumes.ReplacePWD()

	//set mount volumes
	vols := make([]string, len(c.Volumes))
	for index, item := range c.Volumes {
		vols[index] = item.String()
	}

	//set exposed ports for containers and publish ports
	exports := make(nat.PortSet)
	pts := make(nat.PortMap)

	sort.Sort(c.Ports)
	//配置端口映射数据结构
	for _, p := range c.Ports {
		tmpPort, _ := nat.NewPort("tcp", p.Target.String())
		pb := make([]nat.PortBinding, 0)
		pb = append(pb, nat.PortBinding{
			HostPort: p.Host.String(),
		})
		exports[tmpPort] = struct{}{}
		pts[tmpPort] = pb
	}

	mode := "default"
	if 0 != len(c.Net) {
		mode = c.Net
	}

	//创建容器
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        c.Image,
		ExposedPorts: exports,
		Cmd:          strings.Split(c.CMD, " "),
		WorkingDir:   c.WorkDir,
	}, &container.HostConfig{
		Binds:        vols,
		PortBindings: pts,
		NetworkMode:  container.NetworkMode(mode),
		AutoRemove:   c.AutoRemove,
	}, nil, c.Name)
	if err != nil {
		panic(err)
	}

	//遇到容器创建错误时发起 panic
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		common.DefaultLogger.Errorf(err.Error())
		panic(err)
	} else {
		common.DefaultLogger.Infof("Container %s is created and started.\n", resp.ID)
		// 创建成功 记录端口号与name的关联关系
		return c.Game, c.Ports
	}
}

//Port:端口映射信息数据
//Port.Host:宿主机端口
//Port.Target: 容器内部端口
type Port struct {
	Host   PortInt `yaml:"host"`
	Target PortInt `yaml:"target"`
}
type Ports []Port

//Port.String: 输出端口映射配列
func (p *Port) String() string {
	return fmt.Sprintf("%d:%d", p.Host, p.Target)
}

func (p Ports) Len() int {
	return len(p)
}
func (p Ports) Less(i, j int) bool {
	return p[i].Host > p[j].Host
}
func (p Ports) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

//Vol: 设置卷映射
//Vol.Host: 宿主机文件夹
//Vol.Target: 目标容器文件夹
type Vol struct {
	Host   string `yaml:"host"`
	Target string `yaml:"target"`
}

//Vol.String: 输出端口映射配列
func (v *Vol) String() string {
	return fmt.Sprintf("%s:%s", v.Host, v.Target)
}

//Vols: 储存多卷映射序列
type Vols []Vol

//ReplacePWD: 替换卷映射过程中的" pwd" 为当前工作目录
func (vs *Vols) ReplacePWD() {
	curDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	for i, v := range *vs {
		if strings.ToLower(v.Host[:3]) == "pwd" {

			(*vs)[i].Host = strings.Replace(v.Host, v.Host[:3], curDir, -1)

		}
	}
}

type ContainerConfigs []ContainerConfig

//YAMLConfig: 储存从 yaml 读取的配置信息
//Title: 配置名称
//Service: 服务(对应于容器)
type YAMLConfig struct {
	Title    string           `yaml:"title"`
	Services ContainerConfigs `yaml:"services"`
	cli      *client.Client
	ctx      context.Context
}

// Init toml from *.yaml
//filename: 文件名信息
func (t *YAMLConfig) InitFromFile(filename string, port uint) map[string]PortInt {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.UnmarshalStrict(yamlFile, t)
	if err != nil {
		log.Fatal(err)
	}

	t.cli, err = client.NewClientWithOpts(client.WithVersion("1.37"))
	if err != nil {
		panic(err)
	}

	t.ctx = context.Background()

	return t.runContainers(port)
}

//RunContainers: 从配置运行容器
//cli:  用于访问 docker 守护进程
//ctx:  传递本次操作的上下文信息
func (t *YAMLConfig) runContainers(port uint) map[string]PortInt {
	if 0 == len(t.Services) {
		return nil
	}

	containerList, _ := t.cli.ContainerList(t.ctx, types.ContainerListOptions{All: true})

	mapping := make(map[string]PortInt)
	//todo : 根据Priority排序
	for _, service := range t.Services {
		name, ports := service.RunContainer(t.cli, t.ctx, containerList)
		if name == "" || ports == nil {
			continue
		}
		mapping[name] = ports[0].Host
		//需要等到docker镜像启动完成
		//time.Sleep(time.Second * 10)
		t.setUrl(ports[0].Host, port)
	}

	return mapping
}
func (config *YAMLConfig) setUrl(portInt PortInt, layer2Port uint) {
	path := fmt.Sprintf("http://0.0.0.0:%d/api/v1/%s", portInt, "init")
	values := url.Values{}
	// todo : refactor statemachine sdk
	values["url"] = []string{fmt.Sprintf("http://%s:%d", "172.17.0.1", layer2Port)}
	//values["port"] = []string{strconv.FormatUint(uint64(layer2Port), 10)}
	if nil != common.DefaultLogger {
		common.DefaultLogger.Errorf("Send post req:path:%s,values:%v", path, values)
	}

	resp, err := http.PostForm(path, values)

	if err != nil {
		if nil != common.DefaultLogger {
			common.DefaultLogger.Errorf(err.Error())
		}
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
		return
	}

	if common.DefaultLogger != nil {
		common.DefaultLogger.Error(string(body))
	}
}
