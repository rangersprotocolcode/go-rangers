package cli

import (
	"encoding/json"
	"log"
	"testing"
	"strconv"
	"fmt"
	"time"
	"x/src/common"
	"crypto/md5"
	"encoding/binary"
)

func TestRPC(t *testing.T) {
	gx := NewGX()
	common.InitConf("tas.ini")
	walletManager = newWallets()
	gx.initMiner(0, "heavy", "keystore", 8080)

	host := "0.0.0.0"
	var port uint = 8989
	if err := StartRPC(host, port); err != nil {
		panic(err)
	}

	tests := []struct {
		method string
		params []interface{}
	}{
		{"Rocket_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"50\",\"assets\":{\"1\":\"dj\"}}]", 1}},
		{"Rocket_getBalance", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_getAsset", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "1"}},
		{"Rocket_getAllAssets", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"-2.25\",\"assets\":{\"1\":\"dj11\",\"2\":\"yyyy\"}}]", 2}},
		{"Rocket_getAsset", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "1"}},
		{"Rocket_getAllAssets", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_getBalance", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"2.25\",\"assets\":{\"1\":\"\",\"2\":\"yyyy\"}}]", 3}},
		{"Rocket_getAsset", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "1"}},
		{"Rocket_getAllAssets", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_getBalance", []interface{}{"a", "0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"Rocket_notify", []interface{}{"tuntun", "a19d069d48d2e9392ec2bb41ecab0a72119d633b","notify one"}},
		//{"Rocket_notifyGroup", []interface{}{"tuntun", "groupA","notify groupA"}},
		//{"Rocket_notifyBroadcast", []interface{}{"tuntun", "notify all"}},
		//{"GTAS_blockHeight", nil},
		//{"GTAS_getWallets", nil},
	}
	for _, test := range tests {
		res, err := rpcPost(host, port, test.method, test.params...)
		if err != nil {
			t.Errorf("%s failed: %v", test.method, err)
			continue
		}
		if res.Error != nil {
			t.Errorf("%s failed: %v", test.method, res.Error.Message)
			continue
		}
		if nil!=res && nil!=res.Result{
			data, _ := json.Marshal(res.Result.Data)
			log.Printf("%s response data: %s", test.method, data)
		}

	}

	time.Sleep(10000 * time.Second)
}

func TestStrToFloat(t *testing.T) {
	var a = "11.23456"
	b, _ := strconv.ParseFloat(a, 64)
	fmt.Printf("float :%v\n", b)

	c := strconv.FormatFloat(b, 'E', -1, 64)
	fmt.Printf("string :%v\n", c)
}

func TestJSONString(t *testing.T) {
	m := make(map[string]string, 0)

	m["a"] = "b"
	//m["aa"] = "bc"
	data, _ := json.Marshal(m)
	log.Printf("response data: %s", data)

	mm := make(map[string]string, 0)
	json.Unmarshal(data, &mm)

	log.Printf("mm response data: %s", mm)

}

func TestSlice(t *testing.T) {
	a := []int{0, 1, 2, 3, 4}
	//删除第i个元素
	i := 4
	a = append(a[:i], a[i+1:]...)
	fmt.Println(a)
}

func TestNotifyId(t *testing.T) {
	gameId := "a"
	userId := "1"

	data := []byte(gameId)
	if 0 != len(userId) {
		data = append(data, []byte(userId)...)
	}

	md5Result := md5.Sum(data)
	idBytes := md5Result[4:12]

	id := uint64(binary.BigEndian.Uint64(idBytes))

	fmt.Println(id)
}
