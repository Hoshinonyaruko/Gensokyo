package idmap

import (
	"bytes"
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
	"sync"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/structs"
	"go.etcd.io/bbolt"
)

var (
	// 用于存储临时指令的全局变量
	TemporaryCommands []string
	// 用于保证线程安全的互斥锁
	MutexT sync.Mutex
)

const (
	DBName          = "idmap.db"
	BucketName      = "ids"
	CacheBucketName = "cache"
	ConfigBucket    = "config"
	UserInfoBucket  = "UserInfo"
	CounterKey      = "currentRow"
)

var db *bbolt.DB

var ErrKeyNotFound = errors.New("key not found")

func InitializeDB() {
	var err error
	// 打开数据库文件
	db, err = bbolt.Open(DBName, 0600, nil)
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}

	// 在数据库中创建必要的buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		// 创建默认的Bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(BucketName)); err != nil {
			return err
		}
		// 创建存储用户信息的Bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(UserInfoBucket)); err != nil {
			return err
		}
		// 创建配置数据的Bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(ConfigBucket)); err != nil {
			return err
		}
		// 创建储存缓存的Bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(CacheBucketName)); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error setting up buckets: %v", err)
	}
}

func DeleteBucket(bucketName string) {
	// 清空指定的bucket
	err := db.Update(func(tx *bbolt.Tx) error {
		// 获取指定的bucket
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			mylog.Printf(bucketName + "表不存在.")
			return nil // 如果bucket不存在，直接返回nil
		}

		// 删除bucket中的所有键值对
		err := bucket.ForEach(func(k, v []byte) error {
			return bucket.Delete(k)
		})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error clearing bucket %s: %v", bucketName, err)
	} else {
		mylog.Printf(bucketName + "清理成功.请手动运行-compaction")
	}
}

func CleanBucket(bucketName string) {
	var deleteCount int

	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		// 使用游标遍历bucket 正向键 k:v 32位openid:大宽int64 64位msgid:大宽int6
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			// 检查键或值是否包含冒号
			if bytes.Contains(k, []byte(":")) || bytes.Contains(v, []byte(":")) || bytes.Contains(k, []byte("row-")) {
				continue // 忽略包含冒号的键值对
			}

			// 检查值id的长度 这里是正向键
			id := string(k)
			if len(id) != 32 {
				if err := c.Delete(); err != nil {
					return err
				}
				deleteCount++
			}
		}

		// 再次遍历处理reverseKey的情况 反向键 row-整数:string 32位openid/64位msgid
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if strings.HasPrefix(string(k), "row-") {
				if bytes.Contains(k, []byte(":")) || bytes.Contains(v, []byte(":")) {
					continue // 忽略包含冒号的键值对
				}
				// 这里检查反向键是否是32位
				id := string(v)
				if len(id) != 32 {
					if err := b.Delete(k); err != nil {
						return err
					}
					deleteCount++
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to clean bucket %s: %v", bucketName, err)
	}

	log.Printf("Cleaned %d entries from bucket %s.", deleteCount, bucketName)
}

func CompactionIdmap() {
	err := Compaction("idmap.db", "idmap_compacted.db")
	if err != nil {
		log.Fatalf("Failed to compact database: %v", err)
	} else {
		log.Println("Database compaction successful.")
		log.Println("请手动备份原始idmap.db(可选)并将idmap_compacted.db改名为idmap.db")
	}
}

// Compaction 创建一个新的数据库文件并复制现有的数据到这个新文件中
func Compaction(sourceDBPath, targetDBPath string) error {
	// 创建目标数据库文件
	targetDB, err := bbolt.Open(targetDBPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer targetDB.Close()

	// 从源数据库复制数据到目标数据库
	err = db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			// 在目标数据库中创建相同的bucket
			return targetDB.Update(func(tx2 *bbolt.Tx) error {
				bucket, err := tx2.CreateBucketIfNotExists(name)
				if err != nil {
					return err
				}
				// 复制所有键值对
				return b.ForEach(func(k, v []byte) error {
					return bucket.Put(k, v)
				})
			})
		})
	})

	if err != nil {
		return err
	}

	// 确保所有操作都已完成
	if err := targetDB.Sync(); err != nil {
		return err
	}

	return nil
}

