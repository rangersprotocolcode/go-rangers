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

package base

import (
	"hash"

	"com.tuntun.rocket/node/src/common"

	"golang.org/x/crypto/sha3"
)

// Hash helpers

// 生成多维字节数组的SHA3_256位哈希
func HashBytes(b ...[]byte) hash.Hash {
	d := sha3.New256()
	for _, bi := range b {
		d.Write(bi)
	}
	return d
}

// 生成数据的256位common.Hash
func Data2CommonHash(data []byte) common.Hash {
	var h common.Hash
	sha3_hash := sha3.Sum256(data)
	if len(sha3_hash) == common.HashLength {
		copy(h[:], sha3_hash[:])
	} else {
		panic("Data2Hash failed, size error.")
	}
	return h
}
