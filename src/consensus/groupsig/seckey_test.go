package groupsig

import (
	"testing"
	"fmt"
	"math/big"
	"x/src/consensus/base"
)

//测试从big.Int生成私钥，以及私钥的序列化
func TestSeckey(t *testing.T) {
	fmt.Printf("\nbegin test sec key...\n")
	t.Log("testSeckey")
	s := "401035055535747319451436327113007154621327258807739504261475863403006987855"
	var b = new(big.Int)
	b.SetString(s, 10)

	sec := NewSeckeyFromBigInt(b)
	str := sec.GetHexString()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)

	{
		var sec2 Seckey
		err := sec2.SetHexString(str) //测试私钥的十六进制字符串导出
		if err != nil || !sec.IsEqual(sec2) { //检查字符串导入生成的私钥是否和之前的私钥相同
			t.Error("bad SetHexString")
		}
		str = sec2.GetHexString()
		fmt.Printf("sec key import and export again, len=%v, data=%v.\n", len(str), str)
	}

	{
		var sec2 Seckey
		err := sec2.Deserialize(sec.Serialize()) //测试私钥的序列化
		if err != nil || !sec.IsEqual(sec2) { //检查反序列化生成的私钥是否和之前的私钥相同
			t.Error("bad Serialize")
		}
	}
	fmt.Printf("end test sec key.\n")
}

//生成n个衍生随机数私钥，然后针对一个特定的ID生成分享片段
func TestShareSeckey(t *testing.T) {
	fmt.Printf("\nbegin testShareSeckey...\n")
	t.Log("testShareSeckey")
	n := 100
	msec := make([]Seckey, n)
	r := base.NewRand()
	for i := 0; i < n; i++ {
		msec[i] = *NewSeckeyFromRand(r.Deri(i)) //生成100个随机私钥
	}
	id := *newIDFromInt64(123)  //随机生成一个ID
	s2 := ShareSeckey(msec, id) //分享函数

	fmt.Printf("Share piece seckey:%s", s2.GetHexString())
	fmt.Printf("end testShareSeckey.\n")
}

//生成n个衍生随机数私钥，对这n个衍生私钥进行聚合生成组私钥，然后萃取出组公钥
//Todo 这里萃取出来的是啥？
func TestAggregateSeckeys(t *testing.T) {
	fmt.Printf("\nbegin test Aggregation...\n")
	t.Log("testAggregation")
	n := 100
	r := base.NewRand()                      //生成随机数基
	seckeyContributions := make([]Seckey, n) //私钥切片
	for i := 0; i < n; i++ {
		seckeyContributions[i] = *NewSeckeyFromRand(r.Deri(i)) //以r为基，i为递增量生成n个相关性私钥
	}
	groupSeckey := AggregateSeckeys(seckeyContributions) //对n个私钥聚合，生成组私钥
	groupPubkey := GeneratePubkey(*groupSeckey)          //从组私钥萃取出组公钥
	t.Log("Group pubkey:", groupPubkey.GetHexString())
	fmt.Printf("end test Aggregation.\n")
}

//由int64创建ID
func newIDFromInt64(i int64) *ID {
	return newIDFromBigInt(big.NewInt(i))
}

//
////私钥恢复函数，m为map(地址->私钥)，k为门限值
//func RecoverSeckeyByMap(m SeckeyMap, k int) *Seckey {
//	ids := make([]ID, k)
//	secs := make([]Seckey, k)
//	i := 0
//	for a, s := range m { //map遍历
//		id := NewIDFromAddress(a) //提取地址对应的id
//		if id == nil {
//			log.Printf("RecoverSeckeyByMap bad Address %s\n", a)
//			return nil
//		}
//		ids[i] = *id //组成员ID
//		secs[i] = s  //组成员签名私钥
//		i++
//		if i >= k { //取到门限值
//			break
//		}
//	}
//	return RecoverSeckey(secs, ids) //调用私钥恢复函数
//}
//
//// RecoverSeckeyByMapInt --
////从签名私钥分片map中取k个（门限值）恢复出组私钥
//func RecoverSeckeyByMapInt(m SeckeyMapInt, k int) *Seckey {
//	ids := make([]ID, k)      //k个ID
//	secs := make([]Seckey, k) //k个签名私钥分片
//	i := 0
//	//取map里的前k个签名私钥生成恢复基
//	for a, s := range m {
//		ids[i] = *newIDFromInt64(int64(a))
//		secs[i] = s
//		i++
//		if i >= k {
//			break
//		}
//	}
//	//恢复出组私钥
//	return RecoverSeckey(secs, ids)
//}
//
//// Set --
//func (sec *Seckey) Set(msk []Seckey, id *ID) error {
//	// #nosec
//	s := ShareSeckey(msk, *id)
//	sec.Deserialize(s.Serialize())
//	return nil
//}
//
