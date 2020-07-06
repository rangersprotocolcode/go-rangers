// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"bytes"
	"com.tuntun.rocket/node/src/common/secp256k1"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"
)

//---------------------------------------Function Test-----------------------------------------------------------------
func TestKeyLength(test *testing.T) {
	key := genRandomKey()
	privateKey := key.PrivKey.D.Bytes()
	fmt.Printf("privateKey:%v,len:%d\n", privateKey, len(privateKey))

	pubKeyX := key.PrivKey.X.Bytes()
	pubKeyY := key.PrivKey.Y.Bytes()
	fmt.Printf("pubkey x:%v,lenX:%d\n", pubKeyX, len(pubKeyX))
	fmt.Printf("pubkey y:%v,lenY:%d\n", pubKeyY, len(pubKeyY))
	//assert.Equal(test, len(privateKey), 32)
	assert.Equal(test, 65, len(key.GetPubKey().ToBytes()))
}

func TestSignAndVerifyOnce(test *testing.T) {
	runSigAndVerifyOnce(test)
}

func TestSignAndVerifyOnceByFixKey(test *testing.T) {
	sk := HexStringToSecKey("0x04008627e8037ef68a2722091ca8507e91e65ce93ab621a59ca647d1361eb8337129713294ffdbef401d609cac491c25094b34ff82190fd54e721628479095d74d4333024273384e11372c1d2ab3af8bd5ae7221ead96bfc3ee36b73c6e8e594d1")

	msg := Hex2Bytes("7fd31ab615e73fc5d091238f00ad3390c651731a0dfdb8867f98b930d21af56c")
	fmt.Printf("privatekey:%v,pubkey:%v\n", sk.GetHexString(), sk.GetPubKey().GetHexString())

	sign := sk.Sign(msg)
	assert.Equal(test, 65, len(sign.Bytes()))
	fmt.Printf("sign:%v,length:%d\n", sign.GetHexString(), len(sign.Bytes()))

	recoveredPubKey, err := secp256k1.RecoverPubkey(msg, sign.Bytes())
	if err != nil {
		test.Error("recover pubkey error:", err)
	}
	fmt.Printf("Recovered pubkey:%v,length:%d\n", Bytes2Hex(recoveredPubKey), len(recoveredPubKey))
	assert.Equal(test, sk.GetPubKey().ToBytes(), recoveredPubKey)

	verifyResult := secp256k1.VerifySignature(recoveredPubKey, msg, sign.Bytes())
	assert.Equal(test, true, verifyResult)

	pk := BytesToPublicKey(recoveredPubKey)
	fmt.Printf("Recovered pubkey:%v\n", pk.GetHexString())
	revoveredId := pk.GetAddress().GetHexString()
	fmt.Printf("Recovered id:%v\n", revoveredId)
}

func TestSignAndVerifyRepeatedly(test *testing.T) {
	var testCount = 1000
	for i := 0; i < testCount; i++ {
		runSigAndVerifyOnce(test)
	}
}

func TestSignAndVerifyByFixedKey(test *testing.T) {
	privateKey := genRandomKey()

	var testCount = 1000
	for i := 0; i < testCount; i++ {
		msg := genRandomMessage(32)

		sign := privateKey.Sign(msg)
		fmt.Printf("sign:%v,length:%d\n", sign.Bytes(), len(sign.Bytes()))

		recoveredPubKey, err := secp256k1.RecoverPubkey(msg, sign.Bytes())
		if err != nil {
			test.Error("recover pubkey error:", err)
		}
		fmt.Printf("Recovered pubkey:%v,length:%d\n", recoveredPubKey, len(recoveredPubKey))

		verifyResult := secp256k1.VerifySignature(recoveredPubKey, msg, sign.Bytes())
		assert.Equal(test, true, verifyResult)
	}
}

func genRandomKey() PrivateKey {
	key := GenerateKey("")

	if len(key.ToBytes()) != 32 {
		privateKey := make([]byte, 32)
		sk := key.PrivKey.D.Bytes()
		copy(privateKey[32-len(sk):32], sk)

		key.PrivKey.D.SetBytes(privateKey)
	}
	return key
}

func genRandomMessage(length uint64) []byte {
	msg := make([]byte, length)

	var i uint64 = 0
	for ; i < length; i++ {
		msg[i] = byte(rand.Uint64() % 256)
	}
	//fmt.Printf("msg:%v\n", msg)
	return msg
}

func runSigAndVerifyOnce(test *testing.T) {
	key := genRandomKey()
	msg := genRandomMessage(32)
	fmt.Printf("privatekey:%v,length:%d\n", key.ToBytes(), len(key.ToBytes()))

	sign := key.Sign(msg)
	assert.Equal(test, 65, len(sign.Bytes()))
	//fmt.Printf("sign:%v,length:%d\n", sign.Bytes(), len(sign.Bytes()))

	recoveredPubKey, err := secp256k1.RecoverPubkey(msg, sign.Bytes())
	if err != nil {
		test.Error("recover pubkey error:", err)
	}
	//fmt.Printf("Recovered pubkey:%v,length:%d\n", recoveredPubKey, len(recoveredPubKey))
	assert.Equal(test, key.GetPubKey().ToBytes(), recoveredPubKey)

	verifyResult := secp256k1.VerifySignature(recoveredPubKey, msg, sign.Bytes())
	assert.Equal(test, true, verifyResult)
}

