package core

import (
	"testing"
	"fmt"
	"x/src/middleware/types"
	"encoding/hex"
	"x/src/common"
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
	signStr := "0xb8a458d065d386ef31d0cddefeb4eee5fcae3ba6d1f9638220a96b470bd3ce1a6a37727e61d72da332e6b7b5a5c76481dae8b60b71cc97861f8a35a1a2b899a700"
	hashStr := "0x0f52724aac3746d6b081e5bf6ba23b7c21c2810e6087366486652a10052dc9ee"

	if len(signStr) < len(PREFIX) || signStr[:len(PREFIX)] != PREFIX {
		return
	}
	signBytes, _ := hex.DecodeString(signStr[len(PREFIX):])
	fmt.Printf("Sign bytes:%v\n\n", signBytes)
	sign := common.BytesToSign(signBytes)

	hashByte := common.HexToHash(hashStr).Bytes()
	fmt.Printf("Hash bytes:%v\n\n", hashByte)

	pk, err := sign.RecoverPubkey(hashByte)
	if err != nil {
		fmt.Printf("Sign revover pubkey error:%s\n", err.Error())
		return
	}
	fmt.Printf("pk byte:%v\n\n", pk.ToBytes())
	fmt.Printf("pk:%s\n\n", pk.GetHexString())
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
