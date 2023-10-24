package idmap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

const (
	DBName       = "idmap.db"
	BucketName   = "ids"
	ConfigBucket = "config"
	CounterKey   = "currentRow"
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

// 根据a储存b
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

// 根据b得到a
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

// 根据a 以b为类别 储存c
func WriteConfig(sectionName, keyName, value string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(ConfigBucket))
		if err != nil {
			return err
		}

		key := joinSectionAndKey(sectionName, keyName)
		return b.Put(key, []byte(value))
	})
}

// 根据a和b取出c
func ReadConfig(sectionName, keyName string) (string, error) {
	var result string
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ConfigBucket))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		key := joinSectionAndKey(sectionName, keyName)
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("key '%s' in section '%s' does not exist", keyName, sectionName)
		}

		result = string(v)
		return nil
	})

	return result, err
}

// 灵感,ini配置文件
func joinSectionAndKey(sectionName, keyName string) []byte {
	return []byte(sectionName + ":" + keyName)
}
