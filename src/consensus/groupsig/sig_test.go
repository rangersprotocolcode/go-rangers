package groupsig

import (
	"testing"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"encoding/hex"
	"fmt"
	"strconv"
)

func TestPubkeyToAddress(t *testing.T) {
	var gpk Pubkey
	gpk.SetHexString("0x04e32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a0a2b2667f7e725ceea70c673093bf67663e0312623c8e091b13cf2c0f11ef652")

	fmt.Printf(gpk.GetAddress().String())
}

func TestSign(t *testing.T) {
	secKey := new(Seckey)
	secKey.SetHexString("0x14262cc0dc2954d7283bb990e6eb4c075eab67356c3c83c6406f5019e4c1ce80")
	fmt.Printf("secKey key:%s\n", secKey.GetHexString())

	msg,_:= hex.DecodeString("4e976a0d30a783c9ad0dfcefb35e0d6c5a98bc35eb41639c696498917e6c23c6")
	sign := Sign(*secKey,msg)
	fmt.Printf("Sign:%s\n", sign.GetHexString())
}

func TestVerifySig(t *testing.T) {
	var sign Signature
	sign.SetHexString("0x8c24451f4e2263916f315d24e96d81d9d06bf819f0c5b41db6421e515520f47e06bf10ef5f7b12b5e0343224abf2b89cb5cbdb1847e8743bec41bae93f03f8f8")
	var gpk Pubkey
	//gpk.SetHexString("0x4a7e2d37757ae2ff0cddde029c6395d8d9f389b122a845e7340fb43d057bf08561c10baf2e86dc53c1c251a892ab8f2dcc33b2160d46b4193b444b6bdf6c3dd92b455c330c4760ec994be4445cab3d1e2df752ba6224ba84c35a60d282c580b77f8439de3d381c59f0628c31ec5f1154d907128d42069df9ccf602a135f83b0a")
	gpk.SetHexString("0x61c00ae61afad998d12833ba3013bff85c4fbf97e182aeaa206de487ae7175dc266567add1594cfd1ae952bc2fba5b375119384f6a6fa764e5839064ae417d5d2abe8b2c5ec9db26ec1a938e4dcc24a569adaa7e629035943edf683cb87c98a21198a21362dc3c55998398cdec2b9f4ef531dc27e1c9b11e137b63ac609acb55")
	var hash = common.HexToHash("0x4e976a0d30a783c9ad0dfcefb35e0d6c5a98bc35eb41639c696498917e6c23c6")

	t.Log(VerifySig(gpk, hash.Bytes(), sign))
}

func TestSignature(t *testing.T) {
	var sig Signature
	sig.SetHexString("0x83b13efab6ffa9219e1adf6dd2c070b14e2d31449f01de838bfcf4554bbe82dd0d4a6b38a286f3aa395d60bfe61ef408008d7d1f1cc1e5d6769a6e14c706d991")
	bs := sig.GetHexString()

	t.Log("len of bs:", len(bs))
	t.Log(string(bs))
}

func TestRecoverGroupSignature(t *testing.T) {
	m := make(map[string]Signature)
	var sign1 Signature
	sign1.SetHexString("0x83b13efab6ffa9219e1adf6dd2c070b14e2d31449f01de838bfcf4554bbe82dd0d4a6b38a286f3aa395d60bfe61ef408008d7d1f1cc1e5d6769a6e14c706d991")
	seckey1 := "0xedcf4a3457361ed75e9d4531362b5d1c8a482c94db3fdb815f821fc18ef59d89"
	m[seckey1] = sign1

	var sign2 Signature
	sign2.SetHexString("0x8a0550bb40575dc9c3e7873dae99dc5bc50f812b37b9364f1027073caa77449e29b0e07dec08d96117a337307c419b43cb14c620e4d2edc1749072bc595731c7")
	seckey2 := "0x05b231ac9481c219957ca7965efecd24c84b2b2bdff6f4b18f090e5059aa87ea"
	m[seckey2] = sign2

	var sign3 Signature
	sign3.SetHexString("0x72a532366b41f147f7d3b14530fbef1d57c52f1f6d58c29e26c01d51061043b275e4b71e9b490177f32413231245861c5daa09ca51cfc1d19e68e1bfd7c3f951")
	seckey3 := "0xdd8d185f2f9c1fb8a990f790d344922e815497e2cb4924a1643c108add7f2823"
	m[seckey3] = sign3

	var sign4 Signature
	sign4.SetHexString("0x8067d62030887bfe90fe97ad10baa3e29fd8939115d462a5cde1a086223810aa0a0275bdbdad4a7985ef1597c590df8d67924d8cedbf68e0311f2a538792f104")
	seckey4 := "0x286595ade352c80fdf26b5863d4aed7ece85cffb0164d024b96e64fa0ae10a6d"
	m[seckey4] = sign4

	var sign5 Signature
	sign5.SetHexString("0x0565c4f9d623159284ecf13c1b919a0315a9c544145403c7779f9999798885686d3fadc29d1781ec67c5dd16957693734a9a8ba0eba447bb4f5f335b4dfc7224")
	seckey5 := "0x97cd69b5a9f4d557574dcd1375f6d894fd130906cea2af42bdb0888908e07a44"
	m[seckey5] = sign5

	var sign6 Signature
	sign6.SetHexString("0x2027c667de9c046c77ee83cb58733436fcd8b3b59e5cd5ba3b3cc5ea3856cc0b8f1d4a177e22f29b7e40bd13cd64e4a33095580f3f8919eff569889280b010d4")
	seckey6 := "0x8510cb890b11424b8d0986f7f3574af96fb00e452c0dbf658048a53a6451855d"
	m[seckey6] = sign6

	var sign7 Signature
	sign7.SetHexString("0x8f8439909ee824299ed8be437c6a2a094bd87ee16acac300fed6882e5f30090c1b158ca4689898a89f6d93c2f6102b9be8df8436e3e5c5eae43bb6908ae4205b")
	seckey7 := "0x34f9079ff55e258ae82118acac676d7bbb014e2584875b3eebd5cb320c5613e2"
	m[seckey7] = sign7

	var sign8 Signature
	sign8.SetHexString("0x1b049159202d3bfd475d59878f18e2843b1f10b1a8fb4c649649aee9629af5354e6d10feb232eed88d1b1bf75fda59ed1d22fb2db62069091645953d9ff49eaa")
	seckey8 := "0x4b7cc1b4eb7aaab669f1a5eb9ebd382cfa6e2a28ac2f1ab6fbb0247219a00ad6"
	m[seckey8] = sign8

	groupSign := RecoverGroupSignature(m, 8)
	fmt.Printf("group sign:%s\n",groupSign.GetHexString())
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
