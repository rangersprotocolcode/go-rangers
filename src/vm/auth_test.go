// Copyright 2020 The RangersProtocol Authors
// This file is part of the RangersProtocol library.
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

package vm

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/common/sha3"
	crypto "com.tuntun.rangers/node/src/eth_crypto"
	"fmt"
	"math/big"
	"testing"
)

func TestAuth(t *testing.T) {
	//msgList := []string{"0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000010b7680c72563c3100f1a539b8c8e0325db1863f00000000000000000000000000000000000000000000000000000000000027100000000000000000000000000000000000000000000000000000000000000064000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000246057361d000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000", "0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000010b7680c72563c3100f1a539b8c8e0325db1863f0000000000000000000000000000000000000000000000000000000000004e200000000000000000000000000000000000000000000000000000000000000064000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000246057361d000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000"}
	//bytes := make([]byte, 0)
	//for _, msg := range msgList {
	//	d := sha3.NewKeccak256()
	//	d.Write(common.FromHex(msg))
	//	txHash := d.Sum(nil)
	//	bytes = append(bytes, txHash[:]...)
	//}
	msg := common.FromHex("0x03000000000000000000000000000000000000000000000000000000000000251c000000000000000000000000f800eddcdbd86fc46df366526f709bef33bd3d45cf6324e01cd6b78a886cc3aaae0a471451b2d6baa811b011ab799f9fcd5a503a")
	d := sha3.NewKeccak256()
	d.Write(msg)
	commitByte := d.Sum(nil)
	fmt.Printf("commit:%s\n", common.ToHex(commitByte))
	contractAddress := common.HexToAddress("0xb745a6937599dd4cd4cb1fd6ed25743d42b4e3f4")

	var commitHash [32]byte
	copy(commitHash[:], commitByte[:])
	hash := calAuthHash(big.NewInt(9500), contractAddress, commitHash)
	fmt.Printf("hash:%s\n", common.ToHex(hash))

	hash = common.FromHex("0x957bf39fd2c8ffe4a6a2d4b6389e8eae5b48ae6be727c6046cf310ccf1c2bb22")
	privateKeyStr := ""
	var privateKey = common.HexStringToSecKey(privateKeyStr)
	sign := privateKey.Sign(hash)
	r := sign.GetR()
	s := sign.GetS()
	fmt.Printf("%v,%v,\n", r.String(), s.String())
	fmt.Printf("%v\n", sign.Bytes())
}

func TestAuth1(t *testing.T) {
	contractAddress := common.HexToAddress("0xf800edDCdbD86FC46dF366526f709bef33Bd3D45")

	var commitByte = common.FromHex("0xcf6324e01cd6b78a886cc3aaae0a471451b2d6baa811b011ab799f9fcd5a503a")
	var commitHash [32]byte
	copy(commitHash[:], commitByte[:])
	hash := calAuthHash(big.NewInt(9500), contractAddress, commitHash)
	fmt.Printf("hash:%s\n", common.ToHex(hash))

	msg1 := common.FromHex("0x03000000000000251c000000000000000000000000000000000000000000000000000000000000000000000000f800eddcdbd86fc46df366526f709bef33bd3d45cf6324e01cd6b78a886cc3aaae0a471451b2d6baa811b011ab799f9fcd5a503a")
	hash1 := crypto.Keccak256(msg1)
	fmt.Printf("hash1:%s\n", common.ToHex(hash1))

	msg2 := common.FromHex("0x03000000000000251c000000000000000000000000000000000000000000000000000000000000000000000000f800eddcdbd86fc46df366526f709bef33bd3d45cf6324e01cd6b78a886cc3aaae0a471451b2d6baa811b011ab799f9fcd5a503a")
	hash2 := crypto.Keccak256(msg2)
	fmt.Printf("hash2:%s\n", common.ToHex(hash2))
}
