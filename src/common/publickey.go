package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"x/src/common/secp256k1"
)

//用户公钥
type PublicKey struct {
	PubKey ecdsa.PublicKey
}

//公钥验证函数
func (pk PublicKey) Verify(hash []byte, s *Sign) bool {
	//return ecdsa.Verify(&pk.PubKey, hash, &s.r, &s.s)
	return secp256k1.VerifySignature(pk.ToBytes(), hash, s.Bytes()[:64])
}

//由公钥萃取地址函数
func (pk PublicKey) GetAddress() Address {
	x := pk.PubKey.X.Bytes()
	y := pk.PubKey.Y.Bytes()
	x = append(x, y...)

	addr_buf := sha3.Sum256(x)
	Addr := BytesToAddress(addr_buf[:])
	return Addr
}

//由公钥萃取矿工ID
func (pk PublicKey) GetID() [32]byte {
	x := pk.PubKey.X.Bytes()
	y := pk.PubKey.Y.Bytes()
	x = append(x, y...)

	addr_buf := sha3.Sum256(x)

	return addr_buf
}

//把公钥转换成字节切片
func (pk PublicKey) ToBytes() []byte {
	buf := elliptic.Marshal(pk.PubKey.Curve, pk.PubKey.X, pk.PubKey.Y)
	//fmt.Printf("end pub key marshal, len=%v, data=%v\n", len(buf), buf)
	return buf
}

//从字节切片转换到公钥
func BytesToPublicKey(data []byte) (pk *PublicKey) {
	pk = new(PublicKey)
	pk.PubKey.Curve = getDefaultCurve()
	//fmt.Printf("begin pub key unmarshal, len=%v, data=%v.\n", len(data), data)
	x, y := elliptic.Unmarshal(pk.PubKey.Curve, data)
	if x == nil || y == nil {
		panic("unmarshal public key failed.")
	}
	pk.PubKey.X = x
	pk.PubKey.Y = y
	return
}

//导出函数
func (pk PublicKey) GetHexString() string {
	buf := pk.ToBytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

//导入函数
func HexStringToPubKey(s string) (pk *PublicKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	pk = BytesToPublicKey(buf)
	return
}
