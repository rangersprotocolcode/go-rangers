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

	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 202, Data: str, Nonce: 2, Time: "1556076659050692000"}
	j, _ := json.Marshal(txJson)
	fmt.Printf("json:%s\n", string(j))

	tx := txJson.ToTransaction()
	fmt.Printf("TX:%v\n", tx)

	hash := tx.GenHash().String()
	fmt.Printf("Hash:%s\n", hash)

}

func TestUpgradeDragonTransactionHash(t *testing.T) {
	txJson := TxJson{Target: "tuntunbiu", Type: 8, Data: "u", Nonce: 2,}
	j, _ := json.Marshal(txJson)
	fmt.Printf("json:%s\n", string(j))

	tx := txJson.ToTransaction()
	fmt.Printf("TX:%v\n", tx)

	hash := tx.GenHash().String()
	fmt.Printf("Hash:%s\n", hash)

}

func TestWithdrawTransactionHash(t *testing.T) {
	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 201, Data: "1.2", Nonce: 1, Time: "1556076659050692000"}
	j, _ := json.Marshal(txJson)
	fmt.Printf("json:%s\n", string(j))

	tx := txJson.ToTransaction()
	fmt.Printf("TX:%v\n", tx)

	tx.Hash = tx.GenHash()
	fmt.Printf("Hash:%s\n", tx.Hash.String())

}
