package idmap

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
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
func generateRowID(id string, length int) (int64, error) {
	// 计算MD5哈希值
	hasher := md5.New()
	hasher.Write([]byte(id))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// 只保留数字
	var digitsBuilder strings.Builder
	for _, r := range hash {
		if r >= '0' && r <= '9' {
			digitsBuilder.WriteRune(r)
		}
	}
	digits := digitsBuilder.String()

	// 取出需要长度的数字
	var rowIDStr string
	if len(digits) >= length {
		rowIDStr = digits[:length]
	} else {
		// 如果数字不足指定长度，返回错误
		return 0, fmt.Errorf("not enough digits in MD5 hash")
	}

	// 将数字字符串转换为int64
	rowID, err := strconv.ParseInt(rowIDStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return rowID, nil
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
		//写入虚拟值
		if !config.GetHashIDValue() {
			// 如果ID不存在，则为它分配一个新的行号 数字递增
			currentRowBytes := b.Get([]byte(CounterKey))
			if currentRowBytes == nil {
				newRow = 1
			} else {
				currentRow := binary.BigEndian.Uint64(currentRowBytes)
				newRow = int64(currentRow) + 1
			}
		} else {
			// 生成新的行号
			var err error
			newRow, err = generateRowID(id, 9)
			if err != nil {
				return err
			}
			// 检查新生成的行号是否重复
			rowKey := fmt.Sprintf("row-%d", newRow)
			if b.Get([]byte(rowKey)) != nil {
				// 如果行号重复，使用10位数字生成行号
				newRow, err = generateRowID(id, 10)
				if err != nil {
					return err
				}
				rowKey = fmt.Sprintf("row-%d", newRow)
				// 再次检查重复性，如果还是重复，则返回错误
				if b.Get([]byte(rowKey)) != nil {
					return fmt.Errorf("unable to find a unique row ID")
				}
			}
		}

		rowBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(rowBytes, uint64(newRow))
		//写入递增值
		if !config.GetHashIDValue() {
			b.Put([]byte(CounterKey), rowBytes)
		}
		//真实对应虚拟 用来直接判断是否存在,并快速返回
		b.Put([]byte(id), rowBytes)

		reverseKey := fmt.Sprintf("row-%d", newRow)
		b.Put([]byte(reverseKey), []byte(id))

		return nil
	})

	return newRow, err
}

// StoreIDv2 根据a储存b
func StoreIDv2(id string) (int64, error) {
	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=1&id=%s", protocol, serverDir, portValue, id)
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
	return StoreID(id)
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

// RetrieveRowByIDv2 根据b得到a
func RetrieveRowByIDv2(rowid string) (string, error) {
	// 根据portValue确定协议
	protocol := "http"
	portValue := config.GetPortValue()
	if portValue == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=2&id=%s", protocol, serverDir, portValue, rowid)
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
	return RetrieveRowByID(rowid)
}

// 根据a 以b为类别 储存c
func WriteConfig(sectionName, keyName, value string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(ConfigBucket))
		if err != nil {
			mylog.Printf("Error creating or accessing bucket: %v", err)
			return fmt.Errorf("failed to access or create bucket %s: %w", ConfigBucket, err)
		}

		key := joinSectionAndKey(sectionName, keyName)
		err = b.Put(key, []byte(value))
		if err != nil {
			mylog.Printf("Error putting data into bucket with key %s: %v", key, err)
			return fmt.Errorf("failed to put data into bucket with key %s: %w", key, err)
		}
		//mylog.Printf("Data saved successfully with key %s,value %s", key, value)
		return nil
	})
}

// WriteConfigv2 根据a以b为类别储存c
func WriteConfigv2(sectionName, keyName, value string) error {
	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL和参数
		baseURL := fmt.Sprintf("%s://%s:%s/getid", protocol, serverDir, portValue)

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
	return WriteConfig(sectionName, keyName, value)
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

// ReadConfigv2 根据a和b取出c
func ReadConfigv2(sectionName, keyName string) (string, error) {
	// 根据portValue确定协议
	protocol := "http"
	portValue := config.GetPortValue()
	if portValue == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()

		// 构建请求URL和参数
		baseURL := fmt.Sprintf("%s://%s:%s/getid", protocol, serverDir, portValue)
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
	return ReadConfig(sectionName, keyName)
}

// 灵感,ini配置文件
func joinSectionAndKey(sectionName, keyName string) []byte {
	return []byte(sectionName + ":" + keyName)
}
