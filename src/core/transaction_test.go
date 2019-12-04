package core

import (
	"testing"
	"fmt"
	"x/src/middleware/types"
	"encoding/hex"
	"x/src/common"
	"x/src/statemachine"
	"encoding/json"
)

func TestTxHash(t *testing.T) {
	tx := types.Transaction{}
	tx.Source = "aaa"
	tx.Target = "bbb"
	tx.Type = 0
	tx.Time = "1572940024276"
	tx.Data = "dafadfa"
	tx.Nonce = 1234

	hash := tx.GenHash()
	fmt.Printf("Tx Hash:%x\n", hash)
	fmt.Printf("Tx Hash:%s\n", hash.String())
}

func TestVerifySign(t *testing.T) {
	const PREFIX = "0x"
	signStr := "0xc93f6f6400587c1123b79823ca7b29adbc7d8c95746be0dab311166875029af35c9c35747eaf6090d5d19081eead2167300a719bdeec1e76014fb64b7d92227b00"
	hashStr := "c0b29b883cd39d5f261024081d1e0d140df1c3394a1980054eb2a75634d21e8a"

	if len(signStr) < len(PREFIX) || signStr[:len(PREFIX)] != PREFIX {
		return
	}
	signBytes, _ := hex.DecodeString(signStr[len(PREFIX):])
	fmt.Printf("Sign bytes:%v\n", signBytes)
	sign := common.BytesToSign(signBytes)

	hashByte := common.HexToHash(hashStr).Bytes()
	fmt.Printf("Hash bytes:%v\n", hashByte)

	pk, err := sign.RecoverPubkey(hashByte)
	if err != nil {
		fmt.Printf("Sign revover pubkey error:%s\n", err.Error())
		return
	}
	fmt.Printf("pk byte:%v\n", pk.ToBytes())
	fmt.Printf("pk:%s\n", pk.GetHexString())
	if !pk.Verify(hashByte, sign) {
		fmt.Printf("verify sign fail\n")
	}
	address := pk.GetAddress()
	fmt.Printf("Address:%v\n", address.Bytes())

	addressStr := address.GetHexString()
	fmt.Printf("Address str:%s\n", addressStr)
}

func TestSign(t *testing.T) {
	tx := types.Transaction{}
	tx.Source = "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"
	tx.Target = ""
	tx.Type = 110
	tx.Time = "1572950991799"
	tx.Data = "{\"symbol\":\"xxx\",\"createTime\":\"1572943640303\",\"name\":\"TestFTName\",\"maxSupply\":\"0\"}"
	tx.Nonce = 0
	hashStr := tx.GenHash()
	fmt.Printf("Tx Hash:%s\n", hashStr.String())

	hashByte := common.HexToHash(hashStr.String()).Bytes()
	fmt.Printf("Hash bytes:%v\n", hashByte)

	skStr := "0x05aa662f06e9a60c1d0d9304e5f8999be12bc4b66277416cf77601dcdd51a071"
	sk := common.HexStringToSecKey(skStr)
	sign := sk.Sign(hashByte)

	fmt.Printf("sign bytes:%v\n", sign.Bytes())
	fmt.Printf("sign:%s\n", sign.GetHexString())
}


func TestStateMachineTx(t *testing.T) {
	containerConfig := statemachine.ContainerConfig{Priority: 0, Game: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Name: "genesis_test",
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