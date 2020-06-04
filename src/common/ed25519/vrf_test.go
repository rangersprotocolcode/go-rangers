package ed25519

import (
	"bytes"
	mathRand "crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"
)

//---------------------------------------Function Test-----------------------------------------------------------------
func TestKeyLength(test *testing.T) {
	privateKey, publicKey := genRandomKey(nil)
	fmt.Printf("privateKey:%v,len:%d\n", privateKey, len(privateKey))
	fmt.Printf("pubkey :%v,len:%d\n", publicKey, len(publicKey))
	assert.Equal(test, len(privateKey), 64)
	assert.Equal(test, len(publicKey), 32)
}

func TestGenProveAndVerifyOnce(test *testing.T) {
	runGenProveAndVerifyOnce(test, nil)
}

func TestSignAndVerifyRepeatedly(test *testing.T) {
	var testCount = 1000
	for i := 0; i < testCount; i++ {
		if i%2 == 0 {
			runGenProveAndVerifyOnce(test, mathRand.Reader)
		} else {
			runGenProveAndVerifyOnce(test, nil)
		}
	}
}

func genRandomKey(random io.Reader) (privateKey PrivateKey, publicKey PublicKey) {
	publicKey, privateKey, err := GenerateKey(random)
	if err != nil {
		panic("Ed25519 generate key error!" + err.Error())
	}
	return
}

func genRandomMessage(length uint64) []byte {
	msg := make([]byte, length)

	var i uint64 = 0
	for ; i < length; i++ {
		msg[i] = byte(rand.Uint64() % 256)
	}
	return msg
}

func runGenProveAndVerifyOnce(test *testing.T, random io.Reader) {
	privateKey, publicKey := genRandomKey(random)
	msg := genRandomMessage(32)

	proof, err := ECVRFProve(privateKey, msg)
	if err != nil {
		panic("ECVRFProve error!" + err.Error())
	}
	//fmt.Printf("proof:%v\n", proof)
	//fmt.Printf("proof size:%v\n", len(proof))

	verifyResult, err := ECVRFVerify(publicKey, proof, msg)
	if err != nil {
		panic("ECVRFVerify error!" + err.Error())
	}
	assert.Equal(test, verifyResult, true)
}

//---------------------------------------Benchmark Test-----------------------------------------------------------------
var testCount = 1000
var privateKeyList = make([]PrivateKey, testCount)
var publicKeyList = make([]PublicKey, testCount)
var messageList = make([][]byte, testCount)
var proofList = make([]VRFProve, testCount)

func BenchmarkGenProve(b *testing.B) {
	prepareData()
	b.ResetTimer()

	begin := time.Now()
	for i := 0; i < testCount; i++ {
		privateKey := privateKeyList[i]
		message := messageList[i]

		proof, err := ECVRFProve(privateKey, message)
		if err != nil {
			panic("ECVRFProve error!" + err.Error())
		}
		proofList[i] = proof
	}
	fmt.Printf("cost:%v\n", time.Since(begin).Seconds())
}

func BenchmarkVerifyProof(b *testing.B) {
	prepareData()
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		proof := proofList[i]
		message := messageList[i]
		publicKey := publicKeyList[i]

		verifyResult, err := ECVRFVerify(publicKey, proof, message)
		if err != nil {
			panic("ECVRFVerify error!" + err.Error())
		}
		assert.Equal(b, verifyResult, true)
	}
}

func BenchmarkSignAndVerifySign(b *testing.B) {
	prepareData()
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		privateKey := privateKeyList[i]
		publicKey := publicKeyList[i]
		message := messageList[i]

		proof, err := ECVRFProve(privateKey, message)
		if err != nil {
			panic("ECVRFProve error!" + err.Error())
		}

		verifyResult, err := ECVRFVerify(publicKey, proof, message)
		if err != nil {
			panic("ECVRFVerify error!" + err.Error())
		}
		assert.Equal(b, verifyResult, true)
	}
}

func prepareData() {
	for i := 0; i < testCount; i++ {
		var privateKey PrivateKey
		var publicKey PublicKey
		if i%2 == 0 {
			privateKey, publicKey = genRandomKey(mathRand.Reader)
		} else {
			privateKey, publicKey = genRandomKey(nil)
		}
		privateKeyList[i] = privateKey
		publicKeyList[i] = publicKey
		messageList[i] = genRandomMessage(32)

		proof, err := ECVRFProve(privateKey, messageList[i])
		if err != nil {
			panic("ECVRFProve error!" + err.Error())
		}
		proofList[i] = proof
	}
}