func CloseDB() {
	db.Close()
}

func GenerateRowID(id string, length int) (int64, error) {
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

	// 取出需要长度的数字或补足0
	var rowIDStr string
	if len(digits) >= length {
		rowIDStr = digits[:length]
	} else {
		// 补足0到右侧
		rowIDStr = digits + strings.Repeat("0", length-len(digits))
	}

	// 将数字字符串转换为int64
	rowID, err := strconv.ParseInt(rowIDStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return rowID, nil
}

// 检查id和value是否是转换关系
func CheckValue(id string, value int64) bool {
	// 计算int64值的长度
	length := len(strconv.FormatInt(value, 10))

	// 使用generateRowID转换id
	generatedValue, err := GenerateRowID(id, length)
	if err != nil {
		fmt.Println("Error:", err)
		return false
	}

	// 比较生成的值与给定的值，如果相等返回false，不相等返回true
	return generatedValue != value
}

func CheckValuev2(value int64) bool {
	var isbinded bool
	if value < 100000 {
		isbinded = false
	} else {
		isbinded = true
	}
	return isbinded
}

// 根据a储存b
func StoreID(id string) (int64, error) {
	var newRow int64

	err := db.Update(func(tx *bbolt.Tx) error {
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
			maxDigits := 18 // int64的位数上限-1
			for digits := 9; digits <= maxDigits; digits++ {
				newRow, err = GenerateRowID(id, digits)
				if err != nil {
					return err
				}
				// 检查新生成的行号是否重复
				rowKey := fmt.Sprintf("row-%d", newRow)
				if b.Get([]byte(rowKey)) == nil {
					// 找到了一个唯一的行号，可以跳出循环
					break
				}
				// 如果到达了最大尝试次数还没有找到唯一的行号，则返回错误
				if digits == maxDigits {
					return fmt.Errorf("unable to find a unique row ID after %d attempts", maxDigits-8)
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

// 根据a储存b
func StoreCache(id string) (int64, error) {
	var newRow int64

	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(CacheBucketName))

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
			maxDigits := 18 // int64的位数上限-1
			for digits := 9; digits <= maxDigits; digits++ {
				newRow, err = GenerateRowID(id, digits)
				if err != nil {
					return err
				}
				// 检查新生成的行号是否重复
				rowKey := fmt.Sprintf("row-%d", newRow)
				if b.Get([]byte(rowKey)) == nil {
					// 找到了一个唯一的行号，可以跳出循环
					break
				}
				// 如果到达了最大尝试次数还没有找到唯一的行号，则返回错误
				if digits == maxDigits {
					return fmt.Errorf("unable to find a unique row ID after %d attempts", maxDigits-8)
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

func SimplifiedStoreID(id string) (int64, error) {
	var newRow int64

	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 生成新的行号
		var err error
		newRow, err = GenerateRowID(id, 9)
		if err != nil {
			return err
		}

		// 检查新生成的行号是否重复
		rowKey := fmt.Sprintf("row-%d", newRow)
		if b.Get([]byte(rowKey)) != nil {
			// 如果行号重复，使用10位数字生成行号
			newRow, err = GenerateRowID(id, 10)
			if err != nil {
				return err
			}
			rowKey = fmt.Sprintf("row-%d", newRow)
			// 再次检查重复性，如果还是重复，则返回错误
			if b.Get([]byte(rowKey)) != nil {
				return fmt.Errorf("unable to find a unique row ID 195")
			}
		}

		// 只写入反向键
		b.Put([]byte(rowKey), []byte(id))

		return nil
	})

	return newRow, err
}

// SimplifiedStoreID 根据a储存b 储存一半
func SimplifiedStoreIDv2(id string) (int64, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=13&id=%s", protocol, serverDir, portValue, id)
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

	// 如果lotus为假,或不走idmaps是真,就保持原来的store的方法
	return SimplifiedStoreID(id)
}

// 群号 然后 用户号
func StoreIDPro(id string, subid string) (int64, int64, error) {
	var newRowID, newSubRowID int64
	var err error

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 生成正向键
		forwardKey := fmt.Sprintf("%s:%s", id, subid)

		// 检查正向键是否已经存在
		existingForwardValue := b.Get([]byte(forwardKey))
		if existingForwardValue != nil {
			// 解析已存在的值
			fmt.Sscanf(string(existingForwardValue), "%d:%d", &newRowID, &newSubRowID)
			return nil
		}

		// 生成新的ID和SubID
		newRowID, err = GenerateRowID(id, 9) // 使用GenerateRowID来生成
		if err != nil {
			return err
		}

		newSubRowID, err = GenerateRowID(subid, 9) // 同样的方法生成SubID
		if err != nil {
			return err
		}
		//反向键
		reverseKey := fmt.Sprintf("%d:%d", newRowID, newSubRowID)
		//正向值
		forwardValue := fmt.Sprintf("%d:%d", newRowID, newSubRowID)
		//反向值
		reverseValue := fmt.Sprintf("%s:%s", id, subid)

		// 存储正向键和反向键
		b.Put([]byte(forwardKey), []byte(forwardValue))
		b.Put([]byte(reverseKey), []byte(reverseValue))

		return nil
	})

	return newRowID, newSubRowID, err
}

// StoreIDv2 根据a储存b
func StoreIDv2(id string) (int64, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
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

// StoreCachev2 根据a储存b
func StoreCachev2(id string) (int64, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=16&id=%s", protocol, serverDir, portValue, id)
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
	return StoreCache(id)
}

// 群号 然后 用户号
func StoreIDv2Pro(id string, subid string) (int64, int64, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=8&id=%s&subid=%s", protocol, serverDir, portValue, id, subid)
		resp, err := http.Get(url)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return 0, 0, fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return 0, 0, fmt.Errorf("error response from server: %s", response["error"])
		}

		newRowValue, ok := response["row"].(float64)
		if !ok {
			return 0, 0, fmt.Errorf("invalid response format for row")
		}

		newSubRowValue, ok := response["subRow"].(float64)
		if !ok {
			return 0, 0, fmt.Errorf("invalid response format for subRow")
		}

		return int64(newRowValue), int64(newSubRowValue), nil
	}

	// 如果lotus为假,调用本地StoreIDPro
	return StoreIDPro(id, subid)
}

// 根据b得到a
func RetrieveRowByID(rowid string) (string, error) {
	var id string
	err := db.View(func(tx *bbolt.Tx) error {
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

// 根据b得到a
func RetrieveRowByCache(rowid string) (string, error) {
	var id string
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(CacheBucketName))

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

// 群号 然后 用户号
func RetrieveRowByIDv2Pro(newRowID string, newSubRowID string) (string, string, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=9&id=%s&subid=%s", protocol, serverDir, portValue, newRowID, newSubRowID)
		resp, err := http.Get(url)
		if err != nil {
			return "", "", fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", "", fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("error response from server: %s", response["error"])
		}

		id, ok := response["id"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format for id")
		}

		subid, ok := response["subid"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format for subid")
		}

		return id, subid, nil
	}

	// 如果lotus为假,调用本地数据库
	return RetrieveRowByIDPro(newRowID, newSubRowID)
}

// 群号 还有用户号
func RetrieveRowByIDPro(newRowID, newSubRowID string) (string, string, error) {
	var id, subid string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 根据新的行号和子行号检索ID和SubID
		reverseKey := fmt.Sprintf("%s:%s", newRowID, newSubRowID)
		reverseValueBytes := b.Get([]byte(reverseKey))
		if reverseValueBytes == nil {
			return ErrKeyNotFound
		}

		reverseValue := string(reverseValueBytes)
		parts := strings.Split(reverseValue, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format for reverse key value")
		}

		id, subid = parts[0], parts[1]

		return nil
	})

	return id, subid, err
}

// RetrieveRowByIDv2 根据b得到a
func RetrieveRowByIDv2(rowid string) (string, error) {
	// 根据portValue确定协议
	protocol := "http"
	portValue := config.GetPortValue()
	if portValue == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
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

// RetrieveRowByCachev2 根据b得到a
func RetrieveRowByCachev2(rowid string) (string, error) {
	// 根据portValue确定协议
	protocol := "http"
	portValue := config.GetPortValue()
	if portValue == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=17&id=%s", protocol, serverDir, portValue, rowid)
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
	return RetrieveRowByCache(rowid)
}

// 根据a 以b为类别 储存c
func WriteConfig(sectionName, keyName, value string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ConfigBucket)) // 直接获取bucket
		if b == nil {
			mylog.Printf("Bucket %s not found", ConfigBucket)
			return fmt.Errorf("bucket %s not found", ConfigBucket)
		}

		key := joinSectionAndKey(sectionName, keyName)
		err := b.Put(key, []byte(value))
		if err != nil {
			mylog.Printf("Error putting data into bucket with key %s: %v", key, err)
			return fmt.Errorf("failed to put data into bucket with key %s: %w", key, err)
		}
		//log.Printf("Data saved successfully with key %s, value %s", key, value)
		return nil
	})
}

