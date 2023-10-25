package idmap

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/boltdb/bolt"
	"github.com/hoshinonyaruko/gensokyo/config"
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
func StoreIDv2(id string) (int64, error) {
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

// StoreIDv2v2 根据a储存b
func StoreIDv2v2(id string) (int64, error) {
	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 构建请求URL
		url := fmt.Sprintf("http://%s:%s/getid?type=1&id=%s", serverDir, portValue, id)
		resp, err := http.Get(url)
		if err != nil {
			return 0, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return 0, fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("error response from server: %s", response["error"])
		}

		rowValue, ok := response["row"].(float64)
		if !ok {
			return 0, fmt.Errorf("invalid response format")
		}

		return int64(rowValue), nil
	}

	// 如果lotus为假,就保持原来的store的方法
	return StoreIDv2(id)
}

// 根据b得到a
func RetrieveRowByIDv2(rowid string) (string, error) {
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

// RetrieveRowByIDv2v2 根据b得到a
func RetrieveRowByIDv2v2(rowid string) (string, error) {
	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 构建请求URL
		url := fmt.Sprintf("http://%s:%s/getid?type=2&id=%s", serverDir, portValue, rowid)
		resp, err := http.Get(url)
		if err != nil {
			return "", fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("error response from server: %s", response["error"])
		}

		idValue, ok := response["id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid response format")
		}

		return idValue, nil
	}

	// 如果lotus为假,就保持原来的RetrieveRowByIDv2的方法
	return RetrieveRowByIDv2(rowid)
}

// 根据a 以b为类别 储存c
func WriteConfigv2(sectionName, keyName, value string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(ConfigBucket))
		if err != nil {
			return err
		}

		key := joinSectionAndKey(sectionName, keyName)
		return b.Put(key, []byte(value))
	})
}

// WriteConfigv2v2 根据a以b为类别储存c
func WriteConfigv2v2(sectionName, keyName, value string) error {
	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 构建请求URL和参数
		baseURL := fmt.Sprintf("http://%s:%s/getid", serverDir, portValue)
		params := url.Values{}
		params.Add("type", "3")
		params.Add("id", sectionName)
		params.Add("subtype", keyName)
		params.Add("value", value)
		url := baseURL + "?" + params.Encode()

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error response from server: %s", resp.Status)
		}

		return nil
	}

	// 如果lotus为假,则使用原始方法在本地写入配置
	return WriteConfigv2(sectionName, keyName, value)
}

// 根据a和b取出c
func ReadConfigv2(sectionName, keyName string) (string, error) {
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

// ReadConfigv2v2 根据a和b取出c
func ReadConfigv2v2(sectionName, keyName string) (string, error) {
	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 构建请求URL和参数
		baseURL := fmt.Sprintf("http://%s:%s/getid", serverDir, portValue)
		params := url.Values{}
		params.Add("type", "4")
		params.Add("id", sectionName)
		params.Add("subtype", keyName)
		url := baseURL + "?" + params.Encode()

		resp, err := http.Get(url)
		if err != nil {
			return "", fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("error response from server: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %v", err)
		}

		var responseMap map[string]interface{}
		if err := json.Unmarshal(body, &responseMap); err != nil {
			return "", fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if value, ok := responseMap["value"]; ok {
			return fmt.Sprintf("%v", value), nil
		}

		return "", fmt.Errorf("value not found in response")
	}

	// 如果lotus为假,则使用原始方法在本地读取配置
	return ReadConfigv2(sectionName, keyName)
}

// 灵感,ini配置文件
func joinSectionAndKey(sectionName, keyName string) []byte {
	return []byte(sectionName + ":" + keyName)
}
