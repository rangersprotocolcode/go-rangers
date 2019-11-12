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
)

type StateMachine struct {
	ContainerConfig

	// docker containerId
	id string

	// docker client
	cli *client.Client
	ctx context.Context

	logger log.Logger
}

func buildStateMachine(c ContainerConfig, cli *client.Client, ctx context.Context, logger log.Logger) StateMachine {
	return StateMachine{c, "", cli, ctx, logger}
}

//将配置信息转换为 json 数据用于输出
//返回值: JSON 格式数据
//用于排查问题
func (c *StateMachine) JSONStr() string {
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
		c.logger.Warnf("Contain is nil.Create container!")
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
			return "", nil
		}

		return c.after(nil)
	}

	if strings.Contains(state, "paused") {
		if err := cli.ContainerUnpause(ctx, resp.ID); nil != err {
			c.logger.Errorf("fail to unpause container. image: %s, error: %s", c.Image, err.Error())
			return "", nil
		}

		return c.after(nil)
	}

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

	c.id = resp.ID
	c.waitUntilRun()

	var p uint16 = 0
	for _, port := range resp.Ports {
		if port.PublicPort > p {
			p = port.PublicPort
		}
	}

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
		return "", nil
	}

	// 本地没镜像，需要下载并加载镜像
	// 下载失败，启动失败
	if !c.checkImageExisted() && !c.download() {
		c.logger.Errorf("cannot get image, %s, downloadProtocol: %s, downloadUrl: %s", c.Image, c.DownloadProtocol, c.DownloadUrl)
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
		return "", nil
	}

	c.id = resp.ID
	c.waitUntilRun()

	c.logger.Warnf("Container %s is created and started. image: %s, game: %s", c.id, c.Image, c.Game)

	// 创建成功 记录端口号与name的关联关系
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
	switch strings.ToLower(s.DownloadProtocol) {
	case "pull":
		return s.downloadByPull()
	case "http":
		return s.downloadByHTTP()
	case "ipfs":
		return s.downloadByIPFS()
	}

	return false
}

func (s *StateMachine) downloadByIPFS() bool {
	return true
}

func (s *StateMachine) downloadByHTTP() bool {
	return true
}

func (s *StateMachine) downloadByPull() bool {
	s.logger.Errorf("start pull image: %s, downloadUrl: %s", s.Image, s.DownloadUrl)

	_, err := s.cli.ImagePull(s.ctx, s.Image, types.ImagePullOptions{})
	if err != nil {
		s.logger.Errorf("fail to pull image: %s, downloadUrl: %s. error: %s", s.Image, s.DownloadUrl, err.Error())
		return false
	}

	s.waitUntilImageExisted()
	s.logger.Errorf("end pull image: %s, downloadUrl: %s", s.Image, s.DownloadUrl)

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
