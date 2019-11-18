package statemachine

import (
	"github.com/docker/docker/api/types/container"
	"strings"
	"encoding/json"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"fmt"
	"github.com/docker/go-connections/nat"
	"sort"
	"context"
	"time"
	"x/src/middleware/log"
	"net/http"
	"os"
	"bufio"
	"io"
	"io/ioutil"
)

const (
	preparing    = "preparing(开始创建)"
	failToCreate = "failToCreate(创建失败，请检查配置文件)"
	prepared     = "prepared(创建成功，初始化中)"
	ready        = "ready(正常服务)"
)

type StateMachine struct {
	ContainerConfig

	// docker containerId
	Id string `json:"containerId"`

	// docker client
	cli *client.Client  `json:"-"`
	ctx context.Context `json:"-"`

	logger log.Logger `json:"-"`

	// 工作状态
	// todo: 心跳检测
	Status string

	httpClient *http.Client `json:"-"`
}

func buildStateMachine(c ContainerConfig, cli *client.Client, ctx context.Context, logger log.Logger, httpClient *http.Client) StateMachine {
	return StateMachine{c, "", cli, ctx, logger, preparing, httpClient}
}

//将配置信息转换为 json 数据用于输出
//返回值: JSON 格式数据
//用于排查问题
func (c *StateMachine) TOJSONString() string {
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
func (c *StateMachine) Run() (string, Ports) {
	// c.name 如果不申明，则默认为c.game
	if 0 == len(c.Name) {
		c.Name = c.Game
	}

	cli := c.cli
	ctx := c.ctx
	resp := c.getContainer()
	if nil == resp {
		c.logger.Warnf("Contain is nil, start to create. stm image: %s, game: %s", c.Image, c.Game)
		return c.runContainer()
	}

	c.logger.Warnf("Contain id: %s,state: %s, image: %s, game: %s", resp.ID, resp.Status, c.Image, c.Game)

	state := strings.ToLower(resp.State)
	if strings.HasPrefix(state, "running") || strings.HasPrefix(state, "up") {
		return c.after(resp)
	}

	if "created" == resp.State || strings.HasPrefix(state, "created") {
		cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return c.Run()
	}

	if strings.Contains(state, "exited") {
		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); nil != err {
			c.logger.Errorf("fail to start container. image: %s, error: %s", c.Image, err.Error())
			c.Status = "fail to start"
			return "", nil
		}

		return c.after(nil)
	}

	if strings.Contains(state, "paused") {
		if err := cli.ContainerUnpause(ctx, resp.ID); nil != err {
			c.logger.Errorf("fail to unpause container. image: %s, error: %s", c.Image, err.Error())
			c.Status = "fail to start"
			return "", nil
		}

		return c.after(nil)
	}

	c.Status = failToCreate
	return "", nil
}

// 根据名字查找当前容器的配置
func (c *StateMachine) getContainer() *types.Container {
	containers, _ := c.cli.ContainerList(c.ctx, types.ContainerListOptions{All: true})
	if nil == containers || 0 == (len(containers)) {
		return nil
	}

	for _, container := range containers {
		if container.Names[0] == fmt.Sprintf("/%s", c.Name) {
			if container.Image == c.Image {
				return &container
			}

			c.cli.ContainerStop(c.ctx, container.ID, nil)
			c.cli.ContainerRemove(c.ctx, container.ID, types.ContainerRemoveOptions{Force: true})
			return nil
		}
	}

	return nil
}

func (c *StateMachine) after(existed *types.Container) (string, Ports) {
	resp := existed
	if nil == resp {
		resp = c.getContainer()
	}

	c.Id = resp.ID
	c.waitUntilRun()

	var p uint16 = 0
	for _, port := range resp.Ports {
		if port.PublicPort > p {
			p = port.PublicPort
		}
	}

	c.Status = prepared
	return c.Game, c.makePorts(p)
}

func (c *StateMachine) checkIfRunning() bool {
	container := c.getContainer()
	state := strings.ToLower(container.State)
	return strings.HasPrefix(state, "running") || strings.HasPrefix(state, "up")
}

func (c *StateMachine) makePorts(port uint16) Ports {
	ports := make(Ports, 1)
	ports[0] = Port{Host: PortInt(port)}

	return ports
}