//---------------------------------------Standard Data Test---------------------------------------------------------------
func TestVRFStandard(test *testing.T) {

	fileName := "vrf_standard_data.txt"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read vrf_standard_data info  file error:" + err.Error())
	}
	records := strings.Split(string(bytes), "\n")

	for _, record := range records {
		elements := strings.Split(record, "|")
		fmt.Println(elements[0])
		fmt.Println(elements[1])
		fmt.Println(elements[2])
		fmt.Println(elements[3])
		fmt.Println("")

		var privateKey PrivateKey
		var publicKey PublicKey

		privateKey, _ = hex.DecodeString(elements[0])
		publicKey, _ = hex.DecodeString(elements[1])
		messageByte, _ := hex.DecodeString(elements[2])
		exceptedProve := elements[3]
		prove, err := ECVRFProve(privateKey, messageByte)
		if err != nil {
			panic(err)
		}
		proveStr := hex.EncodeToString(prove)
		assert.Equal(test, exceptedProve, proveStr)

		var pi VRFProve
		pi, _ = hex.DecodeString(proveStr)

		verifyResult, err := ECVRFVerify(publicKey, pi, messageByte)
		if err != nil {
			panic(err)
		}
		assert.Equal(test, verifyResult, true)
	}
}

//---------------------------------------Comparison Test---------------------------------------------------------------
const prefix = "0x"

func TestGenComparisonData(test *testing.T) {
	fileName := "vrf_comparisonData_go.txt"

	var buffer bytes.Buffer
	var privateKey PrivateKey
	var publicKey PublicKey
	privateKey, publicKey = genRandomKey(nil)
	messageByte := genRandomMessage(32)

	for i := 0; i < testCount; i++ {
		if i%2 == 0 {
			privateKey, publicKey = genRandomKey(mathRand.Reader)
		} else {
			privateKey, publicKey = genRandomKey(nil)
		}

		buffer.WriteString("privateKey:")
		privateKeyHex := prefix + hex.EncodeToString(privateKey[:])
		buffer.WriteString(privateKeyHex)

		buffer.WriteString("|publicKey:")
		buffer.WriteString(prefix + hex.EncodeToString(publicKey[:]))
		buffer.WriteString("|message:")

		message := prefix + hex.EncodeToString(messageByte)
		buffer.WriteString(message)

		buffer.WriteString("|proof:")
		proofByte, err := ECVRFProve(privateKey, messageByte)
		if err != nil {
			panic("ECVRFProve error!" + err.Error())
		}
		buffer.WriteString(prefix + hex.EncodeToString(proofByte))
		buffer.WriteString("\n")
	}
	ioutil.WriteFile(fileName, buffer.Bytes(), 0644)
}

func TestValidateComparisonData(test *testing.T) {
	fileName := "vrf_comparisonData_java.txt"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read vrf_comparisonData_java info  file error:" + err.Error())
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
		proof := strings.Replace(elements[3], "proof:", "", 1)

		//fmt.Println(privateKey)
		//fmt.Println(publicKey)
		//fmt.Println(message)
		//fmt.Println(sign)
		validateFunction(privateKey, publicKey, message, proof, test)
	}
}

func validateFunction(privateKeyStr, publicKeyStr, message, proofStr string, test *testing.T) {

	var privateKey PrivateKey
	privateKey, _ = hex.DecodeString(privateKeyStr[len(prefix):])

	var publicKey PublicKey
	publicKey, _ = hex.DecodeString(publicKeyStr[len(prefix):])

	messageByte, _ := hex.DecodeString(message[len(prefix):])

	proofByte, _ := hex.DecodeString(proofStr[len(prefix):])

	//validate gen prove
	expectedProof, err := ECVRFProve(privateKey, messageByte)
	if err != nil {
		panic("ECVRFProve error!" + err.Error())
	}
	expectedProveStr := prefix + hex.EncodeToString(expectedProof)

	assert.Equal(test, expectedProveStr, proofStr)

	//validate prove
	verifyResult, err := ECVRFVerify(publicKey, proofByte, messageByte)
	if err != nil {
		panic("ECVRFVerify error!" + err.Error())
	}
	assert.Equal(test, verifyResult, true)
}
