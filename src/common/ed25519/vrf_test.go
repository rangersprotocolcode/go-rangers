package ed25519

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"fmt"
	"io"
	mathRand "crypto/rand"
	"math/rand"
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"strings"
	"math/big"
	"crypto/sha256"
)

//---------------------------------------Function Test-----------------------------------------------------------------
func TestKeyLength(test *testing.T) {
	privateKey, publicKey := genRandomKey(nil)
	fmt.Printf("privateKey:%v,len:%d\n", privateKey, len(privateKey))
	fmt.Printf("pubkey :%v,len:%d\n", publicKey, len(publicKey))
	assert.Equal(test, len(privateKey), 32)
	assert.Equal(test, len(publicKey), 64)
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

	proof, err := ECVRFProve(publicKey, privateKey, msg)
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

	for i := 0; i < testCount; i++ {
		privateKey := privateKeyList[i]
		publicKey := publicKeyList[i]
		message := messageList[i]

		proof, err := ECVRFProve(publicKey, privateKey, message)
		if err != nil {
			panic("ECVRFProve error!" + err.Error())
		}
		proofList[i] = proof
	}
}

func BenchmarkVerifyProof(b *testing.B) {
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
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		privateKey := privateKeyList[i]
		publicKey := publicKeyList[i]
		message := messageList[i]

		proof, err := ECVRFProve(publicKey, privateKey, message)
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
	}
}

//---------------------------------------Comparison Test---------------------------------------------------------------
const prefix = "0x"

func TestGenComparisonData(test *testing.T) {
	fileName := "ed25519_comparisonData_go.txt"

	var buffer bytes.Buffer
	var privateKey PrivateKey
	var publicKey PublicKey
	privateKey, publicKey = genRandomKey(nil)
	messageByte := genRandomMessage(32)

	for i := 0; i < testCount; i++ {

		//if i%2 == 0 {
		//	privateKey, publicKey = genRandomKey(mathRand.Reader)
		//} else {
		//	privateKey, publicKey = genRandomKey(nil)
		//}

		buffer.WriteString("privateKey:")
		privateKeyHex := prefix + hex.EncodeToString(privateKey)
		buffer.WriteString(privateKeyHex)

		buffer.WriteString("|publicKey:")
		buffer.WriteString(prefix + hex.EncodeToString(publicKey))
		buffer.WriteString("|message:")

		//messageByte := genRandomMessage(32)
		message := prefix + hex.EncodeToString(messageByte)
		buffer.WriteString(message)

		buffer.WriteString("|proof:")
		proofByte, err := ECVRFProve(publicKey, privateKey, messageByte)
		if err != nil {
			panic("ECVRFProve error!" + err.Error())
		}
		buffer.WriteString(prefix + hex.EncodeToString(proofByte))
		buffer.WriteString("\n")
	}
	ioutil.WriteFile(fileName, buffer.Bytes(), 0644)
}

func TestValidateComparisonData(test *testing.T) {
	fileName := "ed25519_comparisonData_java.txt"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read ed25519_comparisonData_java info  file error:" + err.Error())
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
	expectedProof, err := ECVRFProve(publicKey, privateKey, messageByte)
	if err != nil {
		panic("ECVRFProve error!" + err.Error())
	}
	assert.Equal(test, expectedProof, proofByte)

	//validate prove
	verifyResult, err := ECVRFVerify(publicKey, proofByte, messageByte)
	if err != nil {
		panic("ECVRFVerify error!" + err.Error())
	}
	assert.Equal(test, verifyResult, true)
}