func (c *StateMachine) runContainer() (string, Ports) {
	if 0 == len(c.Image) || 0 == len(c.Name) {
		c.logger.Errorf("skip to start image")
		c.Status = failToCreate
		return "", nil
	}

	// 本地没镜像，需要下载并加载镜像
	// 下载失败，启动失败
	if !c.checkImageExisted() && !c.download() {
		c.Status = failToCreate
		c.logger.Errorf("cannot get image, stm config: %s", c.TOJSONString())
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
	resp, err := c.cli.ContainerCreate(c.ctx, &container.Config{
		Image:        c.Image,
		ExposedPorts: exports,
		Cmd:          strings.Split(c.CMD, " "),
		WorkingDir:   c.WorkDir,
		Hostname:     c.Hostname,
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
	if err := c.cli.ContainerStart(c.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		c.logger.Errorf("fail to start container. image: %s, error: %s", c.Image, err.Error())
		c.Status = failToCreate
		return "", nil
	}

	c.Id = resp.ID
	c.waitUntilRun()

	c.logger.Warnf("Container %s is created and started. image: %s, game: %s", c.Id, c.Image, c.Game)

	c.Status = prepared
	return c.Game, c.Ports
}

// 检查本机是否有对应的docker镜像
func (s *StateMachine) checkImageExisted() bool {
	images, _ := s.cli.ImageList(s.ctx, types.ImageListOptions{})
	for _, image := range images {
		for _, repo := range image.RepoTags {
			if repo == s.Image {
				return true
			}
		}
	}

	return false
}

func (s *StateMachine) download() bool {
	s.logger.Warnf("start download stm: %s, downloadUrl: %s, downloadProtocol: %s", s.Image, s.DownloadUrl, s.DownloadProtocol)
	result := false
	switch strings.ToLower(s.DownloadProtocol) {
	case "pull":
		result = s.downloadByPull()
		break
	case "file":
		result = s.downloadByFile(true)
		break
	case "filecontainer":
		result = s.downloadByFile(false)
		break
	case "ipfs":
		result = s.downloadByIPFS(true)
		break
	case "ipfscontainer":
		result = s.downloadByIPFS(false)
		break
	}

	s.logger.Warnf("end download stm: %s, downloadUrl: %s, downloadProtocol: %s, result: %t", s.Image, s.DownloadUrl, s.DownloadProtocol, result)
	return result
}

func (s *StateMachine) downloadByIPFS(isImage bool) bool {
	return true
}

func (s *StateMachine) downloadByFile(isImage bool) bool {
	url := s.DownloadUrl
	if 0 == len(url) {
		return false
	}

	isHttp := strings.HasPrefix(url, "http")
	if isHttp {
		response, err := s.httpClient.Get(url)
		if nil != err {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}

		err = s.loadOrImport(isImage, response.Body)
		if err != nil {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}
		defer response.Body.Close()
	} else {
		file, err := os.Open(url)
		if nil != err {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}
		r := bufio.NewReader(file)

		err = s.loadOrImport(isImage, r)
		if err != nil {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}
		defer file.Close()
	}

	return true
}

func (s *StateMachine) loadOrImport(isImage bool, reader io.Reader) error {
	if isImage {
		response, err := s.cli.ImageLoad(s.ctx, reader, true)
		if nil != err {
			return err
		}
		defer response.Body.Close()

		body, _ := ioutil.ReadAll(response.Body)
		result := string(body)
		s.logger.Warnf("ImageLoaded, result: %s", result)

		if !s.checkImageExisted() {
			s.cli.ImageTag(s.ctx, s.parse(result), s.Image)
		}

	} else {
		response, err := s.cli.ImageImport(s.ctx, types.ImageImportSource{Source: reader, SourceName: "-"}, "", types.ImageImportOptions{})
		if nil != err {
			return err
		}
		defer response.Close()

		body, _ := ioutil.ReadAll(response)
		result := string(body)
		s.logger.Warnf("ImageImported, result: %s", result)

		if !s.checkImageExisted() {
			s.cli.ImageTag(s.ctx, s.parse(result), s.Image)
		}
	}

	return nil
}

func (s *StateMachine) downloadByPull() bool {
	_, err := s.cli.ImagePull(s.ctx, s.Image, types.ImagePullOptions{})
	if err != nil {
		s.logger.Warnf("fail to pull image: %s, downloadUrl: %s. error: %s", s.Image, s.DownloadUrl, err.Error())
		return false
	}

	s.waitUntilImageExisted()
	return true
}

func (s *StateMachine) waitUntilImageExisted() {
	s.waitUntilCondition(s.checkImageExisted)
}

func (s *StateMachine) waitUntilRun() {
	s.logger.Errorf("wait image until run. image: %s, appId: %s", s.Image, s.Game)
	s.waitUntilCondition(s.checkIfRunning)
	s.logger.Errorf("image running. image: %s, appId: %s", s.Image, s.Game)
}

func (s *StateMachine) waitUntilCondition(condition func() bool) {
	for {
		if condition() {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (s *StateMachine) ready() {
	s.Status = ready
}

func (machine *StateMachine) parse(s string) string {
	index := strings.Index(s, "sha256:")
	return s[index+7 : index+64+7]
}
