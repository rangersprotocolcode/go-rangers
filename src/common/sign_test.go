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

package common

import (
	"bytes"
	"com.tuntun.rocket/node/src/common/secp256k1"
	"fmt"
	"testing"

	"crypto/sha256"
	"strconv"
)

func TestPrivateKey(test *testing.T) {
	fmt.Printf("begin TestPrivateKey...\n")
	sk := GenerateKey("")
	str := sk.GetHexString()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)
	new_sk := HexStringToSecKey(str)
	new_str := new_sk.GetHexString()
	fmt.Printf("import sec key and export again, len=%v, data=%v.\n", len(new_str), new_str)
	fmt.Printf("end TestPrivateKey.\n")
}

func TestPublicKey(test *testing.T) {
	fmt.Printf("begin TestPublicKey...\n")
	sk := GenerateKey("")
	pk := sk.GetPubKey()
	str := pk.GetHexString()
	fmt.Printf("pub key export, len=%v, data=%v.\n", len(str), str)
	new_pk := HexStringToPubKey(str)
	new_str := new_pk.GetHexString()
	fmt.Printf("import pub key and export again, len=%v, data=%v.\n", len(new_str), new_str)

	fmt.Printf("\nbegin test address...\n")
	a := pk.GetAddress()
	str = a.GetHexString()
	fmt.Printf("address export, len=%v, data=%v.\n", len(str), str)
	new_a := HexStringToAddress(str)
	new_str = new_a.GetHexString()
	fmt.Printf("import address and export again, len=%v, data=%v.\n", len(new_str), new_str)

	fmt.Printf("end TestPublicKey.\n")
}

func TestSign(test *testing.T) {
	fmt.Printf("begin TestSign...\n")
	plain_txt := "My name is thiefox."
	buf := []byte(plain_txt)
	sha3_hash := sha256.Sum256(buf)

	pri_k := GenerateKey("")
	pub_k := pri_k.GetPubKey()

	pub_buf := pub_k.ToBytes() //测试公钥到字节切片的转换
	pub_k = *BytesToPublicKey(pub_buf)

	sha3_si := pri_k.Sign(sha3_hash[:])
	{
		buf_r := sha3_si.r.Bytes()
		buf_s := sha3_si.s.Bytes()
		fmt.Printf("sha3 sign, r len = %v, s len = %v.\n", len(buf_r), len(buf_s))
	}
	success := pub_k.Verify(sha3_hash[:], &sha3_si)
	fmt.Printf("sha3 sign verify result=%v.\n", success)
	fmt.Printf("end TestSign.\n")
}

func TestSignBytes(test *testing.T) {
	plain_txt := "dafaefaewfef"
	buf := []byte(plain_txt)
	sha3_hash := Sha256(buf)
	s := BytesToHash(sha3_hash).Hex()
	fmt.Printf("hash:%s\n", s)

	pri_k := GenerateKey("")
	sign := pri_k.Sign(sha3_hash[:]) //私钥签名

	address := pri_k.GetPubKey().GetAddress()
	fmt.Printf("Address:%s\n", address.GetHexString())
	//测试签名十六进制转换
	h := sign.GetHexString() //签名十六进制表示
	fmt.Println(h)

	si := HexStringToSign(h) //从十六进制恢复出签名
	fmt.Println(si.Bytes())  //签名打印
	fmt.Println(sign.Bytes())

	sign_bytes := sign.Bytes()
	sign_r := BytesToSign(sign_bytes)
	fmt.Println(sign_r.GetHexString())
	if h != sign_r.GetHexString() {
		fmt.Println("sign dismatch!", h, sign_r.GetHexString())
	}

}

func TestRecoverPubkey(test *testing.T) {
	fmt.Printf("begin TestRecoverPubkey...\n")
	plain_txt := "Sign Recover Pubkey tesing."
	buf := []byte(plain_txt)
	sha3_hash := sha256.Sum256(buf)

	sk := GenerateKey("")
	sign := sk.Sign(sha3_hash[:])

	pk, err := sign.RecoverPubkey(sha3_hash[:])
	if err == nil {
		if !bytes.Equal(pk.ToBytes(), sk.GetPubKey().ToBytes()) {
			fmt.Printf("original pk = %v\n", sk.GetPubKey().ToBytes())
			fmt.Printf("recover pk = %v\n", pk)
		}
	}
	fmt.Printf("revovered pubkey:%s\n", pk.GetHexString())
	fmt.Printf("expected pubkey:%s\n", sk.GetPubKey().GetHexString())
	fmt.Printf("end TestRecoverPubkey.\n")
}

func BenchmarkSign(b *testing.B) {
	msg := []byte("This is TASchain achates' testing message")
	sk := GenerateKey("")
	sha3_hash := sha256.Sum256(msg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Sign(sha3_hash[:]) //私钥签名
	}
}

