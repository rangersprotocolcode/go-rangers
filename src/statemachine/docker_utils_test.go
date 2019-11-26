package statemachine

import (
	"testing"
	"x/src/common"
	"time"
	"fmt"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"strings"
)

func TestDockerInit(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	InitSTMManager("test.yaml", 8080)
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
	InitSTMManager("test1.yaml", 8080)
	time.Sleep(1000 * time.Minute)
}

func TestDownloadByContainer(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	InitSTMManager("test2.yaml", 8080)
	time.Sleep(1000 * time.Minute)
}

func TestParse(t *testing.T) {
	str := `{"stream":"Loaded image ID: sha256:00f6ec4b97ae644112f18a51927911bc06afbd4b395bb3771719883cfa64451e\n"}`
	index := strings.Index(str, "sha256:")

	fmt.Println(str[index+7 : index+64+7])
}