// WriteConfigv2 根据a以b为类别储存c
func WriteConfigv2(sectionName, keyName, value string) error {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
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
	err := db.View(func(tx *bbolt.Tx) error {
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

// DeleteConfig根据sectionName和keyName删除指定的键值对
func DeleteConfig(sectionName, keyName string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ConfigBucket))
		if b == nil {
			return fmt.Errorf("bucket %s does not exist", ConfigBucket)
		}

		key := joinSectionAndKey(sectionName, keyName)
		err := b.Delete(key)
		if err != nil {
			return fmt.Errorf("failed to delete data with key %s: %w", key, err)
		}

		return nil
	})
}

// DeleteConfigv2 根据sectionName和keyName远程删除配置
func DeleteConfigv2(sectionName, keyName string) error {
	// 根据portValue确定协议
	protocol := "http"
	portValue := config.GetPortValue()
	if portValue == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()

		// 构建请求URL和参数
		baseURL := fmt.Sprintf("%s://%s:%s/getid", protocol, serverDir, portValue)
		params := url.Values{}
		params.Add("type", "15") // type 15是用于删除操作的
		params.Add("id", sectionName)
		params.Add("subtype", keyName)
		url := baseURL + "?" + params.Encode()

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 如果HTTP状态码是200 OK，表示操作成功完成
		if resp.StatusCode == http.StatusOK {
			// 成功，可以返回nil或者根据需要返回具体的成功消息
			return nil
		} else {
			// 如果状态码不是200 OK，返回错误信息
			return fmt.Errorf("error response from server: %s", resp.Status)
		}
	}

	// 如果lotus为假,则使用原始方法在本地删除配置
	return DeleteConfig(sectionName, keyName) // 假设你已经有了一个本地删除的方法
}

