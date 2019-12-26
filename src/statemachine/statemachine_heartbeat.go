package statemachine

import (
	"github.com/docker/docker/api/types"
	"time"
)

func (c *StateMachine) heartbeat() {
	go func() {
		for {
			if !c.heartBeat {
				return
			}

			if c.checkIfRunning() {
				c.setStatus(ready)
			} else if !c.isSync() && !c.isSynced() {
				c.stopped()
				c.logger.Errorf("stm stopped, id: %s, game: %s", c.This.ID, c.Game)
			}

			time.Sleep(1 * time.Second)
		}
	}()
}

// 根据ID刷新当前容器的配置
func (c *StateMachine) refreshThis() {
	// todo: 执行过docker rm命令后，container就查不到了
	containers, _ := c.cli.ContainerList(c.ctx, types.ContainerListOptions{All: true})
	if nil == containers || 0 == (len(containers)) {
		return
	}

	for _, container := range containers {
		if container.ID != c.This.ID {
			continue
		}
		c.This = container
		return
	}

}
