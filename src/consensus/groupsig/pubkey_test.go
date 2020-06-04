package groupsig

import (
	"testing"
	"fmt"
	"x/src/consensus/base"
)

//测试用衍生随机数生成私钥，从私钥萃取公钥，以及公钥的序列化
func TestPubkey(t *testing.T) {
	fmt.Printf("\nbegin test pub key...\n")
	t.Log("testPubkey")
	r := base.NewRand() //生成随机数

	fmt.Printf("size of rand = %v\n.", len(r))
	sec := NewSeckeyFromRand(r.Deri(1)) //以r的衍生随机数生成私钥
	if sec == nil {
		t.Fatal("NewSeckeyFromRand")
	}

	pub := GeneratePubkey(*sec) //从私钥萃取出公钥
	if pub == nil {
		t.Log("NewPubkeyFromSeckey")
	}

	{
		var pub2 Pubkey
		err := pub2.SetHexString(pub.GetHexString()) //测试公钥的字符串导出
		if err != nil || !pub.IsEqual(pub2) {        //检查字符串导入生成的公钥是否跟之前的公钥相同
			t.Log("pub != pub2")
		}
	}
	{
		var pub2 Pubkey
		err := pub2.Deserialize(pub.Serialize()) //测试公钥的序列化
		if err != nil || !pub.IsEqual(pub2) {    //检查反序列化生成的公钥是否跟之前的公钥相同
			t.Log("pub != pub2")
		}
	}
	fmt.Printf("\nend test pub key.\n")
}



func BenchmarkPubkeyFromSeckey(b *testing.B) {
	b.StopTimer()

	r := base.NewRand() //生成随机数

	//var sec Seckey
	for n := 0; n < b.N; n++ {
		//sec.SetByCSPRNG()
		sec := NewSeckeyFromRand(r.Deri(1))
		b.StartTimer()
		GeneratePubkey(*sec)
		b.StopTimer()
	}
}

