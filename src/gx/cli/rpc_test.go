package cli

import (
	"encoding/json"
	"log"
	"testing"
	"x/src/common"
	"strconv"
	"fmt"
)

func TestRPC(t *testing.T) {
	gx := NewGX()
	common.InitConf("tas.ini")
	walletManager = newWallets()
	gx.initMiner(0, true, "", "", "heavy", "keystore")

	host := "127.0.0.1"
	var port uint = 8080
	StartRPC(host, port)
	tests := []struct {
		method string
		params []interface{}
	}{
		{"GTAS_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"1.25\",\"assetId\":[\"1\"],\"assetValue\":[\"dj\"]}]"}},
		{"GTAS_getBalance", []interface{}{"a","0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"GTAS_getAsset", []interface{}{"a","0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb","1"}},
		{"GTAS_getAllAssets", []interface{}{"a","0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"GTAS_updateAssets", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "[{\"address\":\"a\",\"balance\":\"0.25\",\"assetId\":[\"a1\",\"3\"],\"assetValue\":[\"dj11\",\"yy\"]}]"}},
		{"GTAS_getAsset", []interface{}{"a","0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb","1"}},
		{"GTAS_getAllAssets", []interface{}{"a","0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"GTAS_getBalance", []interface{}{"a","0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
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
		data, _ := json.Marshal(res.Result.Data)
		log.Printf("%s response data: %s", test.method, data)
	}
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