// ReadConfigv2 根据a和b取出c
func ReadConfigv2(sectionName, keyName string) (string, error) {
	// 根据portValue确定协议
	protocol := "http"
	portValue := config.GetPortValue()
	if portValue == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
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

// UpdateVirtualValue 更新旧的虚拟值到新的虚拟值的映射
func UpdateVirtualValue(oldRowValue, newRowValue int64) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 查找旧虚拟值对应的真实值
		oldRowKey := fmt.Sprintf("row-%d", oldRowValue)
		idBytes := b.Get([]byte(oldRowKey))
		if idBytes == nil {
			return fmt.Errorf("不存在:%v", oldRowValue)
		}
		id := string(idBytes)

		// 检查新虚拟值是否已经存在
		newRowKey := fmt.Sprintf("row-%d", newRowValue)
		if b.Get([]byte(newRowKey)) != nil {
			return fmt.Errorf("%v :已存在", newRowValue)
		}

		// 更新真实值到新的虚拟值的映射
		newRowBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(newRowBytes, uint64(newRowValue))
		if err := b.Put([]byte(id), newRowBytes); err != nil {
			return err
		}

		// 更新反向映射
		if err := b.Delete([]byte(oldRowKey)); err != nil {
			return err
		}
		if err := b.Put([]byte(newRowKey), []byte(id)); err != nil {
			return err
		}

		return nil
	})
}

// RetrieveRealValue 根据虚拟值获取真实值，并返回虚拟值及其对应的真实值
func RetrieveRealValue(virtualValue int64) (string, string, error) {
	var realValue string
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 构造键，根据虚拟值查找
		virtualKey := fmt.Sprintf("row-%d", virtualValue)
		realValueBytes := b.Get([]byte(virtualKey))
		if realValueBytes == nil {
			return fmt.Errorf("no real value found for virtual value: %d", virtualValue)
		}
		realValue = string(realValueBytes)

		return nil
	})

	if err != nil {
		return "", "", err
	}

	// 返回虚拟值和对应的真实值
	return fmt.Sprintf("%d", virtualValue), realValue, nil
}

