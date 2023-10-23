package idmap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

const (
	DBName     = "idmap.db"
	BucketName = "ids"
	CounterKey = "currentRow"
)

var db *bolt.DB

var ErrKeyNotFound = errors.New("key not found")

func InitializeDB() {
	var err error
	db, err = bolt.Open(DBName, 0600, nil)
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BucketName))
		return err
	})
}

func CloseDB() {
	db.Close()
}

func StoreID(id string) (int64, error) {
	var newRow int64

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 检查ID是否已经存在
		existingRowBytes := b.Get([]byte(id))
		if existingRowBytes != nil {
			newRow = int64(binary.BigEndian.Uint64(existingRowBytes))
			return nil
		}

		// 如果ID不存在，则为它分配一个新的行号
		currentRowBytes := b.Get([]byte(CounterKey))
		if currentRowBytes == nil {
			newRow = 1
		} else {
			currentRow := binary.BigEndian.Uint64(currentRowBytes)
			newRow = int64(currentRow) + 1
		}

		rowBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(rowBytes, uint64(newRow))
		b.Put([]byte(CounterKey), rowBytes)
		b.Put([]byte(id), rowBytes)

		reverseKey := fmt.Sprintf("row-%d", newRow)
		b.Put([]byte(reverseKey), []byte(id))

		return nil
	})

	return newRow, err
}

func RetrieveRowByID(rowid string) (string, error) {
	var id string
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 根据行号检索ID
		idBytes := b.Get([]byte("row-" + rowid))
		if idBytes == nil {
			return ErrKeyNotFound
		}
		id = string(idBytes)

		return nil
	})

	return id, err
}
