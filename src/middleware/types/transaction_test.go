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
	tx := Transaction{Source: "0x888d72313eb41cb1c703d4cf4f1971a8adaf7299", Target: "0x888d72313eb41cb1c703d4cf4f1971a8adaf7299", Type: TransactionTypeMintNFT, Time: "1556076659050692000", SocketRequestId: "12140"}

	mintNFTInfo := make(map[string]string)
	mintNFTInfo["setId"] = "5bdb0c27-3162-4abd-abf2-126998735035"
	mintNFTInfo["id"] = "123459"
	mintNFTInfo["data"] = "5.99"
	mintNFTInfo["createTime"] = "1569736452603"
	mintNFTInfo["target"] = "0x888d72313eb41cb1c703d4cf4f1971a8adaf7299"

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

