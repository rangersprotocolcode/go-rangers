package statemachine

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"os"
	"testing"
	"time"
)

func TestDocker_Bind(t *testing.T) {
	ctx := context.Background()
	cli, _ := client.NewClientWithOpts(client.FromEnv)
	str, _ := os.Getwd()
	path := str + "/logs:/tmp"
	//创建容器
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine:latest",
		//ExposedPorts: exports,
		Cmd:        []string{"echo", "hello"},
		WorkingDir: "/root",
		//Hostname:     c.Hostname,
	}, &container.HostConfig{
		Binds: []string{path},
		//PortBindings: pts,
		//NetworkMode:  container.NetworkMode(mode),
		AutoRemove: false,
	}, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, out)
}

func TestDocker_Container(t *testing.T) {
	ctx := context.Background()
	cli, _ := client.NewClientWithOpts(client.FromEnv)

	containers, _ := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	data, _ := json.Marshal(containers)
	fmt.Println(string(data))

}

func TestTicker(t *testing.T) {
	ticker := time.NewTicker(time.Millisecond * 500)
	go func() {
		for {
			<-ticker.C
			fmt.Println("Tick at")
		}
	}()
	time.Sleep(time.Millisecond * 1500)
	ticker.Stop()
	fmt.Println("Ticker stopped")
}
