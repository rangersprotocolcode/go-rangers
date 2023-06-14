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

package vm

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/common/sha3"
	crypto "com.tuntun.rocket/node/src/eth_crypto"
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
	//todo
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

//func TestABI(t *testing.T) {
//	uintTyp, err := abi.NewType("uint256")
//	if err != nil {
//		panic(err)
//	}
//	uintEncoded, err := uintTyp.Encode(1000)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("uint256:%s\n",common.ToHex(uintEncoded))
//
//	address := common.HexToAddress("0x454dfd1a16d1c6dc33fd4f045a4b7a2b2898d384")
//	addressTyp, err := abi.NewType("address")
//	if err != nil {
//		panic(err)
//	}
//	addressEncoded, err := addressTyp.Encode(address)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("address:%s\n",common.ToHex(addressEncoded))
//
//	bytesInfo := common.FromHex("0x6057361d0000000000000000000000000000000000000000000000000000000000000002")
//	bytesTyp, err := abi.NewType("bytes")
//	if err != nil {
//		panic(err)
//	}
//	bytesEncoded, err := bytesTyp.Encode(bytesInfo)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("bytes:%s\n",common.ToHex(bytesEncoded))
//
//	strInfo := "0x6057361d0000000000000000000000000000000000000000000000000000000000000002"
//	strTyp, err := abi.NewType("string")
//	if err != nil {
//		panic(err)
//	}
//	strEncoded, err := strTyp.Encode(strInfo)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("string:%s\n",common.ToHex(strEncoded))
//
//	typ,_:= abi.NewType("tuple(address to, uint256 value, uint256 gaslimit, uint256 nonce,bytes data)")
//
//	type Tx struct {
//		To common.Address
//		Value *big.Int
//		Gaslimit *big.Int
//		Nonce *big.Int
//		Data []byte
//	}
//	obj := &Tx{
//		To:common.HexToAddress("0x454dfd1a16d1c6dc33fd4f045a4b7a2b2898d384"),
//		Value: big.NewInt(1),
//		Gaslimit: big.NewInt(100),
//		Nonce: big.NewInt(2),
//		Data: common.FromHex("0x6057361d0000000000000000000000000000000000000000000000000000000000000002"),
//	}
//
//	// Encode
//	encodedTx, err := typ.Encode(obj)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("tx:%s\n",common.ToHex(encodedTx))
//}
