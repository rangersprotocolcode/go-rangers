package statemachine

import (
	"github.com/docker/docker/api/types/container"
	"strings"
	"encoding/json"
	"x/src/common"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"fmt"
	"github.com/docker/go-connections/nat"
	"sort"
	"context"
	"time"
)

type StateMachine struct {
	ContainerConfig

	// docker containerId
	id string

	// docker client
	cli *client.Client
	ctx context.Context
}

func buildStateMachine(c ContainerConfig, cli *client.Client, ctx context.Context) StateMachine {
	return StateMachine{c, "", cli, ctx}
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

func (c *StateMachine) log(msg string) {
	if nil != common.DefaultLogger {
		common.DefaultLogger.Info(msg)
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
		c.log("Contain is nil.Create container!")
		return c.runContainer()
	}

	c.log(fmt.Sprintf("Contain id:%s,state:%s", resp.ID, resp.Status))

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
			panic(err)
		}

		return c.after(nil)
	}

	if strings.Contains(state, "paused") {
		if err := cli.ContainerUnpause(ctx, resp.ID); nil != err {
			panic(err)
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
			return &container
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

func (c *StateMachine) waitUntilRun() {
	for {
		if c.checkIfRunning() {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
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
		c.log("skip to start image")
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
		c.log(err.Error())
		panic(err)
	}

	c.id = resp.ID
	c.waitUntilRun()

	c.log(fmt.Sprintf("Container %s is created and started.\n", resp.ID))

	// 创建成功 记录端口号与name的关联关系
	return c.Game, c.Ports
}
