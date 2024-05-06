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
	"bytes"
	"sync"

	"com.tuntun.rangers/node/src/common"

	"github.com/hashicorp/golang-lru"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

var (
	ErrLDBInit   = errors.New("LDB instance not inited")
	instance     *LDBDatabase
	instanceLock = sync.RWMutex{}
)

type PrefixedDatabase struct {
	db     *LDBDatabase
	prefix string
}

type databaseConfig struct {
	database string
	cache    int
	handler  int
}

func NewDatabase(prefix string) (Database, error) {
	dbInner, err := getInstance()
	if nil != err {
		return nil, err
	}

	return &PrefixedDatabase{
		db:     dbInner,
		prefix: prefix,
	}, nil
}

func getInstance() (*LDBDatabase, error) {
	instanceLock.Lock()
	defer instanceLock.Unlock()

	var (
		instanceInner *LDBDatabase
		err           error
	)
	if nil != instance {
		return instance, nil
	}

	defaultConfig := &databaseConfig{
		database: common.DefaultDatabase,
		cache:    512,
		handler:  512,
	}

	if nil == common.GlobalConf {
		instanceInner, err = NewLDBDatabase(defaultConfig.database, defaultConfig.cache, defaultConfig.handler)
	} else {
		instanceInner, err = NewLDBDatabase(common.GlobalConf.GetString(common.ConfigSec, common.DefaultDatabase, defaultConfig.database), common.GlobalConf.GetInt(common.ConfigSec, "cache", defaultConfig.cache), common.GlobalConf.GetInt(common.ConfigSec, "handler", defaultConfig.handler))
	}

	if nil == err {
		instance = instanceInner
	}

	return instance, err
}

//func (db *PrefixedDatabase) Clear() error {
//	inner := db.db
//	if nil == inner {
//		return ErrLDBInit
//	}
//
//	return inner.Clear()
//}

func (db *PrefixedDatabase) Close() {
	instanceLock.Lock()
	defer instanceLock.Unlock()

	instance = nil
	db.db.Close()
}

func (db *PrefixedDatabase) Put(key []byte, value []byte) error {
	return db.db.Put(generateKey(key, db.prefix), value)
}

func (db *PrefixedDatabase) Get(key []byte) ([]byte, error) {
	return db.db.Get(generateKey(key, db.prefix))
}

func (db *PrefixedDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(generateKey(key, db.prefix))
}

func (db *PrefixedDatabase) Delete(key []byte) error {
	return db.db.Delete(generateKey(key, db.prefix))
}

func (db *PrefixedDatabase) NewIterator() iterator.Iterator {
	return db.db.NewIteratorWithPrefix([]byte(db.prefix))
}

func (db *PrefixedDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return db.db.NewIteratorWithPrefix(generateKey(prefix, db.prefix))
}

func (db *PrefixedDatabase) NewBatch() Batch {

	return &prefixBatch{db: db.db.db, b: new(leveldb.Batch), prefix: db.prefix}
}

type prefixBatch struct {
	db     *leveldb.DB
	b      *leveldb.Batch
	size   int
	prefix string
}

func (b *prefixBatch) Put(key, value []byte) error {
	b.b.Put(generateKey(key, b.prefix), value)
	b.size += len(value)
	return nil
}

func (b *prefixBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *prefixBatch) ValueSize() int {
	return b.size
}

func (b *prefixBatch) Reset() {
	b.b.Reset()
	b.size = 0
}

func generateKey(raw []byte, prefix string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte(prefix))
	bytesBuffer.Write(raw)
	return bytesBuffer.Bytes()
}

type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemDatabase() (*MemDatabase, error) {
	return &MemDatabase{
		db: make(map[string][]byte),
	}, nil
}

func (db *MemDatabase) Clear() error {
	db.db = make(map[string][]byte)
	return nil
}
func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) NewIterator() iterator.Iterator {
	panic("Not support")
}

func (db *MemDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	panic("Not support")
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

func (db *MemDatabase) Close() {}

func (db *MemDatabase) NewBatch() Batch {
	return &memBatch{db: db}
}

func (db *MemDatabase) Len() int { return len(db.db) }

type kv struct{ k, v []byte }

type memBatch struct {
	db     *MemDatabase
	writes []kv
	size   int
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value)})
	b.size += len(value)
	return nil
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		b.db.db[string(kv.k)] = kv.v
	}
	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}

type LRUMemDatabase struct {
	db   *lru.Cache
	lock sync.RWMutex
}

func NewLRUMemDatabase(size int) (*LRUMemDatabase, error) {
	cache, _ := lru.New(size)
	return &LRUMemDatabase{
		db: cache,
	}, nil
}

func (db *LRUMemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.db.Add(string(key), common.CopyBytes(value))
	return nil
}

func (db *LRUMemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db.Get(string(key))
	return ok, nil
}

func (db *LRUMemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db.Get(string(key)); ok {
		vl, _ := entry.([]byte)
		return common.CopyBytes(vl), nil
	}
	return nil, nil
}

func (db *LRUMemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.db.Remove(string(key))
	return nil
}

func (db *LRUMemDatabase) Close() {}

func (db *LRUMemDatabase) NewBatch() Batch {
	return &LruMemBatch{db: db}
}

func (db *LRUMemDatabase) NewIterator() iterator.Iterator {
	panic("Not support")
}

func (db *LRUMemDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	panic("Not support")
}

type LruMemBatch struct {
	db     *LRUMemDatabase
	writes []kv
	size   int
}

func (b *LruMemBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value)})
	b.size += len(value)
	return nil
}

func (b *LruMemBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		b.db.db.Add(string(kv.k), kv.v)
	}
	return nil
}

func (b *LruMemBatch) ValueSize() int {
	return b.size
}

func (b *LruMemBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}
