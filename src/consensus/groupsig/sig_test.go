package groupsig

import (
	"testing"

	"x/src/common"
	"fmt"
	"x/src/common/secp256k1"
	"strconv"
	"x/src/consensus/base"
)


func TestPubkeyToAddress(t *testing.T){
	var gpk Pubkey
	gpk.SetHexString("0x04e32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a0a2b2667f7e725ceea70c673093bf67663e0312623c8e091b13cf2c0f11ef652")

	fmt.Printf(gpk.GetAddress().String())
}

func TestSign(t *testing.T){
	privateKeyStr := "0x0415a3f885882169f6b740059c12dccbd4776acc7a90c0387b90418166ad9364a9f3c7fd293af8808bdea7d1f00371cbf0dbd44199d6288b7d833d25d6f8c74f7162f0f08eeb8db3a2a710397c4ddfcdb61512dbca31da457995d86e51c0f66679"
	privateKey := common.HexStringToSecKey(privateKeyStr)
	fmt.Printf("Private key:%s\n",privateKey.GetHexString())

	pubkey := privateKey.GetPubKey()
	fmt.Printf("Public key:%s\n",pubkey.GetHexString())

	msg := "abcdef"
	sign:= privateKey.Sign([]byte(msg))
	fmt.Printf("Sign:%s\n",sign.GetHexString())

	compress := secp256k1.CompressPubkey(pubkey.PubKey.X,pubkey.PubKey.Y)
	fmt.Printf("Compress pubkey:%02x",compress)
}

func TestVerifySig(t *testing.T) {
	var sign Signature
	sign.SetHexString("0x041eda274745e471ea9b4ee4da8d1ba9667fee6f32399531c55cd288a40525b501")
	var gpk Pubkey
	gpk.SetHexString("0x1a471817d295dc93c8492a84116f1a7200f88504eec8754be234df104583f3b820de4dc3fb6aca4b4873fb714dc39879756dd8d2b6cbeaf1d722032b664b129421bf91f4ed295c0d2ed27fe05cb3f641f24f014641471eee492e3e961f8fd6c40523abd48aec1bf89d292ad2643e4daefc34f74c35dcb0df057c517dc9ea6887")
	var hash = common.HexToHash("0x05830417649117d587035cdd5f9f874c98ceba8423640277bf7ed8657ea2b211verifySign")

	t.Log(VerifySig(gpk, hash.Bytes(), sign))
}


func TestSignature(t *testing.T) {
	var sig Signature
	sig.SetHexString("0x0724b751e096becd93127a5be441989a9fd8fe328828f6ce5e1817c70bf10f2f00")
	bs := sig.GetHexString()

	t.Log("len of bs:", len(bs))
	t.Log(string(bs))
}

func BenchmarkSign(b *testing.B) {
	b.StopTimer()

	r := base.NewRand() //生成随机数

	for n := 0; n < b.N; n++ {
		sec := NewSeckeyFromRand(r.Deri(1))
		b.StartTimer()
		Sign(*sec, []byte(strconv.Itoa(n)))
		b.StopTimer()
	}
}

func BenchmarkVerifySign(b *testing.B) {
	b.StopTimer()

	r := base.NewRand() //生成随机数

	for n := 0; n < b.N; n++ {
		sec := NewSeckeyFromRand(r.Deri(1))
		pub := GeneratePubkey(*sec)
		m := strconv.Itoa(n)
		sig := Sign(*sec, []byte(m))
		b.StartTimer()
		result := VerifySig(*pub, []byte(m), sig)
		if result != true {
			fmt.Println("VerifySig failed.")
		}
		b.StopTimer()
	}
}



func BenchmarkVerifySign2(b *testing.B) {
	b.StopTimer()
	r := base.NewRand() //生成随机数
	for n := 0; n < b.N; n++ {
		sec := NewSeckeyFromRand(r.Deri(1))
		pub := GeneratePubkey(*sec)
		m := strconv.Itoa(n) + "abc"

		sig := Sign(*sec, []byte(m))
		buf := sig.Serialize()
		sig2 := DeserializeSign(buf)

		b.StartTimer()
		result := VerifySig(*pub, []byte(m), *sig2)
		if result != true {
			fmt.Println("VerifySig failed.")
		}
		b.StopTimer()
	}
}