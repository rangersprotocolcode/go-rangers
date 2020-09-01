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
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/sha3"
	"math/big"
	"testing"
)

func TestSha256(t *testing.T) {
	hash := sha3.Sum256([]byte{})
	fmt.Println(Bytes2Hex(hash[:]))

	hash2 := sha3.Sum256([]byte("test"))
	fmt.Println(Bytes2Hex(hash2[:]))
}

func TestID(t *testing.T) {
	privateKeyStr := "0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea  "
	privateKeyBuf, _ := hex.DecodeString(privateKeyStr[len(PREFIX):])
	fmt.Printf("privateKeyBuf len:%d\n", len(privateKeyBuf))
	var privateKey PrivateKey
	privateKey.PrivKey.PublicKey.Curve = getDefaultCurve()
	privateKey.PrivKey.D = new(big.Int).SetBytes(privateKeyBuf)
	privateKey.PrivKey.PublicKey.X, privateKey.PrivKey.PublicKey.Y = getDefaultCurve().ScalarBaseMult(privateKey.PrivKey.D.Bytes())

	pubkey := privateKey.GetPubKey()
	address := pubkey.GetAddress()
	fmt.Printf("SK:%v\n", privateKey.GetHexString())
	fmt.Printf("pubkey:%v\n", pubkey.GetHexString())
	fmt.Printf("address:%v\n", address.GetHexString())
}
