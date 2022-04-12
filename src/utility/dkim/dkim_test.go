package dkim

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"github.com/go-playground/validator/v10"
	"math"
	"math/big"
	"testing"
)

func TestNewHead(t *testing.T) {
	header := NewHeader("123", "456")
	fmt.Println(header)
}

func TestValidate(t *testing.T) {
	header := Header{}
	err := validator.New().Struct(header)
	if nil != err {
		fmt.Println(err.Error())
	} else {
		fmt.Println("end")
	}
}

func TestReadMail(t *testing.T) {
	asString, err := ReadFileAsString("special_ch.eml", "")
	if nil != err {
		fmt.Println(err)
		return
	}

	fmt.Println(asString)
	email, err := FromString(asString)
	if nil != err {
		fmt.Println(err)
		return
	}
	fmt.Println(email)

	message, err := email.getDkimMessage()
	if nil != err {
		fmt.Println(err)
		return
	}
	fmt.Println(message)

	N := "cfb0520e4ad78c4adb0deb5e605162b6469349fc1fde9269b88d596ed9f3735c00c592317c982320874b987bcc38e8556ac544bdee169b66ae8fe639828ff5afb4f199017e3d8e675a077f21cd9e5c526c1866476e7ba74cd7bb16a1c3d93bc7bb1d576aedb4307c6b948d5b8c29f79307788d7a8ebf84585bf53994827c23a5"
	E := 65537
	bigN := new(big.Int)
	_, ok := bigN.SetString(N, 16)
	if !ok {
		fmt.Println("error big int")
		return
	}
	pub := rsa.PublicKey{
		N: bigN,
		E: E,
	}

	digest := sha256.Sum256([]byte(message))
	verifyErr := rsa.VerifyPKCS1v15(&pub, crypto.SHA256, digest[:], email.DkimHeader.Signature)

	if nil != verifyErr {
		fmt.Println("verify err")
		return
	}
	fmt.Println("ok")

}

func TestVerify(t *testing.T) {
	asString, err := ReadFileAsString("special_ch.eml", "")
	if nil != err {
		fmt.Println(err)
		return
	}

	res := Verify(utility.StrToBytes(asString))
	fmt.Println(common.ToHex(res))
	fmt.Println(len(res))
}

func TestRSA(t *testing.T) {

	h := uint64(0)
	h = math.MaxUint64
	fmt.Println(h)
	fmt.Println(h==math.MaxUint64)
	h += 50
	fmt.Println(h)

	p, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := p.PublicKey

	fmt.Println(pk.Size())
	fmt.Println(pk.E)
	fmt.Println(pk.N.String())
}

//func bytesReverse(source []byte) []byte {
//	for i, j := 0, len(source)-1; i < j; i, j = i+1, j-1 {
//		source[i],source[j] = source[j],source[i]
//	}
//	return source
//}