// RetrieveVirtualValue 根据真实值获取虚拟值，并返回真实值及其对应的虚拟值
func RetrieveVirtualValue(realValue string) (string, string, error) {
	var virtualValue int64
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 根据真实值查找虚拟值
		virtualValueBytes := b.Get([]byte(realValue))
		if virtualValueBytes == nil {
			return fmt.Errorf("no virtual value found for real value: %s", realValue)
		}
		virtualValue = int64(binary.BigEndian.Uint64(virtualValueBytes))

		return nil
	})

	if err != nil {
		return "", "", err
	}

	// 返回真实值和对应的虚拟值
	return realValue, fmt.Sprintf("%d", virtualValue), nil
}

// 更新真实值对应的虚拟值
func UpdateVirtualValuev2(oldRowValue, newRowValue int64) error {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 构建请求URL
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}
		url := fmt.Sprintf("%s://%s:%s/getid?type=5&oldRowValue=%d&newRowValue=%d", protocol, serverDir, portValue, oldRowValue, newRowValue)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error response from server")
		}
		return nil
	}

	return UpdateVirtualValue(oldRowValue, newRowValue)
}

// RetrieveRealValuev2 根据虚拟值获取真实值
func RetrieveRealValuev2(virtualValue int64) (string, string, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}
		url := fmt.Sprintf("%s://%s:%s/getid?type=6&virtualValue=%d", protocol, serverDir, portValue, virtualValue)
		resp, err := http.Get(url)
		if err != nil {
			return "", "", fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", "", fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("error response from server")
		}

		realValue, ok := response["real"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format")
		}
		return fmt.Sprintf("%d", virtualValue), realValue, nil
	}

	return RetrieveRealValue(virtualValue)
}

// RetrieveVirtualValuev2 根据真实值获取虚拟值
func RetrieveVirtualValuev2(realValue string) (string, string, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=7&id=%s", protocol, serverDir, portValue, realValue)
		resp, err := http.Get(url)
		if err != nil {
			return "", "", fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", "", fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("error response from server: %s", response["error"])
		}

		virtualValue, ok := response["virtual"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format")
		}

		return realValue, virtualValue, nil
	}

	// 如果lotus为假,就保持原来的RetrieveVirtualValue的方法
	return RetrieveVirtualValue(realValue)
}

// 根据2个真实值 获取2个虚拟值 群号 然后 用户号
func RetrieveVirtualValuev2Pro(realValue string, realValueSub string) (string, string, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=9&id=%s&subid=%s", protocol, serverDir, portValue, realValue, realValueSub)
		resp, err := http.Get(url)
		if err != nil {
			return "", "", fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", "", fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("error response from server: %s", response["error"])
		}

		firstValue, ok := response["id"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format for first value")
		}

		secondValue, ok := response["subid"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format for second value")
		}

		return firstValue, secondValue, nil
	}

	// 如果lotus为假,调用本地RetrieveVirtualValuePro
	return RetrieveVirtualValuePro(realValue, realValueSub)
}

// 根据2个真实值 获取2个虚拟值 群号 然后 用户号
func RetrieveVirtualValuePro(realValue string, realValueSub string) (string, string, error) {
	var newRowID, newSubRowID string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 构建正向键
		forwardKey := fmt.Sprintf("%s:%s", realValue, realValueSub)

		// 从数据库检索正向键对应的值
		forwardValueBytes := b.Get([]byte(forwardKey))
		if forwardValueBytes == nil {
			return ErrKeyNotFound
		}

		forwardValue := string(forwardValueBytes)
		parts := strings.Split(forwardValue, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format for forward key value")
		}

		newRowID, newSubRowID = parts[0], parts[1]

		return nil
	})

	if err != nil {
		return "", "", err
	}

	return newRowID, newSubRowID, nil
}

