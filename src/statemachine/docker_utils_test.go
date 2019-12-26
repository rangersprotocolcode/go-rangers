package statemachine

import (
	"testing"
	"x/src/common"
	"time"
	"fmt"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"strings"
	"x/src/middleware/notify"
)

func TestDockerInit(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")

	notify.BUS = notify.NewBus()
	InitSTMManager("test.yaml", "daijia")

	//msg := notify.STMStorageReadyMessage{FileName: []byte("/ip4/192.168.0.101/tcp/4001/ipfs/QmU8Pu6hkzJY1P4JmtgJCgy3Z52rqdChjvkysPmrqWGEkM:QmaqSc7Y1Aw2pzE2KeNBkV1MqG58pU8HDGdmjJ4fuW7XQH:j-0-1575878863425150000.zip")}
	//STMManger.updateSTMStorage(&msg)


	time.Sleep(10 * time.Second)

	config:="{\"image\":\"tequiladj/stm:v1.0\",\"download_url\":\"tequiladj/stm:v1.0\",\"download_protocol\":\"pUlL\"}"
	STMManger.UpgradeSTM("j", config)
	time.Sleep(1000 * time.Minute)
}

func TestContainerConfig(t *testing.T) {
	var config ContainerConfig
	config.Image = "tequiladj/stm:v1.0"
	config.Priority = 1
	fmt.Println(config.Image)

	data, _ := json.Marshal(config)
	fmt.Println(string(data))

	str := `    priority: 0
    game: "j"
    name: "j1"
    image: "tequiladj/stm:v1.0"
    ports:
      - host: 8888
        target: 80
    detached: true
    downloadUrl: "tequiladj/stm:v1.0"
    downloadProtocol: "pUlL"`

	err := yaml.Unmarshal([]byte(str), &config)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("yaml success")
	}
	fmt.Println(config.Priority)

	var config2 ContainerConfig
	data, _ = json.Marshal(config)
	fmt.Println(string(data))

	json.Unmarshal(data, &config2)
	data, _ = json.Marshal(config2)
	fmt.Println(string(data))
}

func TestDownloadByFile(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	InitSTMManager("test1.yaml", "daijia")
	time.Sleep(1000 * time.Minute)
}

func TestDownloadByContainer(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	InitSTMManager("test2.yaml", "daijia")
	time.Sleep(1000 * time.Minute)
}

func TestParse(t *testing.T) {
	str := `{"stream":"Loaded image ID: sha256:00f6ec4b97ae644112f18a51927911bc06afbd4b395bb3771719883cfa64451e\n"}`
	index := strings.Index(str, "sha256:")

	fmt.Println(str[index+7 : index+64+7])
}

func TestStateMachineManager_AddStatemachine(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	InitSTMManager("", "daijia")

	//config:="{\"priority\":0,\"game\":\"0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54\",\"name\":\"genesis_test\",\"image\":\"littlebear234/genesis_image:latest\",\"hostname\":\"genesis_host_name\",\"detached\":true,\"work_dir\":\"\",\"cmd\":\"\",\"net\":\"\",\"ports\":[{\"host\":0,\"target\":0}],\"volumes\":null,\"auto_remove\":false,\"download_url\":\"littlebear234/genesis_image:latest\",\"download_protocol\":\"pull\"}"
	//config:="{\"priority\":0,\"game\":\"0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54\",\"name\":\"yeatol_genesis_test\",\"image\":\"yeatol/statemachine:test\",\"hostname\":\"yeatol_statemachine_test\",\"detached\":true,\"work_dir\":\"\",\"cmd\":\"/root/statemachine\",\"net\":\"\",\"ports\":[{\"host\":0,\"target\":80}],\"volumes\":null,\"auto_remove\":false,\"download_url\":\"yeatol/statemachine:test\",\"download_protocol\":\"pull\"}"
	//config := "{\"priority\":0,\"game\":\"yeatol\",\"name\":\"yeatol_genesis_test\",\"image\":\"yeatol/centos:7\",\"hostname\":\"yeatol_statemachine_test\",\"detached\":true,\"work_dir\":\"\",\"cmd\":\"/root/statemachine\",\"net\":\"\",\"ports\":[{\"host\":9231,\"target\":80}],\"storages\":[\"\"],\"auto_remove\":false,\"download_url\":\"yeatol/centos:7\",\"download_protocol\":\"pull\"}"
	config := "{\"priority\":0,\"game\":\"yeatol\",\"name\":\"yeatol_genesis_test\",\"image\":\"yeatol/statemachine:dev\",\"hostname\":\"yeatol_statemachine_test\",\"detached\":true,\"work_dir\":\"/root/docker\",\"cmd\":\"/root/statemachine\",\"net\":\"\",\"ports\":[{\"host\":9231,\"target\":80}],\"storages\":[\"/root/docker\"],\"auto_remove\":false,\"download_url\":\"yeatol/statemachine:dev\",\"download_protocol\":\"pull\"}"
	STMManger.AddStatemachine("yeatol", config)
	time.Sleep(1000 * time.Minute)
}
