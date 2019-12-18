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
	tx := Transaction{Source: "0x1ee7c120be9587415d235be9fe7032c17a610900", Target: "0x1ee7c120be9587415d235be9fe7032c17a610900", Type: TransactionTypeMintNFT, Time: "1556076659050692000", SocketRequestId: "12140"}

	mintNFTInfo := make(map[string]string)
	mintNFTInfo["setId"] = "689dab31-548e-4b88-9a02-a3c84b4004dc"
	mintNFTInfo["id"] = "aaa"
	mintNFTInfo["data"] = "5.99"
	mintNFTInfo["createTime"] = "1569736452603"
	mintNFTInfo["target"] = "0x1ee7c120be9587415d235be9fe7032c17a610900"

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

