package statemachine

import (
	"time"
	"github.com/docker/docker/api/types"
)

func (c *StateMachine) heartbeat() {
	go func() {
		for {
			c.refreshThis()
			if c.checkIfRunning() {
				c.setStatus(ready)
			} else {
				c.stopped()
				c.logger.Errorf("stm stopped, id: %s, game: %s", c.This.ID, c.Game)
			}

			time.Sleep(1 * time.Second)
		}
	}()
}

// 根据ID刷新当前容器的配置
func (c *StateMachine) refreshThis() {
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