//---------------------------------------Debug Test---------------------------------------------------------------
func TestGenProveDebug(test *testing.T) {

	var buffer bytes.Buffer
	var privateKey PrivateKey
	var publicKey PublicKey

	privateKey, _ = hex.DecodeString("759b83f6b8d512beea30ad0f77234197b78994ad9eb39dfd908a6d382cc8e4ea584ca6fb8839a8e56ffcd43db259bdd75fbb926791381261bbf1c13e629162ea")
	publicKey, _ = hex.DecodeString("584ca6fb8839a8e56ffcd43db259bdd75fbb926791381261bbf1c13e629162ea")

	buffer.WriteString("privateKey:")
	buffer.WriteString(prefix + hex.EncodeToString(privateKey))
	buffer.WriteString("\nprivateKey binary:" + fmt.Sprintf("%b", privateKey))

	buffer.WriteString("\n\npublicKey:")
	buffer.WriteString(prefix + hex.EncodeToString(publicKey))
	buffer.WriteString("\npublicKey binary:" + fmt.Sprintf("%b", publicKey))

	buffer.WriteString("\n\nmessage:")
	messageByte, _ := hex.DecodeString("524f1d03d1d81e94a099042736d40bd9681b867321443ff58a4568e274dbd83b")
	buffer.WriteString(prefix + hex.EncodeToString(messageByte))
	buffer.WriteString("\nmessage binary:" + fmt.Sprintf("%b", messageByte))

	hash := sha256.New()
	hash.Write(messageByte)
	result := hash.Sum(nil)
	fmt.Printf("message hash:%b\n", result)

	kp, _ := hex.DecodeString("e34037f227063bbd763d869b35e01ac2cb9364bb71548bf3df99ffeedacd8984a97ecbb0493569cb7a3cdfbcffe36583a05d80193a68d38a584c1f691e3f89f1")
	ks, _ := hex.DecodeString("a97ecbb0493569cb7a3cdfbcffe36583a05d80193a68d38a584c1f691e3f89f1")
	buffer.WriteString("\n\nks:")
	buffer.WriteString(prefix + hex.EncodeToString(ks))
	buffer.WriteString("\nks binary:" + fmt.Sprintf("%b", ks))

	buffer.WriteString("\n\nkp:")
	buffer.WriteString(prefix + hex.EncodeToString(kp))
	buffer.WriteString("\nkp binary:" + fmt.Sprintf("%b", kp))

	buffer.WriteString("\n\nproof:")
	proofByte, err := ECVRFProveDebug(publicKey, privateKey, messageByte, kp, ks)
	if err != nil {
		panic("ECVRFProve error!" + err.Error())
	}
	buffer.WriteString(prefix + hex.EncodeToString(proofByte))
	buffer.WriteString("\nproof binary:" + fmt.Sprintf("%b", proofByte))
	buffer.WriteString("\n")

	fmt.Println(buffer.String())
}

func ECVRFProveDebug(pk PublicKey, sk PrivateKey, m []byte, kp []byte, ks []byte) (pi VRFProve, err error) {
	x := expandSecret(sk)
	fmt.Printf("x:%v\n", x)
	fmt.Printf("x binary: %b\n", *x)
	h := ecVRFHashToCurve(m, pk)
	osh := ecp2os(h)
	fmt.Printf("ecp2os h:%b\n", osh)
	r := ecp2os(geScalarMult(h, x))

	//kp, ks, err := GenerateKey(nil) // use GenerateKey to generate a random
	//if err != nil {
	//	return nil, err
	//}
	k := expandSecret(ks)

	// ECVRF_hash_points(g, h, g^x, h^x, g^k, h^k)
	c := ecVRFHashPoints(ecp2os(g), ecp2os(h), s2os(pk), r, s2os(kp), ecp2os(geScalarMult(h, k)))

	// s = k - c*x mod q
	var z big.Int
	s := z.Mod(z.Sub(f2ip(k), z.Mul(c, f2ip(x))), q)

	// pi = gamma || I2OSP(c, N) || I2OSP(s, 2N)
	var buf bytes.Buffer
	buf.Write(r) // 2N
	buf.Write(i2osp(c, N))
	buf.Write(i2osp(s, N2))
	return buf.Bytes(), nil
}

//func DoTestECVRF(t *testing.T, pk PublicKey, sk PrivateKey, msg []byte, verbose bool) {
//	pi, err := ECVRFProve(pk, sk, msg[:])
//	if err != nil {
//		t.Fatal(err)
//	}
//	res, err := ECVRFVerify(pk, pi, msg[:])
//	if err != nil {
//		t.Fatal(err)
//	}
//	if !res {
//		t.Errorf("VRF failed")
//	}
//
//	// when everything get through
//	if verbose {
//		fmt.Printf("alpha: %s\n", hex.EncodeToString(msg))
//		fmt.Printf("x: %s\n", hex.EncodeToString(sk))
//		fmt.Printf("P: %s\n", hex.EncodeToString(pk))
//		fmt.Printf("pi: %s\n", hex.EncodeToString(pi))
//		fmt.Printf("vrf: %s\n", hex.EncodeToString(ECVRFProof2hash(pi)))
//
//		r, c, s, err := ecVRFDecodeProof(pi)
//		if err != nil {
//			t.Fatal(err)
//		}
//		// u = (g^x)^c * g^s = P^c * g^s
//		var u edwards25519.ProjectiveGroupElement
//		P := os2ecp(pk, pk[31]>>7)
//		edwards25519.GeDoubleScalarMultVartime(&u, c, P, s)
//		fmt.Printf("r: %s\n", hex.EncodeToString(ecp2os(r)))
//		fmt.Printf("c: %s\n", hex.EncodeToString(c[:]))
//		fmt.Printf("s: %s\n", hex.EncodeToString(s[:]))
//		fmt.Printf("u: %s\n", hex.EncodeToString(ecp2osProj(&u)))
//	}
//}
