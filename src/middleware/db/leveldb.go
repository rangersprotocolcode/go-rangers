package db

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/utility"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
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

	db, err := newLevelDBInstance("storage"+strconv.Itoa(common.InstanceIndex)+"/"+file, 8, 8)
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

// 生成leveldb实例
func newLevelDBInstance(file string, cache int, handles int) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     200 * opt.MiB,
		WriteBuffer:            cache * opt.MiB, // Two of these are used internally
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

func (ldb *LDBDatabase) Clear() error {
	ldb.inited = false
	ldb.Close()

	// todo: 直接删除文件，是不是过于粗暴？
	os.RemoveAll(ldb.Path())

	db, err := newLevelDBInstance(ldb.Path(), ldb.cacheConfig, ldb.handlesConfig)
	if err != nil {
		return err
	}

	ldb.db = db
	ldb.inited = true
	return nil
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

	db.logger.Debugf("put, key: %s, length: %d", utility.Bytes2Str(key), len(value))
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
			//db.log.Error("Metrics collection failed", "err", err)
		}
	}

	db.db.Close()
	//err := db.db.Close()
	//if err == nil {
	//	db.log.Info("Database closed")
	//} else {
	//	db.log.Error("Failed to close database", "err", err)
	//}
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