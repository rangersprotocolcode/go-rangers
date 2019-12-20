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

	port := statemachine.Port{Host: 0, Target: 0}
	ports := statemachine.Ports{port}
	containerConfig.Ports = ports

	containerConfig.DownloadUrl = "littlebear234/genesis_image:latest"
	containerConfig.DownloadProtocol = "pull"

	tx := types.Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "", Type: types.TransactionTypeAddStateMachine, Time: "12121"}
	tx.Data = containerConfig.TOJSONString()

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestMintNFTTx(t *testing.T) {
	tx := types.Transaction{Source: "0xe4cb43dc1659b7978ce2e5f71b0d1163fc96936c", Target: "0xe4cb43dc1659b7978ce2e5f71b0d1163fc96936c", Type: types.TransactionTypeMintNFT, Time: "1556076659050692000", SocketRequestId: "12140"}

	mintNFTInfo := make(map[string]string)
	mintNFTInfo["setId"] = "c5313630-5d5b-43e4-aea7-fb11b8163803"
	mintNFTInfo["id"] = "123456"
	mintNFTInfo["data"] = "5.99"
	mintNFTInfo["createTime"] = "1569736452603"
	mintNFTInfo["target"] = "0xe4cb43dc1659b7978ce2e5f71b0d1163fc96936c"

	b, _ := json.Marshal(mintNFTInfo)
	tx.Data = string(b)
	fmt.Printf("data:\n%s\n", tx.Data)

	tx.Hash = tx.GenHash()

	//skStr := "0x05aa662f06e9a60c1d0d9304e5f8999be12bc4b66277416cf77601dcdd51a071"
	////skStr := "0x0bdfc3725a93be336de9cd2e97e508b41d67d7fefe368fb7fd7f02b6236c7cc0"
	//sk := common.HexStringToSecKey(skStr)
	//sign := sk.Sign(tx.Hash.Bytes())
	//tx.Sign = &sign

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestTxJson(t *testing.T) {
	txJson := types.TxJson{}
	fmt.Printf("%v",txJson.Nonce)

	txJson.Nonce = 0
	txJson.Source = "111"
	byte, _ := json.Marshal(txJson)
	fmt.Printf("%s",string(byte))

}
