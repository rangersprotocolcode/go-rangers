// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package groupsig

import (
	"log"
	"math/big"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	bn_curve "com.tuntun.rocket/node/src/consensus/groupsig/bn256"
)

// Curve and Field order
var curveOrder = bn_curve.Order //曲线整数域
var fieldOrder = bn_curve.P
var bitLength = curveOrder.BitLen()

// Seckey -- represented by a big.Int modulo curveOrder
type Seckey struct {
	value BnInt
}

//由随机数构建私钥
func NewSeckeyFromRand(seed base.Rand) *Seckey {
	//把随机数转换成字节切片（小端模式）后构建私钥
	return newSeckeyFromByte(seed.Bytes())
}

//由大整数构建私钥
func NewSeckeyFromBigInt(b *big.Int) *Seckey {
	nb := &big.Int{}
	nb.Set(b)
	b.Mod(nb, curveOrder) //大整数在曲线域上求模

	sec := new(Seckey)
	sec.value.setBigInt(b)

	return sec
}

//把私钥转换成字节切片（小端模式）
func (sec Seckey) Serialize() []byte {
	return sec.value.serialize()
}

//由字节切片初始化私钥
func (sec *Seckey) Deserialize(b []byte) error {
	return sec.value.deserialize(b)
}

//把私钥转换成big.Int
func (sec Seckey) GetBigInt() (s *big.Int) {
	s = new(big.Int)
	s.Set(sec.value.getBigInt())
	return s
}

//返回十六进制字符串表示，带前缀
func (sec Seckey) GetHexString() string {
	return sec.value.getHexString()
}

//由带前缀的十六进制字符串转换
func (sec *Seckey) SetHexString(s string) error {
	return sec.value.setHexString(s)
}

//比较两个私钥是否相等
func (sec Seckey) IsEqual(rhs Seckey) bool {
	return sec.value.isEqual(&rhs.value)
}

func (sec Seckey) IsValid() bool {
	bi := sec.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0
}

func (sec *Seckey) ShortS() string {
	str := sec.GetHexString()
	return common.ShortHex12(str)
}

//私钥聚合函数
//萃取出签名私钥
func AggregateSeckeys(secs []Seckey) *Seckey {
	if len(secs) == 0 { //没有私钥要聚合
		log.Printf("AggregateSeckeys no secs")
		return nil
	}
	sec := new(Seckey) //创建一个新的私钥
	sec.value.setBigInt(secs[0].value.getBigInt())
	for i := 1; i < len(secs); i++ {
		sec.value.add(&secs[i].value)
	}

	x := new(big.Int)
	x.Set(sec.value.getBigInt())
	sec.value.setBigInt(x.Mod(x, curveOrder))
	return sec
}

//用多项式替换生成特定于某个ID的签名私钥分片
//msec : master私钥切片
//id : 获得该分片的id
//这里最开始是引用C语言版本的BLS的分片生成，后来改成了GO语言版本
func ShareSeckey(msec []Seckey, id ID) *Seckey {
	secret := big.NewInt(0)
	k := len(msec) - 1

	// evaluate polynomial f(x) with coefficients c0, ..., ck
	secret.Set(msec[k].GetBigInt()) //最后一个master key的big.Int值放到secret
	x := id.GetBigInt()             //取得id的big.Int值
	new_b := &big.Int{}

	for j := k - 1; j >= 0; j-- { //从master key切片的尾部-1往前遍历
		new_b.Set(secret)
		secret.Mul(new_b, x) //乘上id的big.Int值，每一遍都需要乘，所以是指数？

		new_b.Set(secret)
		secret.Add(new_b, msec[j].GetBigInt()) //加法

		new_b.Set(secret)
		secret.Mod(new_b, curveOrder) //曲线域求模
	}

	return NewSeckeyFromBigInt(secret) //生成签名私钥
}

//由字节切片（小端模式）构建私钥
func newSeckeyFromByte(b []byte) *Seckey {
	sec := new(Seckey)
	err := sec.Deserialize(b[:32])
	if err != nil {
		log.Printf("NewSeckeyFromByte %s\n", err)
		return nil
	}

	sec.value.mod()
	return sec
}

////由master私钥切片和TAS地址生成针对该地址的签名私钥分片
//func ShareSeckeyByAddr(msec []Seckey, addr common.Address) *Seckey {
//	id := NewIDFromAddress(addr)
//	if id == nil {
//		log.Printf("ShareSeckeyByAddr bad addr=%s\n", addr)
//		return nil
//	}
//	return ShareSeckey(msec, *id)
//}
//
////由master私钥切片和整数i生成签名私钥分片
//func ShareSeckeyByInt(msec []Seckey, i int) *Seckey {
//	return ShareSeckey(msec, *NewIDFromInt64(int64(i)))
//}
//
////由master私钥切片和整数id，生成id+1者的签名私钥分片
//func ShareSeckeyByMembershipNumber(msec []Seckey, id int) *Seckey {
//	return ShareSeckey(msec, *NewIDFromInt64(int64(id + 1)))
//}

////MAP(地址->私钥)
//type SeckeyMap map[common.Address]Seckey
//
//// SeckeyMapInt -- a map from addresses to Seckey
////map(地址->私钥)
//type SeckeyMapInt map[int]Seckey
//
//type SeckeyMapID map[string]Seckey
