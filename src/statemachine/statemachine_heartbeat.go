// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

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
