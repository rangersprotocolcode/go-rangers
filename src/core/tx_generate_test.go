package core

import (
	"testing"
	"fmt"
	"encoding/json"
	"x/src/statemachine"
	"x/src/middleware/types"
)

func TestQueryBNTBalanceTx(t *testing.T) {
	tx := types.Transaction{Source: "0x6ed3a2ea39e1774096de4d920b4fb5b32d37fa98", Target: "0x6ed3a2ea39e1774096de4d920b4fb5b32d37fa98", Type: types.TransactionTypeGetCoin, Time: "1556076659050692000", SocketRequestId: "12140"}
	tx.Data = string("ETH.ETH")
	fmt.Printf("data:\n%s\n", tx.Data)

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestStateMachineTx(t *testing.T) {
	containerConfig := statemachine.ContainerConfig{Priority: 0, Game: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54",
		Image: "littlebear234/genesis_image:latest", Detached: true, Hostname: "genesis_host_name"}

	port := statemachine.Port{Host:0,Target:0}
	ports := statemachine.Ports{port}
	containerConfig.Ports = ports

	containerConfig.DownloadUrl = "littlebear234/genesis_image:latest"
	containerConfig.DownloadProtocol = "pull"

	tx := types.Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "", Type: types.TransactionTypeAddStateMachine,Time:"12121"}
	tx.Data = containerConfig.TOJSONString()

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestMintNFTTx(t *testing.T) {
	tx := types.Transaction{Source: "0x1ee7c120be9587415d235be9fe7032c17a610900", Target: "0x1ee7c120be9587415d235be9fe7032c17a610900", Type: types.TransactionTypeMintNFT, Time: "1556076659050692000", SocketRequestId: "12140"}

	mintNFTInfo := make(map[string]string)
	mintNFTInfo["setId"] = "bcb2d247-6586-4367-8e05-5fa3b2d88958"
	mintNFTInfo["id"] = "123457"
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


