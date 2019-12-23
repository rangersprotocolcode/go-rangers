// 本文件包含statemachine结构定义
// statemachine的启动相关方法
package statemachine

import (
	"github.com/docker/docker/api/types/container"
	"strings"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"fmt"
	"github.com/docker/go-connections/nat"
	"sort"
	"context"
	"time"
	"x/src/middleware/log"
	"net/http"
	"encoding/json"
	"crypto/md5"
	"github.com/ipfs/go-ipfs-api"
	"math/rand"
	"x/src/utility"
)

const containerPrefix = "rp-"

type StateMachine struct {
	ContainerConfig

	// docker client
	cli *client.Client  `json:"-"`
	ctx context.Context `json:"-"`

	logger log.Logger `json:"-"`

	// 下载image用
	httpClient *http.Client `json:"-"`
	ipfsShell  *shell.Shell `json:"-"`

	// 与stm 通信用
	wsServer *wsServer `json:"-"`
	// 工作状态
	Status string

	// stm的存储与宿主机的映射
	storageRoot string   `json:"-"` // "${pwd}/storage"
	storageGame string   `json:"-"` // "${pwd}/storage/${appId}"
	storagePath []string `json:"-"`
	// 存储的状态值
	StorageStatus [md5.Size]byte `json:"storage"`
	RequestId     uint64         `json:"requestId"`
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

func buildStateMachine(c ContainerConfig, storageRoot string, cli *client.Client, ctx context.Context, logger log.Logger, httpClient *http.Client) *StateMachine {
	sh := shell.NewShell("localhost:5001")

	return &StateMachine{c, cli, ctx, logger, httpClient, sh,
		nil, preparing,
		storageRoot, fmt.Sprintf("%s/%s", storageRoot, c.Game), nil,
		[md5.Size]byte{}, 0}
}

// cli:  用于访问 docker 守护进程
// ctx:  传递本次操作的上下文信息
func (c *StateMachine) Run() (string, Ports) {
	// 从配置运行容器
	if 0 == len(c.This.ID) {
		c.logger.Infof("stm is nil, start to create. stm image: %s, game: %s", c.Image, c.Game)
		return c.runByConfig()
	}

	c.logger.Infof("existing stm id: %s,state: %s, image: %s, game: %s", c.This.ID, c.This.Status, c.Image, c.Game)

	state := strings.ToLower(c.This.State)
	if strings.HasPrefix(state, "running") || strings.HasPrefix(state, "up") {
		return c.runByExistedContainer()
	}

	cli := c.cli
	ctx := c.ctx

	// 比较危险
	if "created" == c.This.State || strings.HasPrefix(state, "created") {
		cli.ContainerRemove(ctx, c.This.ID, types.ContainerRemoveOptions{Force: true})
		c.This.ID = ""
		return c.Run()
	}

	if strings.Contains(state, "exited") {
		c.logger.Warnf("start exited stm, id: %s, image: %s", c.This.ID, c.Image)
		if err := cli.ContainerStart(ctx, c.This.ID, types.ContainerStartOptions{}); nil != err {
			c.logger.Errorf("fail to start stm. image: %s, error: %s", c.Image, err.Error())
			c.Status = "fail to start"
			return "", nil
		}

		return c.runByExistedContainer()
	}

	if strings.Contains(state, "paused") {
		c.logger.Warnf("start paused stm, id: %s, image: %s", c.This.ID, c.Image)
		if err := cli.ContainerUnpause(ctx, c.This.ID); nil != err {
			c.logger.Errorf("fail to unpause stm. image: %s, error: %s", c.Image, err.Error())
			c.Status = "fail to start"
			return "", nil
		}

		return c.runByExistedContainer()
	}

	c.failed()
	c.logger.Errorf("fail to start existing stm, id: %s, status: %s", c.This.ID, c.This.Status)
	return "", nil
}

// 根据已存在的容器
func (s *StateMachine) runByExistedContainer() (string, Ports) {
	s.waitUntilRun()

	// 刷新storagePath配置
	s.storagePath = make([]string, len(s.This.Mounts))
	for i, mount := range s.This.Mounts {
		s.storagePath[i] = fmt.Sprintf("%s:%s", mount.Source, mount.Destination)
	}

	// 启动ws服务器，供stm调用
	if s.wsServer == nil {
		s.wsServer = newWSServer(s.Game)
		s.wsServer.Start()
	}

	// 刷新端口
	var p uint16 = 0
	for _, port := range s.This.Ports {
		if port.PublicPort > p {
			p = port.PublicPort
		}
	}

	s.prepared()

	return s.Game, s.makePorts(p)
}

func (c *StateMachine) makePorts(port uint16) Ports {
	ports := make(Ports, 1)
	ports[0] = Port{Host: PortInt(port)}

	return ports
}

// 根据配置启动容器
func (c *StateMachine) runByConfig() (string, Ports) {
	if 0 == len(c.Image) {
		c.failed()
		c.logger.Errorf("skip to start image, stm config: %s", c.TOJSONString())
		return "", nil
	}

	// 本地没镜像，需要下载并加载镜像
	// 下载失败，启动失败
	if !c.checkImageExisted() && !c.download() {
		c.failed()
		c.logger.Errorf("cannot get image, stm config: %s", c.TOJSONString())
		return "", nil
	}

	c.logger.Debugf("image ready!")
	//set mount volumes
	c.storagePath = make([]string, len(c.Storage))
	for index, item := range c.Storage {
		c.storagePath[index] = fmt.Sprintf("%s/%s/%d:/%s", c.storageRoot, c.Game, index, item)
	}

	//set exposed ports for containers and publish ports
	exports := make(nat.PortSet)
	pts := make(nat.PortMap)

	sort.Sort(c.Ports)
	//配置端口映射数据结构
	for _, p := range c.Ports {
		tmpPort, _ := nat.NewPort("tcp", p.Target.String())
		pb := make([]nat.PortBinding, 0)
		if p.Host.String() == "0" {
			for {
				rand.Seed(int64(time.Now().UnixNano()))
				port := 9000 + int(rand.Float32()*1000)
				c.logger.Debugf("check port:%v", port)
				if !utility.PortInUse(port) {
					c.logger.Debugf("port not in use :%v", port)
					p.Host = PortInt(port)
					break
				}
			}

		}
		pb = append(pb, nat.PortBinding{
			HostPort: p.Host.String(),
		})
		exports[tmpPort] = struct{}{}
		pts[tmpPort] = pb
	}

	c.logger.Debugf("port:%v", c.Ports)
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
		Binds:        c.storagePath,
		PortBindings: pts,
		NetworkMode:  container.NetworkMode(mode),
		AutoRemove:   c.AutoRemove,
	}, nil, fmt.Sprintf("%s%s-%d", containerPrefix, c.Game, time.Now().UnixNano()))
	if err != nil {
		panic(err)
	}

	c.logger.Debugf("container creating")
	//遇到容器创建错误时发起 panic
	if err := c.cli.ContainerStart(c.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		c.failed()
		c.logger.Errorf("fail to start stm. config: %s, error: %s", c.TOJSONString(), err.Error())
		return "", nil
	}

	// 获取container实例
	containers, _ := c.cli.ContainerList(c.ctx, types.ContainerListOptions{All: true})
	for _, container := range containers {
		if container.ID == resp.ID {
			c.This = container
			break
		}

	}

	c.logger.Warnf("stm %s is created, waiting for running. image: %s, game: %s", resp.ID, c.Image, c.Game)
	c.waitUntilRun()

	// 启动ws服务器，供stm调用
	if c.wsServer == nil {
		c.wsServer = newWSServer(c.Game)
		c.wsServer.Start()
	}

	c.logger.Warnf("stm %s is created and prepared. image: %s, game: %s", resp.ID, c.Image, c.Game)
	c.prepared()

	return c.Game, c.Ports
}

// 检查container运行状态
func (c *StateMachine) checkIfRunning() bool {
	c.refreshThis()
	state := strings.ToLower(c.This.State)
	return strings.HasPrefix(state, "running") || strings.HasPrefix(state, "up")
}

// 检查本机是否有对应的docker镜像
func (s *StateMachine) checkImageExisted() bool {
	images, _ := s.cli.ImageList(s.ctx, types.ImageListOptions{})
	for _, image := range images {
		for _, repo := range image.RepoTags {
			if repo == s.Image {
				s.logger.Debugf("images has been found")
				return true
			}
		}
	}

	s.logger.Debugf("images not found")
	return false
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
