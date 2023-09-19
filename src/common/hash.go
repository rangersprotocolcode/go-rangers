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
	"crypto/sha256"
	"hash"
	"sync"
)

var hasherPool = sync.Pool{
	New: func() interface{} {
		return sha256.New()
	},
}

// 计算sha256
func Sha256(blockByte []byte) []byte {
	hasher := hasherPool.Get().(hash.Hash)
	hasher.Reset()
	defer hasherPool.Put(hasher)

	hasher.Write(blockByte)
	return hasher.Sum(nil)

}

type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}
