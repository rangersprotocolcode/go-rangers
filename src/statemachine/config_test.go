package statemachine

import (
	"testing"
	"fmt"
	"strconv"
	"encoding/json"
)

func TestConfig(t *testing.T) {
	var tom = new(YAMLConfig)
	tom.InitFromFile("test.yaml")

	assertEqual(t, len(tom.Services), 2)

	tom.runContainers()
}

func TestDocker(t *testing.T) {
	var tom = new(DockerManager)
	tom.Filename = "test.yaml"
	tom.init()

	nonce := tom.Nonce("j")
	fmt.Println(nonce)

	//fmt.Println(tom.Validate("j1", "0x1fc2119a6255817f8fe01f9200d0afbc3490fece0d1788901806cd6c7bf3e03b", "111"))
	output := tom.Process("j", "operator", strconv.Itoa(nonce+1),
		"{\"timestamp\": 1537056003, \"msg_name\": \"arena_init\", \"msg_data\": {\"match_level\": 1, \"match_type\": 3, \"spec_type\": 0}}")

	data, _ := json.Marshal(output)
	fmt.Println(string(data))
	assertEqual(t, len(tom.Config.Services), 1)

}

func assertEqual(t *testing.T, a, b interface{}) {
	if a != b {
		t.Errorf("Not Equal. %d %d", a, b)
	}
}
