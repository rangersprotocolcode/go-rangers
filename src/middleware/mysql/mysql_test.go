package mysql

import (
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
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()

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

func TestSelectLogs(t *testing.T) {
	defer func() {
		os.RemoveAll("logs")
		os.RemoveAll("logs-0.db")
		os.RemoveAll("logs-0.db-shm")
		os.RemoveAll("logs-0.db-wal")
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()

	InitMySql()
	logs := SelectLogs(0, 100000000, nil)

	for _, log := range logs {
		fmt.Printf("%d %s %s\n", log.BlockNumber, log.BlockHash.String(), log.TxHash.Hex())
	}
}
