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
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	fmt.Println(runtime.GOOS)

	var tom = new(YAMLConfig)
	tom.InitFromFile("test.yaml")

	assertEqual(t, len(tom.Services), 2)

}

func TestDocker(t *testing.T) {
	var tom = InitSTMManager("test.yaml","daijia")

	nonce := tom.Nonce("j")
	fmt.Println(nonce)

	//fmt.Println(tom.Validate("j1", "0x1fc2119a6255817f8fe01f9200d0afbc3490fece0d1788901806cd6c7bf3e03b", "111"))
	output := tom.Process("j", "operator", 2222,
		"{\"timestamp\": 1537056003, \"msg_name\": \"arena_init\", \"msg_data\": {\"match_level\": 1, \"match_type\": 3, \"spec_type\": 0}}", nil)

	data, _ := json.Marshal(output)
	fmt.Println(string(data))
	assertEqual(t, len(tom.Config.Services), 2)

}

func assertEqual(t *testing.T, a, b interface{}) {
	if a != b {
		t.Errorf("Not Equal. %d %d", a, b)
	}
}

func TestString(t *testing.T) {
	str := "Up"

	fmt.Println(strings.EqualFold("up", str[0:2]))

	fmt.Println(strings.HasPrefix(strings.ToLower(str), "exited"))
}

func TestRand64(t *testing.T) {
	rand.Seed(int64(utility.GetTime().Unix()))
	i := rand.Int()

	fmt.Printf("%s", strconv.Itoa(i))
}

func TestFormatTime(t *testing.T){
	fmt.Println(utility.GetTime().Format("2006-01-02 15:04:05.999 -0700 MST"))
}