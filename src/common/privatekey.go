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
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
)

type PrivateKey struct {
	PrivKey ecdsa.PrivateKey
}

func (pk PrivateKey) Sign(hash []byte) Sign {
	var sign Sign
	sig, err := secp256k1.Sign(hash, pk.PrivKey.D.Bytes())
	if err == nil {
		if len(sig) != 65 {
			fmt.Printf("secp256k1 sign wrong! hash = %v\n", hash)
		}
		sign = *BytesToSign(sig)
	} else {
		panic(fmt.Sprintf("Sign Failed, reason : %v.\n", err.Error()))
	}

	return sign
}

func GenerateKey(s string) PrivateKey {
	var r io.Reader
	if len(s) > 0 {
		r = strings.NewReader(s)
	} else {
		r = rand.Reader
	}
	var pk PrivateKey
	_pk, err := ecdsa.GenerateKey(getDefaultCurve(), r)
	if err == nil {
		pk.PrivKey = *_pk
	} else {
		panic(fmt.Sprintf("GenKey Failed, reason : %v.\n", err.Error()))
	}
	return pk
}

func (pk *PrivateKey) GetPubKey() PublicKey {
	var pubk PublicKey
	pubk.PubKey = pk.PrivKey.PublicKey
	return pubk
}

func (pk *PrivateKey) GetHexString() string {
	buf := pk.PrivKey.D.Bytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

func HexStringToSecKey(s string) (sk *PrivateKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	sk = new(PrivateKey)
	sk.PrivKey.D = new(big.Int).SetBytes(FromHex(s))
	sk.PrivKey.PublicKey.Curve = getDefaultCurve()
	sk.PrivKey.PublicKey.X, sk.PrivKey.PublicKey.Y = getDefaultCurve().ScalarBaseMult(sk.PrivKey.D.Bytes())
	return
}

func (pk *PrivateKey) Decrypt(rand io.Reader, ct []byte) (m []byte, err error) {
	prv := ecies.ImportECDSA(&pk.PrivKey)
	return prv.Decrypt(rand, ct, nil, nil)
}

func BytesToSecKey(data []byte) (sk *PrivateKey) {
	if len(data) < SecKeyLength {
		return nil
	}
	sk = new(PrivateKey)

	buf_pub := data[:PubKeyLength]
	buf_d := data[PubKeyLength:]
	sk.PrivKey.PublicKey = BytesToPublicKey(buf_pub).PubKey
	sk.PrivKey.D = new(big.Int).SetBytes(buf_d)
	if sk.PrivKey.X != nil && sk.PrivKey.Y != nil && sk.PrivKey.D != nil {
		return sk
	}
	return nil
}
