package cli

import (
	"encoding/json"
	"log"
	"testing"
	"x/src/common"
)

func TestRPC(t *testing.T) {
	gx := NewGX()
	common.InitConf("tas.ini")
	walletManager = newWallets()
	gx.initMiner(0, true, "", "", "heavy", "")

	host := "127.0.0.1"
	var port uint = 8080
	StartRPC(host, port)
	tests := []struct {
		method string
		params []interface{}
	}{
		{"GTAS_newWallet", nil},
		{"GTAS_tx", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "0x5ca33e8ce7c3c97e0f7fa66db4371367e298621f", 1, ""}},
		{"GTAS_balance", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"GTAS_blockHeight", nil},
		{"GTAS_getWallets", nil},
		//{},
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
