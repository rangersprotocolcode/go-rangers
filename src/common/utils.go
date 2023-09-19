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
	"fmt"

	lru "github.com/hashicorp/golang-lru"
)

// CreateLRUCache MustNewLRUCache creates a new lru cache.
// Caution: if failed, the function will cause panic
func CreateLRUCache(size int) *lru.Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}

func ShortHex12(hex string) string {
	s := len(hex)
	if s < 12 {
		return hex
	}
	return hex[0:6] + "-" + hex[s-6:]
}
