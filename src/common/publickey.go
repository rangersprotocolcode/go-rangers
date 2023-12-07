// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"com.tuntun.rangers/node/src/common/ecies"
	"com.tuntun.rangers/node/src/common/secp256k1"
	"com.tuntun.rangers/node/src/common/sha3"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"io"
)

type PublicKey struct {
	PubKey ecdsa.PublicKey
}

func (pk PublicKey) Verify(hash []byte, s *Sign) bool {
	return secp256k1.VerifySignature(pk.ToBytes(), hash, s.Bytes()[:64])
}

func (pk PublicKey) GetAddress() Address {
	addrBuf := pk.GetID()
	return BytesToAddress(addrBuf[:])
}

func (pk PublicKey) GetID() []byte {
	x := pk.PubKey.X.Bytes()
	y := pk.PubKey.Y.Bytes()

	digest := make([]byte, 64)
	copy(digest[32-len(x):], x)
	copy(digest[64-len(y):], y)

	d := sha3.NewKeccak256()
	d.Write(digest)
	hash := d.Sum(nil)
	return hash
}

func (pk PublicKey) ToBytes() []byte {
	buf := elliptic.Marshal(pk.PubKey.Curve, pk.PubKey.X, pk.PubKey.Y)
	return buf
}

func BytesToPublicKey(data []byte) (pk *PublicKey) {
	pk = new(PublicKey)
	pk.PubKey.Curve = getDefaultCurve()
	x, y := elliptic.Unmarshal(pk.PubKey.Curve, data)
	if x == nil || y == nil {
		panic("unmarshal public key failed.")
	}
	pk.PubKey.X = x
	pk.PubKey.Y = y
	return
}

func (pk PublicKey) GetHexString() string {
	buf := pk.ToBytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

func (pk *PublicKey) Encrypt(rand io.Reader, msg []byte) ([]byte, error) {
	return Encrypt(rand, pk, msg)
}

func HexStringToPubKey(s string) (pk *PublicKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	pk = BytesToPublicKey(buf)
	return
}

func Encrypt(rand io.Reader, pub *PublicKey, msg []byte) (ct []byte, err error) {
	pubECIES := ecies.ImportECDSAPublic(&pub.PubKey)
	return ecies.Encrypt(rand, pubECIES, msg, nil, nil)
}
