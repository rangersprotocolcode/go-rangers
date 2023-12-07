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

package db

import (
	"fmt"
	"testing"
)

func TestCreateLDB(t *testing.T) {
	ldb, err := NewDatabase("testldb")
	if err != nil {
		fmt.Printf("error to create ldb : %s\n", "testldb")
		return
	}

	err = ldb.Put([]byte("testkey"), []byte("testvalue"))
	if err != nil {
		fmt.Printf("failed to put key in testldb\n")
	}

	result, err := ldb.Get([]byte("testkey"))
	if err != nil {
		fmt.Printf("failed to get key in testldb\n")
	}
	if result != nil {
		fmt.Printf("get key : testkey, value: %s \n", result)
	}

	exist, err := ldb.Has([]byte("testkey"))
	if err != nil {
		fmt.Printf("error to check key : %s\n", "testkey")

	}
	if exist {
		fmt.Printf("get key : %s\n", "testkey")
	}

	err = ldb.Delete([]byte("testkey"))
	if err != nil {
		fmt.Printf("error to delete key : %s\n", "testkey")

	}

	result, err = ldb.Get([]byte("testkey"))
	if err != nil {
		fmt.Printf("failed to get key in testldb\n")
	}
	if result != nil {
		fmt.Printf("get key : testkey, value: %s \n", result)
	} else {
		fmt.Printf("get key : testkey, value: null")
	}

	if ldb != nil {
		ldb.Close()
	}

}

func TestLDBScan(t *testing.T) {
	//ldb, _ := NewLDBDatabase("/Users/Kaede/TasProject/test1",1,1)
	ldb, _ := NewDatabase("testldb")
	key1 := []byte{0, 1, 1}
	key2 := []byte{0, 1, 2}
	key3 := []byte{0, 2, 1}
	ldb.Put(key1, key1)
	ldb.Put(key2, key2)
	ldb.Put(key3, key3)
	iter := ldb.NewIteratorWithPrefix([]byte{0, 1})
	for iter.Next() {
		fmt.Println(iter.Value())
	}
}

func TestLRUMemDatabase(t *testing.T) {
	mem, _ := NewLRUMemDatabase(10)
	for i := (byte)(0); i < 11; i++ {
		mem.Put([]byte{i}, []byte{i})
	}
	data, _ := mem.Get([]byte{0})
	if data != nil {
		t.Errorf("expected value nil")
	}
	data, _ = mem.Get([]byte{10})
	if data == nil {
		t.Errorf("expected value not nil")
	}
	data, _ = mem.Get([]byte{5})
	if data == nil {
		t.Errorf("expected value not nil")
	}
	mem.Delete([]byte{5})
	data, _ = mem.Get([]byte{5})
	if data != nil {
		t.Errorf("expected value nil")
	}
}

func TestClearLDB(t *testing.T) {
	// 创建ldb实例
	ldb, err := NewDatabase("testldb")
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	err = ldb.Put([]byte("testkey"), []byte("testvalue"))
	if err != nil {
		t.Fatalf("failed to put key in testldb\n")
	}

	if err != nil {
		t.Fatalf("error to clear ldb : %s\n", "testldb")
		return
	}

}

func TestLDB(t *testing.T) {

	ldb := newLDB()

	byte, err := ldb.Get([]byte("testkey"))
	if err != nil {
		fmt.Printf("failed to put key in testldb\n")
	}
	fmt.Printf("got byte:%v\n", byte)
}

func newLDB() *LDBDatabase {
	ldb, err := NewLDBDatabase("testldb", 128, 128)
	if err != nil {
		fmt.Printf("error to create ldb : %s\n", "testldb")
		return nil
	}
	return ldb
}
