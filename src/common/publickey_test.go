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
	addr := BigToAddress(big.NewInt(5))
	fmt.Println(addr.String())

	var h Hash
	h = sha3.Sum256(addr[:])
	fmt.Println(h.String())
}

func TestID(t *testing.T) {
	privateKeyStr := "0x99a01aedffd712ca2471e99fbc95008e873ee8d93d0ee9b5dd90cb1d9547ddb1"
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
