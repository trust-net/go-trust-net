package db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

type DatabaseLevelDB struct {
	fn string      // filename for reporting
	ldb *leveldb.DB // LevelDB instance
}

func NewDatabaseLevelDB(file string, cache int, handles int) (*DatabaseLevelDB, error) {
	// Ensure we have some minimal caching and file guarantees
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	ldb, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2,
		WriteBuffer:            cache / 4, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		ldb, err = leveldb.RecoverFile(file, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}

	db := &DatabaseLevelDB{
		ldb: ldb,
		fn: file,
	}
	return db, nil
}


func (db *DatabaseLevelDB) Put(key []byte, value []byte) error {
	return db.ldb.Put(key, value, nil)
}

func (db *DatabaseLevelDB) Get(key []byte) ([]byte, error) {
	return db.ldb.Get(key, nil)
}

func (db *DatabaseLevelDB) Has(key []byte) (bool, error) {
	return db.ldb.Has(key, nil)
}

func (db *DatabaseLevelDB) Delete(key []byte) error {
	return db.ldb.Delete(key, nil)
}

func (db *DatabaseLevelDB) Close() error {
	return db.ldb.Close()
}
