package groupsig

import (
	"math/big"

	"x/src/common"
	"golang.org/x/crypto/sha3"
	"log"
)

const ID_LENGTH = 32 //ID字节长度(256位，同私钥长度)

// ID -- id for secret sharing, represented by big.Int
//秘密共享的ID，64位int，共256位
type ID struct {
	value BnInt
}

//由公钥构建ID，公钥->（缩小到160位）地址->（放大到256/384位）ID
func NewIDFromPubkey(pk Pubkey) *ID {
	h := sha3.Sum256(pk.Serialize()) //取得公钥的SHA3 256位哈希
	bi := new(big.Int).SetBytes(h[:])
	return newIDFromBigInt(bi)
}

//DeserializeId
func DeserializeID(bs []byte) ID {
	var id ID
	if err := id.Deserialize(bs); err != nil {
		return ID{}
	}
	return id
}

//把ID转换到十六进制字符串
func (id ID) GetHexString() string {
	bs := id.Serialize()
	return common.ToHex(bs)
}

//把十六进制字符串转换到ID
func (id *ID) SetHexString(s string) error {
	return id.value.setHexString(s)
}

//把ID转换到big.Int
func (id ID) GetBigInt() *big.Int {
	x := new(big.Int)
	x.Set(id.value.getBigInt())
	return x
}

//把big.Int转换到ID
func (id *ID) SetBigInt(b *big.Int) error {
	id.value.setBigInt(b)
	return nil
}

//把ID转换到字节切片（小端模式）
func (id ID) Serialize() []byte {
	idBytes := id.value.serialize()
	if len(idBytes) == ID_LENGTH {
		return idBytes
	}
	if len(idBytes) > ID_LENGTH {
		panic("ID Serialize error: ID bytes is more than IDLENGTH")
	}
	buff := make([]byte, ID_LENGTH)
	copy(buff[ID_LENGTH-len(idBytes):ID_LENGTH], idBytes)
	return buff
}

//把字节切片转换到ID
func (id *ID) Deserialize(b []byte) error {
	return id.value.deserialize(b)
}

//判断2个ID是否相同
func (id ID) IsEqual(rhs ID) bool {
	return id.value.isEqual(&rhs.value)
}

func (id ID) IsValid() bool {
	bi := id.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0

}

func (id ID) ShortS() string {
	str := id.GetHexString()
	return common.ShortHex12(str)
}

func (id ID) ToAddress() common.Address {
	return common.BytesToAddress(id.Serialize())
}

//由big.Int创建ID
func newIDFromBigInt(b *big.Int) *ID {
	id := new(ID)
	err := id.value.setBigInt(b) //bn_curve C库函数
	if err != nil {
		log.Printf("NewIDFromBigInt %s\n", err)
		return nil
	}
	return id
}

////从160位地址创建（FP254曲线256位或FP382曲线384位的）ID
////bn_curve.ID和common.Address不支持双向来回互转，因为两者的值域不一样（384位和160位），互转就会生成不同的值。
//func NewIDFromAddress(addr common.Address) *ID {
//	return NewIDFromBigInt(addr.BigInteger())
//}
