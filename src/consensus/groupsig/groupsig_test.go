package groupsig

//import (
//	"testing"
//	"x/src/consensus/model"
//	"x/src/common"
//	"fmt"
//)

//
//import (
//	"fmt"
//	"math/big"
//	"bytes"
//	"testing"
//
//	"x/src/consensus/base"
//	"x/src/common"
//)
//
////用big.Int生成私钥，取得公钥和签名。然后对私钥、公钥和签名各复制一份后测试加法后的验证是否正确。
////同时测试签名的序列化。
//func TestVerifyAggregateSign(t *testing.T) {
//	fmt.Printf("\nbegin test Comparison...\n")
//	t.Log("begin testComparison")
//	var b = new(big.Int)
//	b.SetString("16798108731015832284940804142231733909759579603404752749028378864165570215948", 10)
//	sec := NewSeckeyFromBigInt(b) //从big.Int（固定的常量）生成原始私钥
//	t.Log("sec.Hex: ", sec.GetHexString())
//
//	// Add Seckeys
//	sum := AggregateSeckeys([]Seckey{*sec, *sec}) //同一个原始私钥相加，生成聚合私钥
//	if sum == nil {
//		t.Error("AggregateSeckeys failed.")
//	}
//
//	// Pubkey
//	pub := GeneratePubkey(*sec) //从原始私钥萃取出公钥
//	if pub == nil {
//		t.Error("NewPubkeyFromSeckey failed.")
//	} else {
//		fmt.Printf("size of pub key = %v.\n", len(pub.Serialize()))
//	}
//
//	// Sig
//	sig := Sign(*sec, []byte("hi")) //以原始私钥对明文签名，生成原始签名
//	fmt.Printf("size of sign = %v\n.", len(sig.Serialize()))
//	asig := AggregateSigs([]Signature{sig, sig})                       //同一个原始签名相加，生成聚合签名
//	if !VerifyAggregateSig([]Pubkey{*pub, *pub}, []byte("hi"), asig) { //对同一个原始公钥进行聚合后（生成聚合公钥），去验证聚合签名
//		t.Error("Aggregated signature does not verify")
//	}
//	{
//		var sig2 Signature
//		err := sig2.SetHexString(sig.GetHexString()) //测试原始签名的字符串导出
//		if err != nil || !sig.IsEqual(sig2) {        //检查字符串导入生成的签名是否和之前的签名相同
//			t.Error("sig2.SetHexString")
//		}
//	}
//	{
//		var sig2 Signature
//		err := sig2.Deserialize(sig.Serialize()) //测试原始签名的序列化
//		if err != nil || !sig.IsEqual(sig2) {    //检查反序列化生成的签名是否跟之前的签名相同
//			t.Error("sig2.Deserialize")
//		}
//	}
//	t.Log("end testComparison")
//	fmt.Printf("\nend test Comparison.\n")
//}
//
//// AggregateXXX函数族是把全部切片相加，而不是k个相加。
////签名聚合函数。
//func AggregateSigs(sigs []Signature) (sig Signature) {
//	n := len(sigs)
//	sig = Signature{}
//	if n >= 1 {
//		sig.value.Set(&sigs[0].value)
//		for i := 1; i < n; i++ {
//			newsig := &Signature{}
//			newsig.value.Set(&sig.value)
//			sig.value.Add(&newsig.value, &sigs[i].value)
//		}
//	}
//	return sig
//}
//
////分片合并验证函数。先把公钥切片合并，然后验证该签名是否来自公钥对应的私钥。
//func VerifyAggregateSig(pubs []Pubkey, msg []byte, asig Signature) bool {
//	pub := AggregatePubkeys(pubs) //用公钥切片合并出组公钥（全部公钥切片而不只是k个）
//	if pub == nil {
//		return false
//	}
//	return VerifySig(*pub, msg, asig) //调用验证函数
//}
//
//func TestGroupsigIDStringConvert(t *testing.T) {
//	str := "0xedb67046af822fd6a778f3a1ec01ad2253e5921d3c1014db958a952fdc1b98e2"
//	id := NewIDFromString(str)
//	s := id.GetHexString()
//	fmt.Printf("id str:%s\n", s)
//	fmt.Printf("id str compare result:%t\n", str == s)
//}
//
//func TestGroupsigIDDeserialize(t *testing.T) {
//	s := "abc"
//	id1 := DeserializeID([]byte(s))
//	id2 := NewIDFromString(s)
//	t.Log(id1.GetHexString(), id2.GetHexString(), id1.IsEqual(*id2))
//
//	t.Log([]byte(s))
//	t.Log(id1.Serialize(), id2.Serialize())
//	t.Log(id1.GetHexString(), id2.GetHexString())
//
//	b := id2.Serialize()
//	id3 := DeserializeID(b)
//	t.Log(id3.GetHexString())
//}
//
////从字符串生成ID 传入的STRING必须保证离散性
//func NewIDFromString(s string) *ID {
//	bi := new(big.Int).SetBytes([]byte(s))
//	return NewIDFromBigInt(bi)
//}
//
//
//func BenchmarkGroupsigRecover100(b *testing.B)  { benchmarkGroupsigRecover(100, 100, b) }
//func BenchmarkGroupsigRecover200(b *testing.B)  { benchmarkGroupsigRecover(200, 200, b) }
//func BenchmarkGroupsigRecover500(b *testing.B)  { benchmarkGroupsigRecover(500, 500, b) }
//func BenchmarkGroupsigRecover1000(b *testing.B) { benchmarkGroupsigRecover(1000, 1000, b) }
//
////测试BLS门限签名.
////Added by FlyingSquirrel-Xu. 2018-08-24.
//func mockBLS(n int, k int, b *testing.B) {
//	//n := 50
//	//k := 10
//
//	//定义k-1次多项式 F(x): <a[0], a[1], ..., a[k-1]>. F(0)=a[0].
//	a := make([]Seckey, k)
//	r := base.NewRand()
//	for i := 0; i < k; i++ {
//		a[i] = *NewSeckeyFromRand(r.Deri(i))
//	}
//	//fmt.Println("a[0]:", a[0].Serialize())
//
//	//生成n个成员ID: {IDi}, i=1,..,n.
//	ids := make([]ID, n)
//	for i := 0; i < n; i++ {
//		ids[i] = *newIDFromInt64(int64(i + 3)) //生成50个ID
//	}
//
//	//计算得到多项式F(x)上的n个点 <IDi, Si>. 满足F(IDi)=Si.
//	secs := make([]Seckey, n) //私钥切片
//	for j := 0; j < n; j++ {
//		bs := ShareSeckey(a, ids[j])
//		secs[j].value.SetBigInt(bs.value.GetBigInt())
//	}
//
//	//通过Lagrange插值公式, 由{<IDi, Si>|i=1..n}计算得到组签名私钥s=F(0).
//	new_secs := secs[:k]
//	s := recoverMasterSeckey(new_secs, ids)
//	//fmt.Println("s:", s.Serialize())
//
//	//检查 F(0) = a[0]?
//	if !bytes.Equal(a[0].Serialize(), s.Serialize()) {
//		fmt.Errorf("secreky Recover failed.")
//	}
//
//	//通过组签名私钥s得到组签名公钥 pub.
//	pub := GeneratePubkey(*s)
//
//	//成员签名: H[i] = si·H(m)
//	sig := make([]Signature, n)
//	for i := 0; i < n; i++ {
//		sig[i] = Sign(secs[i], []byte("hi")) //以原始私钥对明文签名，生成原始签名
//	}
//
//	//Recover组签名: H = ∑ ∆i(0)·Hi.
//	new_sig := sig[:k]
//	H := RecoverSignature(new_sig, ids) //调用big.Int加法求模的私钥恢复函数
//
//	//组签名验证：Pair(H,Q)==Pair(Hm,Pub)?
//	result := VerifySig(*pub, []byte("hi"), *H)
//	if result != true {
//		fmt.Errorf("VerifySig failed.")
//	}
//}
//
//
////用（签名）私钥分片切片和id切片恢复出master私钥（通过拉格朗日插值法）
////私钥切片和ID切片的数量固定为门限值k
////实际情况不存在这个SECKEY的
//func recoverMasterSeckey(secs []Seckey, ids []ID) *Seckey {
//	secret := big.NewInt(0) //组私钥
//	k := len(secs)          //取得输出切片的大小，即门限值k
//	//fmt.Println("k:", k)
//	xs := make([]*big.Int, len(ids))
//	for i := 0; i < len(xs); i++ {
//		xs[i] = ids[i].GetBigInt() //把所有的id转化为big.Int，放到xs切片
//	}
//	// need len(ids) = k > 0
//	for i := 0; i < k; i++ { //输入元素遍历
//		// compute delta_i depending on ids only
//		//为什么前面delta/num/den初始值是1，最后一个diff初始值是0？
//		var delta, num, den, diff *big.Int = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
//		for j := 0; j < k; j++ { //ID遍历
//			if j != i { //不是自己
//				num.Mul(num, xs[j])      //num值先乘上当前ID
//				num.Mod(num, curveOrder) //然后对曲线域求模
//				diff.Sub(xs[j], xs[i])   //diff=当前节点（内循环）-基节点（外循环）
//				den.Mul(den, diff)       //den=den*diff
//				den.Mod(den, curveOrder) //den对曲线域求模
//			}
//		}
//		// delta = num / den
//		den.ModInverse(den, curveOrder) //模逆
//		delta.Mul(num, den)
//		delta.Mod(delta, curveOrder)
//		//最终需要的值是delta
//		// apply delta to secs[i]
//		delta.Mul(delta, secs[i].GetBigInt()) //delta=delta*当前节点私钥的big.Int
//		// skip reducing delta modulo curveOrder here
//		secret.Add(secret, delta)      //把delta加到组私钥（big.Int形式）
//		secret.Mod(secret, curveOrder) //组私钥对曲线域求模（big.Int形式）
//	}
//
//	return NewSeckeyFromBigInt(secret)
//}
//
//
//func benchmarkGroupsigRecover(n int, k int, b *testing.B) {
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		mockBLS(n, k, b)
//	}
//}
//
//
//
//
//
//func benchmarkDeriveSeckeyShare(k int, b *testing.B) {
//	b.StopTimer()
//
//	r := base.NewRand() //生成随机数
//	sec := NewSeckeyFromRand(r.Deri(1))
//
//	msk := sec.GetMasterSecretKey(k)
//	var id ID
//	for n := 0; n < b.N; n++ {
//		err := id.Deserialize([]byte{1, 2, 3, 4, 5, byte(n)})
//		if err != nil {
//			b.Error(err)
//		}
//		b.StartTimer()
//		err = sec.Set(msk, &id)
//		b.StopTimer()
//		if err != nil {
//			b.Error(err)
//		}
//	}
//}
//
//
//// GetMasterSecretKey --
//func (sec *Seckey) GetMasterSecretKey(k int) (msk []Seckey) {
//	msk = make([]Seckey, k)
//	msk[0] = *sec
//
//	r := base.NewRand() //生成随机数
//	for i := 1; i < k; i++ {
//		msk[i] = *NewSeckeyFromRand(r.Deri(1))
//	}
//	return msk
//}
//
//func (sec *Seckey) Set(msk []Seckey, id *ID) error {
//	// #nosec
//	s := ShareSeckey(msk, *id)
//	sec.Deserialize(s.Serialize())
//	return nil
//}
//
//func BenchmarkDeriveSeckeyShare500(b *testing.B) { benchmarkDeriveSeckeyShare(500, b) }
//
//func benchmarkRecoverSeckey(k int, b *testing.B) {
//	b.StopTimer()
//
//	r := base.NewRand() //生成随机数
//	sec := NewSeckeyFromRand(r.Deri(1))
//
//	msk := sec.GetMasterSecretKey(k)
//
//	// derive n shares
//	n := k
//	secVec := make([]Seckey, n)
//	idVec := make([]ID, n)
//	for i := 0; i < n; i++ {
//		err := idVec[i].Deserialize([]byte{1, 2, 3, 4, 5, byte(i)})
//		if err != nil {
//			b.Error(err)
//		}
//		err = secVec[i].Set(msk, &idVec[i])
//		if err != nil {
//			b.Error(err)
//		}
//	}
//
//	// recover from secVec and idVec
//	var sec2 Seckey
//	b.StartTimer()
//	for n := 0; n < b.N; n++ {
//		err := sec2.Recover(secVec, idVec)
//		if err != nil {
//			b.Errorf("%s\n", err)
//		}
//	}
//}
//// Recover --
//func (sec *Seckey) Recover(secVec []Seckey, idVec []ID) error {
//	// #nosec
//	s := recoverMasterSeckey(secVec, idVec)
//	sec.Deserialize(s.Serialize())
//
//	return nil
//}
//
//func BenchmarkRecoverSeckey100(b *testing.B)  { benchmarkRecoverSeckey(100, b) }
//func BenchmarkRecoverSeckey200(b *testing.B)  { benchmarkRecoverSeckey(200, b) }
//func BenchmarkRecoverSeckey500(b *testing.B)  { benchmarkRecoverSeckey(500, b) }
//func BenchmarkRecoverSeckey1000(b *testing.B) { benchmarkRecoverSeckey(1000, b) }
//
//func benchmarkRecoverSignature(k int, b *testing.B) {
//	b.StopTimer()
//
//	r := base.NewRand() //生成随机数
//	sec := NewSeckeyFromRand(r.Deri(1))
//
//	msk := sec.GetMasterSecretKey(k)
//
//	// derive n shares
//	n := k
//	idVec := make([]ID, n)
//	secVec := make([]Seckey, n)
//	signVec := make([]Signature, n)
//	for i := 0; i < n; i++ {
//		err := idVec[i].Deserialize([]byte{1, 2, 3, 4, 5, byte(i)})
//		if err != nil {
//			b.Error(err)
//		}
//		err = secVec[i].Set(msk, &idVec[i])
//		if err != nil {
//			b.Error(err)
//		}
//		signVec[i] = Sign(secVec[i], []byte("test message"))
//	}
//
//	// recover signature
//	b.StartTimer()
//	for n := 0; n < b.N; n++ {
//		RecoverSignature(signVec, idVec)
//	}
//}
//
//func BenchmarkRecoverSignature100(b *testing.B)  { benchmarkRecoverSignature(100, b) }
//func BenchmarkRecoverSignature200(b *testing.B)  { benchmarkRecoverSignature(200, b) }
//func BenchmarkRecoverSignature500(b *testing.B)  { benchmarkRecoverSignature(500, b) }
//func BenchmarkRecoverSignature1000(b *testing.B) { benchmarkRecoverSignature(1000, b) }
//
//
//
//func TestIDAndAddress(t *testing.T) {
//	addr := common.HexToAddress("0x0bf03e69b31aa1caa45e79dd8d7f8031bfe81722d435149ffa2d0b66b9e9b6b7")
//	id := DeserializeID(addr.Bytes())
//	t.Log(id.GetHexString(), len(id.GetHexString()), addr.GetHexString() == id.GetHexString())
//	t.Log(id.Serialize(), addr.Bytes(), bytes.Equal(id.Serialize(), addr.Bytes()))
//
//	id2 := ID{}
//	id2.SetHexString("0x0bf03e69b31aa1caa45e79dd8d7f8031bfe81722d435149ffa2d0b66b9e9b6b7")
//	t.Log(id2.GetHexString())
//}


