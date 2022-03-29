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

package core

import (
	"com.tuntun.rocket/node/src/middleware/db"
	"fmt"
	"testing"
)

func TestBlockChain_GenerateHeightKey(t *testing.T) {
	result := generateHeightKey(10)
	fmt.Println(len(result))
	fmt.Printf("%v", result)
}

func TestDB(t *testing.T) {
	db1, _ := db.NewDatabase(blockForkDBPrefix)
	db1.Put([]byte("1"), []byte("1"))
	db1.Put([]byte("2"), []byte("2"))

	db1.Delete([]byte("1"))

	iterator1 := db1.NewIterator()
	for iterator1.Next() {
		key := iterator1.Key()
		realKey := key[9:]
		fmt.Printf("key:%v,realkey:%v\n", key, realKey)
		db1.Delete(realKey)
		//db1.Delete(key)
	}
	//for iterator1.Next() {
	//	key := iterator1.Key()
	//	fmt.Printf("key2:%v\n", key)
	//	//append(keyList, key)
	//}
	//
	//db2, _ := db.NewDatabase(blockForkDBPrefix)
	//iterator2 := db2.NewIterator()
	//for iterator2.Next() {
	//	key := iterator2.Key()
	//	realKey := key[9:]
	//	fmt.Printf("second key:%v\n", realKey)
	//}
}
