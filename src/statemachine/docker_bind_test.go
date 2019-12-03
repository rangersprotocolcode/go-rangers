package statemachine

import (
	"testing"
	"github.com/docker/docker/client"
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types"
	"io"
	"os"
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
