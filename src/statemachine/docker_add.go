package statemachine

import (
	"encoding/json"
	"x/src/middleware/notify"
	"strings"
	"x/src/network"
)

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

func (d *StateMachineManager) UpdateSTMStorage(appId, minerId string) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()

	stm, ok := d.StateMachines[appId]
	if !ok {
		d.logger.Errorf("fail to upload stm storage, appId: %s", appId)
		return false
	}

	stm.Stop()

	if minerId == d.minerId {
		zipFile := stm.uploadStorage()
		if 0 != len(zipFile) {
			// todo: 安全问题，需要签名
			msg := network.Message{Body: []byte(zipFile), Code: network.STMStorageReady}
			d.logger.Warnf("%s uploaded stm %s storage, filename: %s", minerId, stm.Game, zipFile)
			go network.GetNetInstance().Broadcast(msg)
		}

	}

	return true
}

func (d *StateMachineManager) updateSTMStorage(message notify.Message) {
	d.logger.Warnf("received uploaded stm storage, msg: %v", message)

	msg, ok := message.(*notify.STMStorageReadyMessage)
	if !ok {
		d.logger.Errorf("fail to get msg. %v", message)
		return
	}

	zipFile := string(msg.FileName)
	nameSplit := strings.Split(zipFile, "-")
	appId := nameSplit[0]

	d.lock.RLock()
	defer d.lock.RUnlock()

	stm, ok := d.StateMachines[appId]
	if !ok {
		d.logger.Errorf("fail to update stm storage, appId: %s", appId)
		return
	}

	stm.updateStorage(zipFile)
}
