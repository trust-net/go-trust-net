package db

import (
	"sync"
	"errors"
	"github.com/trust-net/go-trust-net/core"
)

// in memory implementation of database (for testing etc.)
type DatabaseInMem struct {
	mdb map[core.Byte64][]byte
	lock sync.RWMutex
}

func NewDatabaseInMem() (*DatabaseInMem, error) {
	return &DatabaseInMem{
		mdb: make(map[core.Byte64][]byte),
	}, nil
}

func (db *DatabaseInMem) Map() map[core.Byte64][]byte {
	return db.mdb
}

func (db *DatabaseInMem) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.mdb[*core.BytesToByte64(key)] = value
	return nil
}

func (db *DatabaseInMem) Get(key []byte) ([]byte, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	if data, ok := db.mdb[*core.BytesToByte64(key)]; !ok {
		return data, errors.New("not found")
	} else {
		return data, nil
	}
}

func (db *DatabaseInMem) Has(key []byte) (bool, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	_, ok := db.mdb[*core.BytesToByte64(key)]
	return  ok, nil
}

func (db *DatabaseInMem) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	delete(db.mdb, *core.BytesToByte64(key))
	return nil
}

func (db *DatabaseInMem) Close() error{
	return nil
}