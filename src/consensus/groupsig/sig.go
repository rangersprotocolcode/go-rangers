package groupsig

import (
	"sort"
	"bytes"
	"fmt"
	"math/big"

	"x/src/common"
	bn_curve "x/src/consensus/groupsig/bn256"
	"x/src/consensus/base"
)

type Signature struct {
	value bn_curve.G1
}

func DeserializeSign(b []byte) *Signature {
	sig := &Signature{}
	sig.Deserialize(b)
	return sig
}

//把签名转换为字节切片
func (sig Signature) Serialize() []byte {
	if sig.IsNil() {
		return []byte{}
	}
	return sig.value.Marshal()
}

//由字节切片初始化签名
func (sig *Signature) Deserialize(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("signature Deserialized failed.")
	}
	sig.value.Unmarshal(b)
	return nil
}

//把签名转换为十六进制字符串
func (sig Signature) GetHexString() string {
	return PREFIX + common.Bytes2Hex(sig.value.Marshal())
}

//由十六进制字符串初始化签名
func (sig *Signature) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]

	if sig.value.IsNil() {
		sig.value = bn_curve.G1{}
	}

	sig.value.Unmarshal(common.Hex2Bytes(buf))
	return nil
}

func (sig *Signature) IsNil() bool {
	return sig.value.IsNil()
}

//比较两个签名是否相同
func (sig Signature) IsEqual(rhs Signature) bool {
	return bytes.Equal(sig.value.Marshal(), rhs.value.Marshal())
}

func (sig Signature) IsValid() bool {
	s := sig.Serialize()
	if len(s) == 0 {
		return false
	}

	return sig.value.IsValid()
}

//签名函数。用私钥对明文（哈希）进行签名，返回签名对象
func Sign(sec Seckey, msg []byte) (sig Signature) {
	bg := hashToG1(string(msg))
	sig.value.ScalarMult(bg, sec.GetBigInt())
	return sig
}

//验证函数。验证某个签名是否来自公钥对应的私钥。
func VerifySig(pub Pubkey, msg []byte, sig Signature) bool {
	if sig.IsNil() || !sig.IsValid() {
		return false
	}
	if !pub.IsValid() {
		return false
	}
	if sig.value.IsNil() {
		return false
	}
	bQ := bn_curve.GetG2Base()
	p1 := bn_curve.Pair(&sig.value, bQ)

	Hm := hashToG1(string(msg))
	p2 := bn_curve.Pair(Hm, &pub.value)

	return bn_curve.PairIsEuqal(p1, p2)
}

//签名恢复函数，m为map(ID->签名)，k为门限值
func RecoverGroupSignature(memberSignMap map[string]Signature, thresholdValue int) *Signature {
	if thresholdValue < len(memberSignMap) {
		memberSignMap = getRandomKSignInfo(memberSignMap, thresholdValue)
	}
	ids := make([]ID, thresholdValue)
	sigs := make([]Signature, thresholdValue)
	i := 0
	for s_id, si := range memberSignMap { //map遍历
		var id ID
		id.SetHexString(s_id)
		ids[i] = id  //组成员ID值
		sigs[i] = si //组成员签名
		i++
		if i >= thresholdValue {
			break
		}
	}
	return recoverSignature(sigs, ids) //调用签名恢复函数
}

func (sig Signature) ShortS() string {
	str := sig.GetHexString()
	return common.ShortHex12(str)
}

func getRandomKSignInfo(memberSignMap map[string]Signature, k int) map[string]Signature {
	indexs := base.NewRand().RandomPerm(len(memberSignMap), k)
	sort.Ints(indexs)
	ret := make(map[string]Signature)

	i := 0
	j := 0
	for key, sign := range memberSignMap {
		if i == indexs[j] {
			ret[key] = sign
			j++
			if j >= k {
				break
			}
		}
		i++
	}
	return ret
}

//用签名切片和id切片恢复出master签名（通过拉格朗日插值法）
//RecoverXXX族函数的切片数量都固定是k（门限值）
func recoverSignature(sigs []Signature, ids []ID) *Signature {
	//secret := big.NewInt(0) //组私钥
	k := len(sigs) //取得输出切片的大小，即门限值k
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		xs[i] = ids[i].GetBigInt() //把所有的id转化为big.Int，放到xs切片
	}
	// need len(ids) = k > 0
	sig := &Signature{}
	new_sig := &Signature{}
	for i := 0; i < k; i++ { //输入元素遍历
		// compute delta_i depending on ids only
		//为什么前面delta/num/den初始值是1，最后一个diff初始值是0？
		var delta, num, den, diff *big.Int = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
		for j := 0; j < k; j++ { //ID遍历
			if j != i { //不是自己
				num.Mul(num, xs[j])      //num值先乘上当前ID
				num.Mod(num, curveOrder) //然后对曲线域求模
				diff.Sub(xs[j], xs[i])   //diff=当前节点（内循环）-基节点（外循环）
				den.Mul(den, diff)       //den=den*diff
				den.Mod(den, curveOrder) //den对曲线域求模
			}
		}
		// delta = num / den
		den.ModInverse(den, curveOrder) //模逆
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)

		//最终需要的值是delta
		new_sig.value.Set(&sigs[i].value)
		new_sig.mul(delta)

		if i == 0 {
			sig.value.Set(&new_sig.value)
		} else {
			sig.add(new_sig)
		}
	}
	return sig
}

func (sig *Signature) add(sig1 *Signature) error {
	new_sig := &Signature{}
	new_sig.value.Set(&sig.value)
	sig.value.Add(&new_sig.value, &sig1.value)

	return nil
}

func (sig *Signature) mul(bi *big.Int) error {
	g1 := new(bn_curve.G1)
	g1.Set(&sig.value)
	sig.value.ScalarMult(g1, bi)
	return nil
}
