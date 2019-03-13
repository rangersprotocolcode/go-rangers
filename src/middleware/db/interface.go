package db

import "github.com/syndtr/goleveldb/leveldb/iterator"

const IdealBatchSize = 100 * 1024

type Putter interface {
	Put(key []byte, value []byte) error
}

type Batch interface {
	Putter
	ValueSize() int // amount of data in the batch
	Write() error
	// Reset resets the batch for reuse
	Reset()
}

type Database interface {
	Putter
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	Close()
	NewBatch() Batch
	NewIterator() iterator.Iterator
	NewIteratorWithPrefix(prefix []byte) iterator.Iterator
}
