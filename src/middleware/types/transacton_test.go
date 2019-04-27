package types

import (
	"testing"
	"fmt"
	"encoding/json"
)

func TestAssetOnChainTransactionHash(t *testing.T) {
	a := []string{"ss", "dd"}
	b, err := json.Marshal(a)
	if err != nil {
		fmt.Printf("Json marshal []string err:%s", err.Error())
		return
	}
	str:= string(b)

	txJson := TxJson{Source: "aaa", Target: "111", Type: 202, Data: str, Nonce: 2, Time: "1556076659050692000"}
	j, _ := json.Marshal(txJson)
	fmt.Printf("json:%s\n", string(j))

	tx := txJson.ToTransaction()
	fmt.Printf("TX:%v\n", tx)

	hash:= tx.GenHash().String()
	fmt.Printf("Hash:%s\n", hash)

}