//---------------------------------------Benchmark Test-----------------------------------------------------------------
var testCount = 1000
var privateList = make([]PrivateKey, testCount)
var messageList = make([][]byte, testCount)
var signList = make([]Sign, testCount)

func BenchmarkPrivateSign(b *testing.B) {
	prepareData()
	b.ResetTimer()

	begin := time.Now()
	for i := 0; i < testCount; i++ {
		privateKey := privateList[i]
		message := messageList[i]
		signList[i] = privateKey.Sign(message)
	}
	fmt.Printf("cost:%v\n", time.Since(begin).Seconds())
}

func BenchmarkRecoverPubKey(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		sign := signList[i]
		message := messageList[i]
		recoveredPubKey, err := sign.RecoverPubkey(message)
		if err != nil {
			b.Error("recover pubKey error:" + err.Error())
			return
		}

		privateKey := privateList[i]
		assert.Equal(b, privateKey.GetPubKey().ToBytes(), recoveredPubKey)
	}
}

func BenchmarkVerifySign(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		sign := signList[i]
		message := messageList[i]
		privateKey := privateList[i]

		verifyResult := privateKey.GetPubKey().Verify(message, &sign)
		assert.Equal(b, true, verifyResult)
	}
}

func BenchmarkSignAndVerifySign(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		privateKey := privateList[i]
		message := messageList[i]

		sign := privateKey.Sign(message)
		verifyResult := privateKey.GetPubKey().Verify(message, &sign)
		assert.Equal(b, true, verifyResult)
	}
}

func prepareData() {
	for i := 0; i < testCount; i++ {
		privateList[i] = genRandomKey()
		messageList[i] = genRandomMessage(32)
	}
}

//---------------------------------------Comparison Test---------------------------------------------------------------
func TestGenComparisonData(test *testing.T) {
	fileName := "secp256_comparisonData_go.txt"

	var buffer bytes.Buffer
	for i := 0; i < testCount; i++ {
		key := genRandomKey()
		buffer.WriteString("privateKey:")
		privateKeyHex := hex.EncodeToString(key.PrivKey.D.Bytes())
		buffer.WriteString(privateKeyHex)

		buffer.WriteString("|publicKey:")
		buffer.WriteString(key.GetPubKey().GetHexString())
		buffer.WriteString("|message:")

		messageByte := genRandomMessage(32)
		message := hex.EncodeToString(messageByte)
		buffer.WriteString(message)

		buffer.WriteString("|sign:")
		sign := key.Sign(messageByte)
		buffer.WriteString(sign.GetHexString())

		buffer.WriteString("|id:")
		id := key.GetPubKey().GetID()
		buffer.WriteString(ToHex(id[:]))

		buffer.WriteString("\n")
	}
	ioutil.WriteFile(fileName, buffer.Bytes(), 0644)
}

func TestValidateComparisonData(test *testing.T) {
	fileName := "secp256_comparisonData_java.txt"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read secp256_comparisonData_java info  file error:" + err.Error())
	}
	records := strings.Split(string(bytes), "\n")

	for _, record := range records {
		fmt.Println(record)
		elements := strings.Split(record, "|")
		//fmt.Println(elements[0])
		//fmt.Println(elements[1])
		//fmt.Println(elements[2])
		//fmt.Println(elements[3])

		privateKey := strings.Replace(elements[0], "privateKey:", "", 1)
		publicKey := strings.Replace(elements[1], "publicKey:", "", 1)
		message := strings.Replace(elements[2], "message:", "", 1)
		sign := strings.Replace(elements[3], "sign:", "", 1)
		id := strings.Replace(elements[4], "id:", "", 1)

		//fmt.Println(privateKey)
		//fmt.Println(publicKey)
		//fmt.Println(message)
		//fmt.Println(sign)

		validateFunction(privateKey, publicKey, message, sign, id, test)
	}
}

func validateFunction(privateKeyStr, publicKeyStr, message, signStr, idStr string, test *testing.T) {

	privateKeyBuf, _ := hex.DecodeString(privateKeyStr[len(PREFIX):])
	fmt.Printf("privateKeyBuf len:%d\n", len(privateKeyBuf))
	var privateKey PrivateKey
	privateKey.PrivKey.D = new(big.Int).SetBytes(privateKeyBuf)

	publicKeyBuf, _ := hex.DecodeString(publicKeyStr[len(PREFIX):])
	privateKey.PrivKey.PublicKey = BytesToPublicKey(publicKeyBuf).PubKey

	//get public key by private key
	publicKey := privateKey.GetPubKey()
	assert.Equal(test, publicKeyStr, publicKey.GetHexString())

	//Recover public key from sign
	sign := HexStringToSign(signStr)
	messageBytes, _ := hex.DecodeString(message[len(PREFIX):])
	recoveredPublicKey, err := sign.RecoverPubkey(messageBytes)
	if err != nil {
		panic("Recover publicKey from sign error:" + err.Error())
	}
	assert.Equal(test, publicKeyStr, recoveredPublicKey.GetHexString())

	//Sign
	expectedSign := privateKey.Sign(messageBytes)
	assert.Equal(test, signStr, expectedSign.GetHexString())

	//verify sign
	verifyResult := publicKey.Verify(messageBytes, sign)
	assert.Equal(test, true, verifyResult)

	//verify id
	id := publicKey.GetID()
	assert.Equal(test, idStr, ToHex(id[:]))
}
