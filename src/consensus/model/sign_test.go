package model

import (
	"testing"
	"fmt"
	"x/src/common"
	"x/src/consensus/groupsig"
)

var m msg
type msg struct{}

func(m msg)GenHash()common.Hash{
	return common.HexToHash("0x90826cfaf8558749e57824a5dcee981b27ed1ca7765c555985623b42f4e10e66")
}

func TestSig(t *testing.T) {
	var seckey groupsig.Seckey
	seckey.SetHexString("0x12729af40da24bb700561689b268ae39a3da1488107296961ea4b28ab5ccb1d0")
	fmt.Printf("seckey:%v\n",seckey.GetHexString())
	fmt.Printf("pubkey:%v\n",groupsig.GeneratePubkey(seckey).GetHexString())



	var id groupsig.ID
	id.SetHexString("0xf9f7ed82123526e69c2f12ed281572a692e6fc4664e4af76141e84ee619dc16c")

	sign,ok :=NewSignInfo(seckey,id,m)
	fmt.Printf("hash:%v\n",sign.GetDataHash().String())
	fmt.Printf("sign result:%v,signature:%v\n",ok,sign.GetSignature().GetHexString())

	var pubkey groupsig.Pubkey
	pubkey.SetHexString("0x099d62ec71384cb7289ef678b93d8d3e577830d712a9a9d48361370586cf5d0c1d199ff9d49eb254890b50004c7414024bba164619752ffb174a55660f53a3f219983238f0bd8a925a5fa03c7bfa1e3f427663bc0b641c050300c8ec550413b804370c00cf40bd7d36762e939ec896cf84a74b61d0f50dea7b79036099cec78c")
	result := sign.VerifySign(pubkey)
	fmt.Printf("verify sign result:%v\n",result)


}
