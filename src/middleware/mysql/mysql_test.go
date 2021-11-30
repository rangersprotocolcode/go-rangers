package mysql

import (
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"testing"
)

func TestGetTxRaws(t *testing.T) {
	notify.BUS = notify.NewBus()
	InitMySql("rpservice_dev:!890rpServiceDev@tcp(api.tuntunhz.com:3336)/rpservice_dev?charset=utf8&parseTime=true&loc=Asia%2FShanghai")
	if nil == MysqlDB {
		t.Errorf("fail to open mysql")
	}
	defer MysqlDB.Close()

	txs := GetTxRaws(0)
	fmt.Println(len(txs))
	tx := txs[0]

	var transaction types.Transaction
	// jsonrpc请求
	if 0 == len(tx.UserId) {

	} else {
		var txJson types.TxJson
		err := json.Unmarshal(utility.StrToBytes(tx.Data), &txJson)
		if nil != err {
			msg := fmt.Sprintf("handleClientMessage json unmarshal client message error:%s", err.Error())
			t.Fatal(msg)
		}
		transaction = txJson.ToTransaction()
	}

	fmt.Println(transaction.Sign.GetHexString())
}