// RetrieveRealValuePro 根据两个虚拟值获取相应的两个真实值 群号 然后 用户号
func RetrieveRealValuePro(virtualValue1, virtualValue2 int64) (string, string, error) {
	var realValue1, realValue2 string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))

		// 根据两个虚拟值构造键
		compositeKey := fmt.Sprintf("%d:%d", virtualValue1, virtualValue2)
		compositeValueBytes := b.Get([]byte(compositeKey))
		if compositeValueBytes == nil {
			return fmt.Errorf("no real values found for virtual values: %d, %d", virtualValue1, virtualValue2)
		}

		// 解析获取到的真实值
		compositeValue := string(compositeValueBytes)
		parts := strings.Split(compositeValue, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format for composite key value: %s", compositeValue)
		}

		realValue1, realValue2 = parts[0], parts[1]

		return nil
	})

	if err != nil {
		return "", "", err
	}

	return realValue1, realValue2, nil
}

// RetrieveRealValuesv2Pro 根据两个虚拟值获取两个真实值 群号 然后 用户号
func RetrieveRealValuesv2Pro(virtualValue int64, virtualValueSub int64) (string, string, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getrealvalues?type=11&id=%d&subid=%d", protocol, serverDir, portValue, virtualValue, virtualValueSub)
		resp, err := http.Get(url)
		if err != nil {
			return "", "", fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", "", fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("error response from server: %s", response["error"])
		}

		firstRealValue, ok := response["firstRealValue"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format for first real value")
		}

		secondRealValue, ok := response["secondRealValue"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid response format for second real value")
		}

		return firstRealValue, secondRealValue, nil
	}

	// 如果lotus为假,调用本地函数
	return RetrieveRealValuePro(virtualValue, virtualValueSub)
}

// UpdateVirtualValuePro 更新一对旧虚拟值到新虚拟值的映射 旧群号 新群号 旧用户 新用户
func UpdateVirtualValuePro(oldVirtualValue1, newVirtualValue1, oldVirtualValue2, newVirtualValue2 int64) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		// 构造旧和新的复合键
		oldCompositeKey := fmt.Sprintf("%d:%d", oldVirtualValue1, oldVirtualValue2)
		newCompositeKey := fmt.Sprintf("%d:%d", newVirtualValue1, newVirtualValue2)
		// 检查旧复合键是否存在
		compositeValueBytes := b.Get([]byte(oldCompositeKey))
		if compositeValueBytes == nil {
			return fmt.Errorf("不存在的复合虚拟值：%d-%d", oldVirtualValue1, oldVirtualValue2)
		}
		// 检查新复合键是否已经存在
		if b.Get([]byte(newCompositeKey)) != nil {
			return fmt.Errorf("该复合虚拟值已存在：%d-%d", newVirtualValue1, newVirtualValue2)
		}
		// 删除旧的复合键和正向键
		if err := b.Delete([]byte(oldCompositeKey)); err != nil {
			return err
		}
		if err := b.Delete(compositeValueBytes); err != nil {
			return err
		}
		// 反向键
		if err := b.Put([]byte(newCompositeKey), []byte(compositeValueBytes)); err != nil {
			return err
		}
		// 正向键
		if err := b.Put(compositeValueBytes, []byte(newCompositeKey)); err != nil {
			return err
		}

		return nil
	})
}

// UpdateVirtualValuev2Pro 根据配置更新两对虚拟值 旧群 新群 旧用户 新用户
func UpdateVirtualValuev2Pro(oldVirtualValue1, newVirtualValue1, oldVirtualValue2, newVirtualValue2 int64) error {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 构建请求URL
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		url := fmt.Sprintf("%s://%s:%s/getid?type=12&oldVirtualValue1=%d&newVirtualValue1=%d&oldVirtualValue2=%d&newVirtualValue2=%d",
			protocol, serverDir, portValue, oldVirtualValue1, newVirtualValue1, oldVirtualValue2, newVirtualValue2)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error response from server")
		}
		return nil
	}

	return UpdateVirtualValuePro(oldVirtualValue1, newVirtualValue1, oldVirtualValue2, newVirtualValue2)
}

