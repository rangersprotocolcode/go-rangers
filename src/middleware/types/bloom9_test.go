package types

import (
	"math/big"
	"testing"
	"github.com/gin-gonic/gin/json"
	"fmt"
)

func TestBloom(t *testing.T) {
	positive := []string{
		"testtest",
		"test",
		"hallo",
		"other",
	}
	negative := []string{
		"tes",
		"lo",
	}

	var bloom Bloom
	for _, data := range positive {
		bloom.Add(new(big.Int).SetBytes([]byte(data)))
	}

	for _, data := range positive {
		if !bloom.TestBytes([]byte(data)) {
			t.Error("expected", data, "to test true")
		}
	}
	for _, data := range negative {
		if bloom.TestBytes([]byte(data)) {
			t.Error("did not expect", data, "to test true")
		}
	}
}

func TestWithdrawTransactionHash(t *testing.T) {
	txJson := TxJson{Source: "aaa", Target: "111", Type: 201, Data: "1.2", Nonce: 1, Time: "1556076659050692000"}
	j, _ := json.Marshal(txJson)
	fmt.Printf("json:%s\n", string(j))

	tx := txJson.ToTransaction()
	fmt.Printf("TX:%v\n", tx)

    tx.Hash = tx.GenHash()
	fmt.Printf("Hash:%s\n", tx.Hash.String())


}


