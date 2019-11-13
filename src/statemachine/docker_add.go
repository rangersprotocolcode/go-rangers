package statemachine

import "encoding/json"

// 通过交易的方式，添加stm
func (d *StateMachineManager) AddStatemachine(owner, config string) bool {
	if 0 == len(config) {
		d.logger.Errorf("fail to add statemachine, config: %s", config)
		return false
	}

	var containerConfig ContainerConfig
	err := json.Unmarshal([]byte(config), &containerConfig)
	if err != nil {
		d.logger.Errorf("fail to add statemachine, config: %s", config)
		return false
	}

	// check authority
	if containerConfig.Game != owner {
		d.logger.Errorf("fail to add statemachine, check owner failed. owner: %s, config: %s", owner, config)
		return false
	}

	// 异步加载新的状态机
	d.logger.Errorf("add new stateMachine, config: %s", containerConfig.TOJSONString())
	go d.runStateMachine(containerConfig)

	return true
}