// sub 要匹配的类型 typesuffix 相当于:type 的type
func FindKeysBySubAndType(sub string, typeSuffix string) ([]string, error) {
	var ids []string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ConfigBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", ConfigBucket)
		}

		return b.ForEach(func(k, v []byte) error {
			key := string(k)
			value := string(v)

			// 检查键是否以:type结尾，并且值是否匹配sub
			if strings.HasSuffix(key, typeSuffix) && value == sub {
				// 提取id部分
				id := strings.Split(key, ":")[0]
				ids = append(ids, id)
			}
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return ids, nil
}

// 取相同前缀下的所有key的:后边 比如取群成员列表
func FindSubKeysById(id string) ([]string, error) {
	var subKeys []string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("ids"))
		if b == nil {
			return fmt.Errorf("bucket %s not found", "ids")
		}

		c := b.Cursor()
		prefix := []byte(id + ":")
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			keyParts := bytes.Split(k, []byte(":"))
			if len(keyParts) == 2 {
				subKeys = append(subKeys, string(keyParts[1]))
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return subKeys, nil
}

// FindSubKeysByIdPro 根据1个值获取key中的k:v给出k获取所有v，通过网络调用
func FindSubKeysByIdPro(id string) ([]string, error) {
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() {
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" {
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=14&id=%s", protocol, serverDir, portValue, id)
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error response from server: %s", response["error"])
		}

		keys, ok := response["keys"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid response format for keys")
		}

		// 将interface{}类型的keys转换为[]string
		var resultKeys []string
		for _, key := range keys {
			if strKey, ok := key.(string); ok {
				resultKeys = append(resultKeys, strKey)
			} else {
				return nil, fmt.Errorf("invalid key format in response")
			}
		}

		return resultKeys, nil
	}

	// 如果lotus为假，调用本地函数
	return FindSubKeysById(id)
}

// 场景: xxx:yyy zzz:bbb  zzz:bbb xxx:yyy 把xxx(id)替换为newID 比如更换群号(会卡住)
func UpdateKeysWithNewID(id, newID string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BucketName)
		}

		// 临时存储需要更新的键和反向键
		keysToUpdate := make(map[string]string)

		// 查找所有以id开头的键
		err := b.ForEach(func(k, v []byte) error {
			key := string(k)
			if strings.HasPrefix(key, id+":") {
				value := string(v)
				keysToUpdate[key] = value
			}
			return nil
		})

		if err != nil {
			return err
		}

		// 更新找到的键和对应的反向键
		for key, reverseKey := range keysToUpdate {
			newKey := strings.Replace(key, id, newID, 1)

			// 获取原反向键的值
			reverseValueBytes := b.Get([]byte(reverseKey))
			if reverseValueBytes == nil {
				return fmt.Errorf("reverse key %s not found", reverseKey)
			}

			// 更新原键
			err := b.Delete([]byte(key))
			if err != nil {
				return err
			}
			err = b.Put([]byte(newKey), []byte(reverseKey))
			if err != nil {
				return err
			}

			// 更新反向键的值
			newReverseValue := strings.Replace(string(reverseValueBytes), id, newID, 1)
			err = b.Put([]byte(reverseKey), []byte(newReverseValue))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// StoreUserInfo 存储用户信息
func StoreUserInfo(rawID string, userInfo structs.FriendData) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(UserInfoBucket))
		key := fmt.Sprintf("%s:%s", rawID, userInfo.UserID) // 创建复合键
		if v := b.Get([]byte(key)); v != nil {
			return fmt.Errorf("duplicate key: %s", key)
		}

		// 序列化用户信息作为值
		value, err := json.Marshal(userInfo)
		if err != nil {
			return fmt.Errorf("could not encode user info: %s", err)
		}

		// 存储键值对
		if err := b.Put([]byte(key), value); err != nil {
			return fmt.Errorf("could not store user info: %s", err)
		}
		return nil
	})
}

// ListAllUsers 返回数据库中所有用户的信息
func ListAllUsers() ([]structs.FriendData, error) {
	var users []structs.FriendData
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(UserInfoBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", UserInfoBucket)
		}

		// 遍历bucket中的所有键值对
		err := b.ForEach(func(key, value []byte) error {
			var user structs.FriendData
			if err := json.Unmarshal(value, &user); err != nil {
				log.Printf("Error unmarshaling user data: %v", err)
				return err
			}
			users = append(users, user)
			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}
