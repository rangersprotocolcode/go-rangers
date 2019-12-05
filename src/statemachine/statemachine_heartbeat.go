package statemachine

import "time"

func (c *StateMachine) heartbeat() {
	go func() {
		for {
			if c.checkIfRunning() {
				c.setStatus(ready)
			} else {
				c.stopped()
				c.logger.Errorf("stm stopped, id: %s, game: %s", c.This.ID, c.Game)
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()
}
