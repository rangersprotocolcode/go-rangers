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

package rlp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

type mapEntry struct {
	key   string
	value string
}

type MockMap struct {
	KeyList   []string
	ValueList []string
}

func TestMapToRLP(t *testing.T) {
	m := make(map[string]string)
	for i := 1; i < 100000; i++ {
		m[strconv.Itoa(i)] = "111"
	}

	jsonBytes, _ := json.Marshal(m)
	fmt.Printf("map json byte len:%d\n", len(jsonBytes))

	map1 := make([]mapEntry, 0)

	keyList := make([]string, 0)
	valueList := make([]string, 0)

	for k, v := range m {
		map1 = append(map1, mapEntry{k, v})

		keyList = append(keyList, k)
		valueList = append(valueList, v)
	}

	mapBytes1, err1 := EncodeToBytes(map1)
	if err1 != nil {
		fmt.Printf("err:%v", err1)
	}
	fmt.Printf("map1 len:%d, byte len:%d\n", len(map1), len(mapBytes1))

	m2 := &MockMap{KeyList: keyList, ValueList: valueList}
	mapBytes2, err2 := EncodeToBytes(m2)
	if err2 != nil {
		fmt.Printf("err:%v", err2)
	}
	fmt.Printf("map2 len:%d-%d,byte len:%d\n", len(m2.KeyList), len(m2.ValueList), len(mapBytes2))
}
