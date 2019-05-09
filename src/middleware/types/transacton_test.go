package types

import (
	"testing"
	"fmt"
	"encoding/json"
)

func TestAssetOnChainTransactionHash(t *testing.T) {
	a := []string{"drogon"}
	b, err := json.Marshal(a)
	if err != nil {
		fmt.Printf("Json marshal []string err:%s", err.Error())
		return
	}
	str := string(b)

	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 202, Data: str, Nonce: 3, Time: "1556076659050692000"}
	tx := txJson.ToTransaction()
	tx.Hash = tx.GenHash()

	j,_:=  json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))

}

func TestUpgradeDragonTransactionHash(t *testing.T) {
	txJson := TxJson{Target: "tuntunbiu", Type: 8, Data: "u", Nonce: 6,}
	tx := txJson.ToTransaction()
	tx.Hash = tx.GenHash()

	j,_:=  json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestWithdrawTransactionHash(t *testing.T) {
	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 201, Data: "1.35", Nonce: 1, Time: "1556076659050692000"}
	tx := txJson.ToTransaction()
	tx.Hash = tx.GenHash()

	j,_:=  json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}


func TestTransactionHash(t *testing.T) {
	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 11,  Time: "1556076659050692000"}
	tx := txJson.ToTransaction()
	tx.Hash = tx.GenHash()

	j,_:=  json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}