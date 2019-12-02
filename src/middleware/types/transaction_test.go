package types

import (
	"testing"
	"fmt"
	"encoding/json"
)

func TestQueryBNTBalanceTx(t *testing.T) {
	tx := Transaction{Source: "0x6ed3a2ea39e1774096de4d920b4fb5b32d37fa98", Target: "0x6ed3a2ea39e1774096de4d920b4fb5b32d37fa98", Type: TransactionTypeGetCoin, Time: "1556076659050692000", SocketRequestId: "12140"}
	tx.Data = string("ETH.ETH")
	fmt.Printf("data:\n%s\n", tx.Data)

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestMintNFTTx(t *testing.T) {
	tx := Transaction{Source: "0x38eb86eefe56ea3de939d104361ba3699a7bbf0d", Target: "0x38eb86eefe56ea3de939d104361ba3699a7bbf0d", Type: TransactionTypeMintNFT, Time: "1556076659050692000", SocketRequestId: "12140"}

	mintNFTInfo := make(map[string]string)
	mintNFTInfo["setId"] = "f7a026d8-7eb4-4504-8cb2-6f3968b6627e"
	mintNFTInfo["id"] = "ccc"
	mintNFTInfo["data"] = "5.99"
	mintNFTInfo["createTime"] = "1569736452603"
	mintNFTInfo["target"] = "0x38eb86eefe56ea3de939d104361ba3699a7bbf0d"

	b, _ := json.Marshal(mintNFTInfo)
	tx.Data = string(b)
	fmt.Printf("data:\n%s\n", tx.Data)

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestOutputMessage(t *testing.T) {
	output := OutputMessage{}
	fmt.Printf("%v\n", output)
}

