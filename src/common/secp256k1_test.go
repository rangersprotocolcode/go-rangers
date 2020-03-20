package common

import (
	"fmt"
	"testing"
	"math/rand"
	"github.com/stretchr/testify/assert"
	"x/src/common/secp256k1"
	"bytes"
	"encoding/hex"
	"io/ioutil"
)

func TestKeyLength(test *testing.T) {
	key := genRandomKey()
	privateKey := key.PrivKey.D.Bytes()
	fmt.Printf("privateKey:%v,len:%d\n", privateKey, len(privateKey))

	pubKeyX := key.PrivKey.X.Bytes()
	pubKeyY := key.PrivKey.Y.Bytes()
	fmt.Printf("pubkey x:%v,lenX:%d\n", pubKeyX, len(pubKeyX))
	fmt.Printf("pubkey y:%v,lenY:%d\n", pubKeyY, len(pubKeyY))
	//assert.Equal(test, len(privateKey), 32)
	assert.Equal(test, len(key.GetPubKey().ToBytes()), 65)
}

func TestSignAndVerifyOnce(test *testing.T) {
	runSigAndVerifyOnce(test)
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
		assert.Equal(test, verifyResult, true)
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

	sign := key.Sign(msg)
	assert.Equal(test, len(sign.Bytes()), 65)
	//fmt.Printf("sign:%v,length:%d\n", sign.Bytes(), len(sign.Bytes()))

	recoveredPubKey, err := secp256k1.RecoverPubkey(msg, sign.Bytes())
	if err != nil {
		test.Error("recover pubkey error:", err)
	}
	//fmt.Printf("Recovered pubkey:%v,length:%d\n", recoveredPubKey, len(recoveredPubKey))
	assert.Equal(test, recoveredPubKey, key.GetPubKey().ToBytes())

	verifyResult := secp256k1.VerifySignature(recoveredPubKey, msg, sign.Bytes())
	assert.Equal(test, verifyResult, true)
}

var testCount = 1000
var privateList = make([]PrivateKey, testCount)
var messageList = make([][]byte, testCount)
var signList = make([]Sign, testCount)

func BenchmarkPrivateSign(b *testing.B) {
	prepareData()
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		privateKey := privateList[i]
		message := messageList[i]
		signList[i] = privateKey.Sign(message)
	}
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
		assert.Equal(b, recoveredPubKey, privateKey.GetPubKey().ToBytes())
	}
}

func BenchmarkVerifySign(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		sign := signList[i]
		message := messageList[i]
		privateKey := privateList[i]

		verifyResult := privateKey.GetPubKey().Verify(message, &sign)
		assert.Equal(b, verifyResult, true)
	}
}

func BenchmarkSignAndVerifySign(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		privateKey := privateList[i]
		message := messageList[i]

		sign := privateKey.Sign(message)
		verifyResult := privateKey.GetPubKey().Verify(message, &sign)
		assert.Equal(b, verifyResult, true)
	}
}

func prepareData() {
	for i := 0; i < testCount; i++ {
		privateList[i] = genRandomKey()
		messageList[i] = genRandomMessage(32)
	}
}

func TestGenComparisonData(test *testing.T) {
	fileName := "secp256_comparisonData_go.txt"

	var buffer bytes.Buffer
	for i := 0; i < testCount; i++ {
		key := genRandomKey()
		buffer.WriteString("privateKey:")
		privateKeyHex := hex.EncodeToString(key.PrivKey.D.Bytes())
		buffer.WriteString(privateKeyHex)
		//buffer.WriteString(key.GetHexString())

		buffer.WriteString("|publicKey:")
		buffer.WriteString(key.GetPubKey().GetHexString())
		buffer.WriteString("|message:")

		messageByte := genRandomMessage(32)
		message := hex.EncodeToString(messageByte)
		buffer.WriteString(message)

		buffer.WriteString("|sign:")
		sign := key.Sign(messageByte)
		buffer.WriteString(sign.GetHexString())
		buffer.WriteString("\n")

		//fmt.Printf("private binary:")
		//for _, b := range key.PrivKey.D.Bytes() {
		//	fmt.Printf("%b,", b)
		//}
		//fmt.Printf("\n")
		//fmt.Printf("privatekey hex:%s\n\n", privateKeyHex)

		//fmt.Printf("message binary:")
		//for _, b := range messageByte {
		//	fmt.Printf("%b,", b)
		//}
		//fmt.Printf("\n")
		//fmt.Printf("message hex:%s\n\n", message)
		//
		//fmt.Printf("sign binary:")
		//for _, b := range sign.Bytes() {
		//	fmt.Printf("%b,", b)
		//}
		//fmt.Printf("\n")
		//fmt.Printf("sign hex:%s\n\n", sign.GetHexString())
	}
	ioutil.WriteFile(fileName, buffer.Bytes(), 0644)
}
