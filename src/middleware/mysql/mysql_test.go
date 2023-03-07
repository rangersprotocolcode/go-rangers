package mysql

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"os"
	"testing"
)

func TestInitMySql(t *testing.T) {
	defer func() {
		os.RemoveAll("logs")
		os.RemoveAll("logs-0.db")
		os.RemoveAll("logs-0.db-shm")
		os.RemoveAll("logs-0.db-wal")
	}()

	//os.Mkdir("storage0",777)
	InitMySql()

	stmt, err := mysqlDBLog.Prepare("replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values(?,?,?,?,?,?,?,?,?,?,?)")
	result, err := stmt.Exec("1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11")
	if nil != err {
		t.Fatal(err)
	}

	fmt.Println(result.RowsAffected())
	fmt.Println(result.LastInsertId())

	stmt, err = mysqlDBLog.Prepare("replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values(?,?,?,?,?,?,?,?,?,?,?)")
	result, err = stmt.Exec("1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11")
	if nil != err {
		t.Fatal(err)
	}

	fmt.Println(result.RowsAffected())
	fmt.Println(result.LastInsertId())
}

func TestSyncOldData(t *testing.T) {
	os.RemoveAll("logs")
	os.RemoveAll("logs-0.db")
	os.RemoveAll("logs-0.db-shm")
	os.RemoveAll("logs-0.db-wal")

	common.Init(0, "1.ini", "robin")
	common.LocalChainConfig.MysqlDSN = "rpservice_v2:oJ2*bA0:hB3%@tcp(49.0.248.137:5555)/rpservice_v2?charset=utf8&parseTime=true&loc=Asia%2FShanghai"

	InitMySql()

	SyncOldData()

}

func TestSelectLogs(t *testing.T) {
	InitMySql()
	logs := SelectLogs(0, 100000000, nil)

	for _, log := range logs {
		fmt.Printf("%d %s %s\n", log.BlockNumber, log.BlockHash.String(),log.TxHash.Hex())
	}
}
