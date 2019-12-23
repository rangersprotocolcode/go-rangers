package statemachine

import (
	"encoding/json"
	"x/src/middleware/notify"
	"strings"
	"x/src/network"
	"encoding/hex"
)

// 通过交易的方式，新增stm
func (d *StateMachineManager) AddStatemachine(owner, config string) bool {
	if 0 == len(config) {
		d.logger.Errorf("fail to add statemachine, config: %s", config)
		return false
	}

	var containerConfig ContainerConfig
	err := json.Unmarshal([]byte(config), &containerConfig)
	if err != nil {
		d.logger.Errorf("fail to add statemachine, config: %s,err:%s", config, err.Error())
		return false
	}

	// check authority
	if containerConfig.Game != owner {
		d.logger.Errorf("fail to add statemachine, check owner failed. owner: %s, config: %s", owner, config)
		return false
	}

	// 异步加载新的状态机
	d.logger.Errorf("add new stateMachine, config: %s", containerConfig.TOJSONString())
	d.loadStateMachine(containerConfig)

	return true
}

// 节点停stm
// 种子节点上传存储
func (d *StateMachineManager) UpdateSTMStorage(appId, minerId string) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()

	stm, ok := d.StateMachines[appId]
	if !ok {
		d.logger.Errorf("fail to update stm storage, appId: %s", appId)
		return false
	}

	stm.Stop()

	if minerId == d.minerId {
		stm.synced()
		zipFile := stm.uploadStorage()
		if 0 != len(zipFile) {
			// todo: 安全问题，需要签名
			msg := network.Message{Body: []byte(zipFile), Code: network.STMStorageReady}
			d.logger.Warnf("%s uploaded stm %s storage, filename: %s", minerId, stm.Game, zipFile)
			go network.GetNetInstance().Broadcast(msg)
		}
	} else {
		stm.sync()
	}

	return true
}

// 从发布者拉取存储状态并更新
func (d *StateMachineManager) updateSTMStorage(message notify.Message) {
	d.logger.Warnf("received uploaded stm storage, msg: %v", message)

	msg, ok := message.(*notify.STMStorageReadyMessage)
	if !ok {
		d.logger.Errorf("fail to get msg. %v", message)
		return
	}

	//fmt.Sprintf("%s:%s:%s", localID, cid, zipFile)
	data := string(msg.FileName)
	nameSplit := strings.Split(data, ":")
	if 3 != len(nameSplit) {
		d.logger.Errorf("wrong updateSTMStorage msg. %v", message)
		return
	}

	localID := nameSplit[0]
	cid := nameSplit[1]
	zipFile := nameSplit[2]

	//zipFile := fmt.Sprintf("%s-%d-%d.zip", c.Game, c.RequestId, time.Now().UnixNano())
	//fmt.Sprintf("%s-%d-%s-%d.zip", c.Game, c.RequestId, hex.EncodeToString(c.StorageStatus[:]), time.Now().UnixNano())
	zipFileSplit := strings.Split(zipFile, "-")
	appId := zipFileSplit[0]
	requestId := zipFileSplit[1]
	storageStatus := zipFileSplit[2]
	if 3 != len(zipFileSplit) {
		d.logger.Errorf("wrong updateSTMStorage msg. %v", message)
		return
	}

	d.lock.RLock()
	defer d.lock.RUnlock()

	stm, ok := d.StateMachines[appId]
	if !ok {
		d.logger.Errorf("fail to update stm storage, appId: %s", appId)
		return
	}

	if storageStatus != hex.EncodeToString(stm.StorageStatus[:]) {
		stm.updateStorage(localID, cid, zipFile, requestId)
	} else {
		d.logger.Warnf("same storage: %s", storageStatus)
	}
	stm.synced()
}

func (d *StateMachineManager) StartSTM(appId string) {
	d.lock.RLock()
	d.logger.Warnf("start stm, appId: %s", appId)

	stm, ok := d.StateMachines[appId]
	if !ok {
		d.logger.Errorf("fail to start stm, appId: %s", appId)
		d.lock.RUnlock()
		return
	}
	d.lock.RUnlock()

	if stm.isReady() {
		d.logger.Errorf("stm is already running, appId: %s", appId)
		return
	}

	d.runSTM(stm, false)
}

func (d *StateMachineManager) StopSTM(appId string) {
	d.lock.RLock()
	d.logger.Warnf("start stm, appId: %s", appId)

	stm, ok := d.StateMachines[appId]
	if !ok {
		d.logger.Errorf("fail to start stm, appId: %s", appId)
		d.lock.RUnlock()
		return
	}
	d.lock.RUnlock()

	stm.Stop()
}

func (d *StateMachineManager) UpgradeSTM(appId, downloadUrl, downloadProtocol string) {
	d.lock.RLock()
	d.logger.Warnf("start stm, appId: %s", appId)

	stm, ok := d.StateMachines[appId]
	if !ok {
		d.logger.Errorf("fail to start stm, appId: %s", appId)
		d.lock.RUnlock()
		return
	}
	d.lock.RUnlock()

	stm.Stop()
	stm.Remove()
}
