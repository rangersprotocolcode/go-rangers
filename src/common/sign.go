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
	"com.tuntun.rocket/node/src/utility"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"com.tuntun.rocket/node/src/common/secp256k1"
)

type Hasher interface {
	GenHash() Hash
}

type Sign struct {
	r     big.Int
	s     big.Int
	recid byte
}

//数据签名结构 for message casting
type SignData struct {
	DataHash Hash   //哈希值
	DataSign Sign   //签名
	Id       string //用户ID
}

func NewSignData(privateKey PrivateKey, id string, hasher Hasher) SignData {
	result := SignData{}

	hash := hasher.GenHash()
	result.DataHash = hash
	result.Id = id
	result.DataSign = privateKey.Sign(hash.Bytes())
	return result
}

func (signData SignData) ValidateSign(hasher Hasher) error {
	if signData.DataHash != hasher.GenHash() {
		return errors.New(fmt.Sprintf("Invalid hash:except %s,real %s", signData.DataHash.String(), hasher.GenHash().String()))
	}
	pubkey, err := signData.DataSign.RecoverPubkey(signData.DataHash.Bytes())
	if err != nil {
		return err
	}

	if !pubkey.Verify(signData.DataHash.Bytes(), &signData.DataSign) {
		return errors.New("sign verify failed")
	}
	return nil
}

//签名构造函数
func (s *Sign) Set(_r, _s *big.Int, recid int) {
	s.r = *_r
	s.s = *_s
	s.recid = byte(recid)
}

//检查签名是否有效
func (s Sign) Valid() bool {
	return s.r.BitLen() != 0 && s.s.BitLen() != 0 && s.recid < 4
}

func (s Sign) GetR() big.Int {
	return s.r
}

func (s Sign) GetS() big.Int {
	return s.s
}

//Sign必须65 bytes
func (s Sign) Bytes() []byte {
	rb := s.r.Bytes()
	sb := s.s.Bytes()
	r := make([]byte, SignLength)
	copy(r[32-len(rb):32], rb)
	copy(r[64-len(sb):64], sb)
	r[64] = s.recid
	return r
}

//Sign必须65 bytes
func BytesToSign(b []byte) *Sign {
	if len(b) == 65 {
		var r, s big.Int
		br := b[:32]
		r = *r.SetBytes(br)

		sr := b[32:64]
		s = *s.SetBytes(sr)

		recid := b[64]
		return &Sign{r, s, recid}
	} else {
		//这里组签名暂不处理
		return nil
	}
}

func (s Sign) GetHexString() string {
	buf := s.Bytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

//导入函数
func HexStringToSign(s string) (si *Sign) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	si = BytesToSign(buf)
	return si
}

func (s Sign) RecoverPubkey(msg []byte) (pk *PublicKey, err error) {
	pubkey, err := secp256k1.RecoverPubkey(msg, s.Bytes())
	if err != nil {
		return nil, err
	}
	pk = BytesToPublicKey(pubkey)
	return
}


// MarshalText returns the hex representation of h.
func (s Sign) MarshalText() ([]byte, error) {
	return utility.Bytes(s.Bytes()).MarshalText()
}

func (s *Sign) UnmarshalText(input []byte) error {
	result := BytesToSign(FromHex(string(input)))

	s.r = result.r
	s.s = result.s
	s.recid = result.recid

	return nil
}