func BenchmarkVerify(b *testing.B) {
	msg := []byte("This is TASchain achates' testing message")
	sk := GenerateKey("")
	pk := sk.GetPubKey()
	sha3_hash := sha256.Sum256(msg)
	sign := sk.Sign(sha3_hash[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.Verify(sha3_hash[:], &sign)
	}
}

func BenchmarkRecover(b *testing.B) {
	msg := []byte("This is TASchain achates' testing message")
	sk := GenerateKey("")
	sha3_hash := sha256.Sum256(msg)
	sign := sk.Sign(sha3_hash[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sign.RecoverPubkey(sha3_hash[:])
	}
}

func TestGenerateKey(t *testing.T) {
	s := "1111345111111111111111111111111111111111"
	sk := GenerateKey(s)
	t.Logf(sk.GetHexString())

	sk2 := GenerateKey(s)
	t.Logf(sk2.GetHexString())

	sk3 := GenerateKey(s)
	t.Logf(sk3.GetHexString())
}

func TestHashFromBytes(t *testing.T) {
	s := "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"
	hash := HexToHash(s)

	fmt.Println(hash)
	fmt.Println(len(hash))
}

func TestAddress(t *testing.T) {
	s := "0xb253748a50c78ead4c472a8912ba614f12e9d94a"
	hex := FromHex(s)
	fmt.Printf("from hex %v", hex)

	addr := HexToAddress(s)
	fmt.Printf("addr %v", addr)
}

func TestStrToBigInt(t *testing.T) {
	s := "5200000000000000000000000000000000000000000000000000000.32242"
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		fmt.Printf("error:%s", err.Error())
	}
	fmt.Printf("%v\n", f)
	i := int64(f * 1000000000)
	fmt.Printf("%v\n", i)
}

func TestKey(t *testing.T) {
	privateKey := GenerateKey("")
	publicKey := privateKey.GetPubKey()
	id := publicKey.GetID()
	address := publicKey.GetAddress()
	fmt.Printf("Private key:%s\n", privateKey.GetHexString())
	fmt.Printf("Public key:%s\n", privateKey.GetPubKey().GetHexString())
	fmt.Printf("Address:%s\n", address.String())
	fmt.Printf("Id:%s\n", ToHex(id[:]))
}

func TestKeyByHex(t *testing.T) {
	privateKey := HexStringToSecKey("0xd7f5d173593eff81a50f7d8ea345bbc543ad8e356e75975e87114438c8f4eaf4")
	publicKey := privateKey.GetPubKey()
	id := publicKey.GetID()
	address := publicKey.GetAddress()
	fmt.Printf("Private key:%s\n", privateKey.GetHexString())
	fmt.Printf("Public key:%s\n", publicKey.GetHexString())
	fmt.Printf("Address:%s\n", address.String())
	fmt.Printf("Id:%s\n", ToHex(id[:]))
}

func TestRecoverPubkeyFromMsg(t *testing.T) {
	sig := FromHex("0x68fb6c58fd7cfbce99457414d774eb572bd1f13dc725bb2372de42bcf356d687793f2784c5be7181b4741befa817d8657074c1811886d7d0ce1d1cf3314e7ef61b")
	msg := FromHex("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01b3f365018")
	pubkeyBytes, err := secp256k1.RecoverPubkey(msg, sig)
	if err != nil {
		t.Errorf("recover error: %s", err)
	}
	pubkey := BytesToPublicKey(pubkeyBytes)
	fmt.Println(pubkey.GetHexString())
	fmt.Println(pubkey.GetAddress().String())
}

func TestSign1(t *testing.T) {
	hash := FromHex("0xad7c3d5da478dd1d01c155f5c48e495550d9145445017759065ebc5b7165d0de")
	privateKeyStr := "0x0db05c85c5d4685a9fad2d5581f24ee4e42c3b57b92ff0f00ef287532b7da58a"
	var privateKey = HexStringToSecKey(privateKeyStr)
	sign := privateKey.Sign(hash)
	//r := sign.GetR()
	//s := sign.GetS()
	fmt.Printf("%s\n", privateKey.GetPubKey().GetAddress().String())
	fmt.Printf("%s\n", sign.GetHexString())
}
func TestRecoverPubkeyFromMsg1(t *testing.T) {
	sig := FromHex("0xe08a43675b3e8f933cb51172f27aa0b5524ab8e51f90a465e6ab01fe881579144066187359addf9788257ba527461fe0c5ff1fd9499a7cf3f1b98d75beb9342d1c")
	msg := FromHex("0xad7c3d5da478dd1d01c155f5c48e495550d9145445017759065ebc5b7165d0de")
	pubkeyBytes, err := secp256k1.RecoverPubkey(msg, sig)
	if err != nil {
		t.Errorf("recover error: %s", err)
	}
	pubkey := BytesToPublicKey(pubkeyBytes)
	fmt.Println(pubkey.GetHexString())
	fmt.Println(pubkey.GetAddress().String())
}
