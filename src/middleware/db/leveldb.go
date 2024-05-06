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
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
	"sync"
)

type LDBDatabase struct {
	db *leveldb.DB

	quitLock sync.Mutex
	quitChan chan chan error

	filename      string
	cacheConfig   int
	handlesConfig int

	inited bool

	logger log.Logger
}

func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
	if cache < 8 {
		cache = 8
	}
	if handles < 8 {
		handles = 8
	}

	path := "storage" + strconv.Itoa(common.InstanceIndex) + "/" + file
	db, err := newLevelDBInstance(path, cache, handles)
	if err != nil {
		return nil, err
	}

	return &LDBDatabase{
		filename:      file,
		db:            db,
		cacheConfig:   cache,
		handlesConfig: handles,
		inited:        true,
		logger:        log.GetLoggerByIndex(log.LdbLogConfig, strconv.Itoa(common.InstanceIndex)),
	}, nil
}

func newLevelDBInstance(file string, cache int, handles int) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(file, &opt.Options{
		BlockSize:              1 * opt.MiB,
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache * opt.MiB,
		WriteBuffer:            128 * opt.MiB,
		Filter:                 filter.NewBloomFilter(20),
	})

	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}

	if err != nil {
		return nil, err
	}

	return db, nil
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
	return db.filename
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	if !db.inited {
		return ErrLDBInit
	}
	return db.db.Put(key, value, nil)
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	if !db.inited {
		return false, ErrLDBInit
	}

	return db.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	if !db.inited {
		return nil, ErrLDBInit
	}

	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil

}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
	if !db.inited {
		return ErrLDBInit
	}
	return db.db.Delete(key, nil)
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
	if !db.inited {
		return nil
	}
	return db.db.NewIterator(nil, nil)
}

// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
func (db *LDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
}

func (db *LDBDatabase) Close() {
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			fmt.Println(err)
		}
	}

	db.db.Close()
}

func (db *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.db, b: new(leveldb.Batch), logger: db.logger}
}

type ldbBatch struct {
	logger log.Logger
	db     *leveldb.DB
	b      *leveldb.Batch
	size   int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Write() error {
	b.logger.Debugf("batchWrite. length: %d ", b.size)
	return b.db.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

func (b *ldbBatch) Reset() {
	b.b.Reset()
	b.size = 0
}
