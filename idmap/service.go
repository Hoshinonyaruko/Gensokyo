import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/structs"
	"golang.org/x/net/context"
)

var (
	// 用于存储临时指令的全局变量
	TemporaryCommands []string
	// 用于保证线程安全的互斥锁
	MutexT sync.Mutex
)

const (
	RedisAddr       = "localhost:6379"
	RedisPassword   = ""
	RedisDB         = 0
)

var rdb *redis.Client
var ctx = context.Background()

var ErrKeyNotFound = errors.New("key not found")

func InitializeDB() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: RedisPassword,
		DB:       RedisDB,
	})

	// 测试连接
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
}

func DeleteBucket(bucketName string) {
	// 清空指定的bucket
	err := rdb.Del(ctx, bucketName).Err()
	if err != nil {
		log.Fatalf("Error clearing bucket %s: %v", bucketName, err)
	} else {
		mylog.Printf(bucketName + "清理成功.请手动运行-compaction")
	}
}

func CleanBucket(bucketName string) {
	var deleteCount int

	// 获取所有键
	keys, err := rdb.HKeys(ctx, bucketName).Result()
	if err != nil {
		log.Fatalf("Failed to retrieve keys from bucket %s: %v", bucketName, err)
	}

	for _, k := range keys {
		// 获取值
		v, err := rdb.HGet(ctx, bucketName, k).Result()
		if err != nil {
			continue
		}

		// 检查键或值是否包含冒号
		if strings.Contains(k, ":") || strings.Contains(v, ":") || strings.Contains(k, "row-") {
			continue // 忽略包含冒号的键值对
		}

		// 检查值id的长度
		if len(k) != 32 {
			err := rdb.HDel(ctx, bucketName, k).Err()
			if err != nil {
				log.Fatalf("Failed to delete key %s from bucket %s: %v", k, bucketName, err)
			}
			deleteCount++
		}
	}

	// 再次遍历处理reverseKey的情况
	for _, k := range keys {
		if strings.HasPrefix(k, "row-") {
			v, err := rdb.HGet(ctx, bucketName, k).Result()
			if err != nil {
				continue
			}

			if strings.Contains(k, ":") || strings.Contains(v, ":") {
				continue // 忽略包含冒号的键值对
			}

			// 这里检查反向键是否是32位
			if len(v) != 32 {
				err := rdb.HDel(ctx, bucketName, k).Err()
				if err != nil {
					log.Fatalf("Failed to delete key %s from bucket %s: %v", k, bucketName, err)
				}
				deleteCount++
			}
		}
	}

	log.Printf("Cleaned %d entries from bucket %s.", deleteCount, bucketName)
}

func CompactionIdmap() {
	log.Println("Compaction is not applicable for Redis. Please ensure the data integrity manually.")
}

func CloseDB() {
	if err := rdb.Close(); err != nil {
		log.Fatalf("Failed to close Redis connection: %v", err)
	}
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

	// 使用Redis事务来保证数据一致性
	err := rdb.Watch(ctx, func(tx *redis.Tx) error {
		// 检查ID是否已经存在
		existingRowStr, err := tx.HGet(ctx, BucketName, id).Result()
		if err == nil {
			newRow, err = strconv.ParseInt(existingRowStr, 10, 64)
			if err != nil {
				return err
			}
			return nil
		}

		if !config.GetHashIDValue() {
			// 如果ID不存在，则为它分配一个新的行号 数字递增
			currentRowStr, err := tx.Get(ctx, CounterKey).Result()
			if err == redis.Nil {
				newRow = 1
			} else if err != nil {
				return err
			} else {
				currentRow, err := strconv.ParseInt(currentRowStr, 10, 64)
				if err != nil {
					return err
				}
				newRow = currentRow + 1
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
				exists, err := tx.HExists(ctx, BucketName, rowKey).Result()
				if err != nil {
					return err
				}
				if !exists {
					// 找到了一个唯一的行号，可以跳出循环
					break
				}
				// 如果到达了最大尝试次数还没有找到唯一的行号，则返回错误
				if digits == maxDigits {
					return fmt.Errorf("unable to find a unique row ID after %d attempts", maxDigits-8)
				}
			}
		}

		// 使用事务来写入数据
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			if !config.GetHashIDValue() {
				pipe.Set(ctx, CounterKey, newRow, 0)
			}
			pipe.HSet(ctx, BucketName, id, newRow)
			reverseKey := fmt.Sprintf("row-%d", newRow)
			pipe.HSet(ctx, BucketName, reverseKey, id)
			return nil
		})
		return err
	}, BucketName)

	return newRow, err
}

// 根据a储存b
func StoreCache(id string) (int64, error) \{
	var newRow int64

	// 使用Redis事务来保证数据一致性
	err := rdb.Watch(ctx, func(tx *redis.Tx) error \{
		// 检查ID是否已经存在
		existingRowStr, err := tx.HGet(ctx, CacheBucketName, id).Result()
		if err == nil \{
			newRow, err = strconv.ParseInt(existingRowStr, 10, 64)
			if err != nil \{
				return err
			}
			return nil
		}

		if !config.GetHashIDValue() \{
			// 如果ID不存在，则为它分配一个新的行号 数字递增
			currentRowStr, err := tx.Get(ctx, CounterKey).Result()
			if err == redis.Nil \{
				newRow = 1
			} else if err != nil \{
				return err
			} else \{
				currentRow, err := strconv.ParseInt(currentRowStr, 10, 64)
				if err != nil \{
					return err
				}
				newRow = currentRow + 1
			}
		} else \{
			// 生成新的行号
			var err error
			maxDigits := 18 // int64的位数上限-1
			for digits := 9; digits <= maxDigits; digits++ \{
				newRow, err = GenerateRowID(id, digits)
				if err != nil \{
					return err
				}
				// 检查新生成的行号是否重复
				rowKey := fmt.Sprintf("row-%d", newRow)
				exists, err := tx.HExists(ctx, CacheBucketName, rowKey).Result()
				if err != nil \{
					return err
				}
				if !exists \{
					// 找到了一个唯一的行号，可以跳出循环
					break
				}
				// 如果到达了最大尝试次数还没有找到唯一的行号，则返回错误
				if digits == maxDigits \{
					return fmt.Errorf("unable to find a unique row ID after %d attempts", maxDigits-8)
				}
			}
		}

		// 使用事务来写入数据
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error \{
			if !config.GetHashIDValue() \{
				pipe.Set(ctx, CounterKey, newRow, 0)
			}
			pipe.HSet(ctx, CacheBucketName, id, newRow)
			reverseKey := fmt.Sprintf("row-%d", newRow)
			pipe.HSet(ctx, CacheBucketName, reverseKey, id)
			return nil
		})
		return err
	}, CacheBucketName)

	return newRow, err
}

func SimplifiedStoreID(id string) (int64, error) \{
	var newRow int64

	// 使用Redis事务来保证数据一致性
	err := rdb.Watch(ctx, func(tx *redis.Tx) error \{
		// 生成新的行号
		var err error
		newRow, err = GenerateRowID(id, 9)
		if err != nil \{
			return err
		}

		// 检查新生成的行号是否重复
		rowKey := fmt.Sprintf("row-%d", newRow)
		exists, err := tx.HExists(ctx, BucketName, rowKey).Result()
		if err != nil \{
			return err
		}
		if exists \{
			// 如果行号重复，使用10位数字生成行号
			newRow, err = GenerateRowID(id, 10)
			if err != nil \{
				return err
			}
			rowKey = fmt.Sprintf("row-%d", newRow)
			// 再次检查重复性，如果还是重复，则返回错误
			exists, err = tx.HExists(ctx, BucketName, rowKey).Result()
			if err != nil \{
				return err
			}
			if exists \{
				return fmt.Errorf("unable to find a unique row ID 195")
			}
		}

		// 只写入反向键
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error \{
			pipe.HSet(ctx, BucketName, rowKey, id)
			return nil
		})

		return err
	}, BucketName)

	return newRow, err
}

func SimplifiedStoreIDv2(id string) (int64, error) \{
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() \{
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" \{
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=13&id=%s", protocol, serverDir, portValue, id)
		resp, err := http.Get(url)
		if err != nil \{
			return 0, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface\{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil \{
			return 0, fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK \{
			return 0, fmt.Errorf("error response from server: %s", response["error"])
		}

		rowValue, ok := response["row"].(float64)
		if !ok \{
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
	id, err := rdb.HGet(ctx, BucketName, "row-" + rowid).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	return id, err
}

func RetrieveRowByCache(rowid string) (string, error) {
	id, err := rdb.HGet(ctx, CacheBucketName, "row-" + rowid).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	return id, err
}
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

func RetrieveRowByIDPro(newRowID, newSubRowID string) (string, string, error) {
	var id, subid string

	reverseKey := fmt.Sprintf("%s:%s", newRowID, newSubRowID)
	reverseValue, err := rdb.HGet(ctx, BucketName, reverseKey).Result()
	if err == redis.Nil {
		return "", "", ErrKeyNotFound
	} else if err != nil {
		return "", "", err
	}

	parts := strings.Split(reverseValue, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format for reverse key value")
	}

	id, subid = parts[0], parts[1]

	return id, subid, nil
}

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

	// 如果lotus为假,就保持原来的RetrieveRowByCache的方法
	return RetrieveRowByCache(rowid)
}

func WriteConfig(sectionName, keyName, value string) error {
	key := joinSectionAndKey(sectionName, keyName)
	err := rdb.HSet(ctx, ConfigBucket, key, value).Err()
	if err != nil {
		mylog.Printf("Error putting data into bucket with key %s: %v", key, err)
		return fmt.Errorf("failed to put data into bucket with key %s: %w", key, err)
	}
	return nil
}

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

	// 如果lotus为假,则使用Redis写入配置
	return WriteConfig(sectionName, keyName, value)
}

// 根据a和b取出c

func ReadConfig(sectionName, keyName string) (string, error) {
	key := joinSectionAndKey(sectionName, keyName)
	result, err := rdb.HGet(ctx, ConfigBucket, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key '%s' in section '%s' does not exist", keyName, sectionName)
	} else if err != nil {
		return "", err
	}

	return result, nil
}


// DeleteConfig根据sectionName和keyName删除指定的键值对
func DeleteConfig(sectionName, keyName string) error {
	key := joinSectionAndKey(sectionName, keyName)
	err := rdb.HDel(ctx, ConfigBucket, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete data with key %s: %w", key, err)
	}
	return nil
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

	// 如果lotus为假,则使用Redis删除配置
	return DeleteConfig(sectionName, keyName)
}

// ReadConfigv2 根据a和b取出c
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

	// 如果lotus为假,则使用Redis删除配置
	return DeleteConfig(sectionName, keyName)
}

// 灵感,ini配置文件
func joinSectionAndKey(sectionName, keyName string) string {
	return fmt.Sprintf("%s:%s", sectionName, keyName)
}

// UpdateVirtualValue 更新旧的虚拟值到新的虚拟值的映射
func UpdateVirtualValue(oldRowValue, newRowValue int64) error {
	oldRowKey := fmt.Sprintf("row-%d", oldRowValue)
	newRowKey := fmt.Sprintf("row-%d", newRowValue)

	// 使用事务确保操作的原子性
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// 查找旧虚拟值对应的真实值
		id, err := rdb.Get(ctx, oldRowKey).Result()
		if err == redis.Nil {
			return fmt.Errorf("不存在:%v", oldRowValue)
		} else if err != nil {
			return err
		}

		// 检查新虚拟值是否已经存在
		if _, err := rdb.Get(ctx, newRowKey).Result(); err == nil {
			return fmt.Errorf("%v :已存在", newRowValue)
		} else if err != redis.Nil {
			return err
		}

		// 更新真实值到新的虚拟值的映射
		newRowBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(newRowBytes, uint64(newRowValue))
		if err := rdb.Set(ctx, id, newRowBytes, 0).Err(); err != nil {
			return err
		}

		// 更新反向映射
		if err := rdb.Del(ctx, oldRowKey).Err(); err != nil {
			return err
		}
		if err := rdb.Set(ctx, newRowKey, id, 0).Err(); err != nil {
			return err
		}

		return nil
	})

	return err
}

// RetrieveRealValue 根据虚拟值获取真实值，并返回虚拟值及其对应的真实值
func RetrieveRealValue(virtualValue int64) (string, string, error) {
	virtualKey := fmt.Sprintf("row-%d", virtualValue)
	realValue, err := rdb.Get(ctx, virtualKey).Result()
	if err == redis.Nil {
		return "", "", fmt.Errorf("no real value found for virtual value: %d", virtualValue)
	} else if err != nil {
		return "", "", err
	}

	// 返回虚拟值和对应的真实值
	return fmt.Sprintf("%d", virtualValue), realValue, nil
}

// RetrieveVirtualValue 根据真实值获取虚拟值，并返回真实值及其对应的虚拟值
func RetrieveVirtualValue(realValue string) (string, string, error) {
	virtualValueBytes, err := rdb.Get(ctx, realValue).Bytes()
	if err == redis.Nil {
		return "", "", fmt.Errorf("no virtual value found for real value: %s", realValue)
	} else if err != nil {
		return "", "", err
	}

	virtualValue := int64(binary.BigEndian.Uint64(virtualValueBytes))

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

// UpdateVirtualValue 更新旧的虚拟值到新的虚拟值的映射
func UpdateVirtualValue(oldRowValue, newRowValue int64) error {
	oldRowKey := fmt.Sprintf("row-%d", oldRowValue)
	newRowKey := fmt.Sprintf("row-%d", newRowValue)

	// 使用事务确保操作的原子性
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// 查找旧虚拟值对应的真实值
		id, err := rdb.Get(ctx, oldRowKey).Result()
		if err == redis.Nil {
			return fmt.Errorf("不存在:%v", oldRowValue)
		} else if err != nil {
			return err
		}

		// 检查新虚拟值是否已经存在
		if _, err := rdb.Get(ctx, newRowKey).Result(); err == nil {
			return fmt.Errorf("%v :已存在", newRowValue)
		} else if err != redis.Nil {
			return err
		}

		// 更新真实值到新的虚拟值的映射
		newRowBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(newRowBytes, uint64(newRowValue))
		if err := rdb.Set(ctx, id, newRowBytes, 0).Err(); err != nil {
			return err
		}

		// 更新反向映射
		if err := rdb.Del(ctx, oldRowKey).Err(); err != nil {
			return err
		}
		if err := rdb.Set(ctx, newRowKey, id, 0).Err(); err != nil {
			return err
		}

		return nil
	})

	return err
}

// RetrieveRealValue 根据虚拟值获取真实值，并返回虚拟值及其对应的真实值
func RetrieveRealValue(virtualValue int64) (string, string, error) {
	virtualKey := fmt.Sprintf("row-%d", virtualValue)
	realValue, err := rdb.Get(ctx, virtualKey).Result()
	if err == redis.Nil {
		return "", "", fmt.Errorf("no real value found for virtual value: %d", virtualValue)
	} else if err != nil {
		return "", "", err
	}

	// 返回虚拟值和对应的真实值
	return fmt.Sprintf("%d", virtualValue), realValue, nil
}

// RetrieveVirtualValue 根据真实值获取虚拟值，并返回真实值及其对应的虚拟值
func RetrieveVirtualValue(realValue string) (string, string, error) {
	virtualValueBytes, err := rdb.Get(ctx, realValue).Bytes()
	if err == redis.Nil {
		return "", "", fmt.Errorf("no virtual value found for real value: %s", realValue)
	} else if err != nil {
		return "", "", err
	}

	virtualValue := int64(binary.BigEndian.Uint64(virtualValueBytes))

	// 返回真实值和对应的虚拟值
	return realValue, fmt.Sprintf("%d", virtualValue), nil
}

// RetrieveVirtualValuePro 根据2个真实值获取2个虚拟值
func RetrieveVirtualValuePro(realValue string, realValueSub string) (string, string, error) {
	// 拼接主键和子键
	key := fmt.Sprintf("%s:%s", realValue, realValueSub)

	// 从Redis中获取主键和子键对应的虚拟值
	virtualValue, err := rdb.HGet(ctx, "virtualValues", key).Result()
	if err == redis.Nil {
		return "", "", fmt.Errorf("no virtual value found for real values: %s, %s", realValue, realValueSub)
	} else if err != nil {
		return "", "", err
	}

	// 返回主键和子键对应的虚拟值
	return realValue, virtualValue, nil
}
// 根据2个真实值 获取2个虚拟值 群号 然后 用户号
func RetrieveVirtualValuePro(realValue string, realValueSub string) (string, string, error) {
	forwardKey := fmt.Sprintf("%s:%s", realValue, realValueSub)

	// 从Redis检索正向键对应的值
	forwardValue, err := rdb.Get(ctx, forwardKey).Result()
	if err == redis.Nil {
		return "", "", fmt.Errorf("key not found")
	} else if err != nil {
		return "", "", err
	}

	parts := strings.Split(forwardValue, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format for forward key value")
	}

	return parts[0], parts[1], nil
}

// RetrieveRealValuePro 根据两个虚拟值获取相应的两个真实值 群号 然后 用户号
func RetrieveRealValuePro(virtualValue1, virtualValue2 int64) (string, string, error) {
	compositeKey := fmt.Sprintf("%d:%d", virtualValue1, virtualValue2)

	// 从Redis中获取复合键对应的值
	compositeValue, err := rdb.Get(ctx, compositeKey).Result()
	if err == redis.Nil {
		return "", "", fmt.Errorf("no real values found for virtual values: %d, %d", virtualValue1, virtualValue2)
	} else if err != nil {
		return "", "", err
	}

	parts := strings.Split(compositeValue, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format for composite key value: %s", compositeValue)
	}

	return parts[0], parts[1], nil
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
	oldCompositeKey := fmt.Sprintf("%d:%d", oldVirtualValue1, oldVirtualValue2)
	newCompositeKey := fmt.Sprintf("%d:%d", newVirtualValue1, newVirtualValue2)

	// 使用事务确保操作的原子性
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// 检查旧复合键是否存在
		compositeValue, err := rdb.Get(ctx, oldCompositeKey).Result()
		if err == redis.Nil {
			return fmt.Errorf("不存在的复合虚拟值：%d-%d", oldVirtualValue1, oldVirtualValue2)
		} else if err != nil {
			return err
		}

		// 检查新复合键是否已经存在
		if _, err := rdb.Get(ctx, newCompositeKey).Result(); err == nil {
			return fmt.Errorf("该复合虚拟值已存在：%d-%d", newVirtualValue1, newVirtualValue2)
		} else if err != redis.Nil {
			return err
		}

		// 删除旧的复合键和正向键
		if err := rdb.Del(ctx, oldCompositeKey).Err(); err != nil {
			return err
		}
		if err := rdb.Del(ctx, compositeValue).Err(); err != nil {
			return err
		}

		// 反向键
		if err := rdb.Set(ctx, newCompositeKey, compositeValue, 0).Err(); err != nil {
			return err
		}

		// 正向键
		if err := rdb.Set(ctx, compositeValue, newCompositeKey, 0).Err(); err != nil {
			return err
		}

		return nil
	})

	return err
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
func FindKeysBySubAndType(sub string, typeSuffix string) ([]string, error) \{
	var ids []string

	// 获取所有键
	keys, err := rdb.Keys(ctx, "*").Result()
	if err != nil \{
		return nil, err
	}

	for _, key := range keys \{
		value, err := rdb.Get(ctx, key).Result()
		if err != nil \{
			continue
		}

		if strings.HasSuffix(key, typeSuffix) && value == sub \{
			id := strings.Split(key, ":")[0]
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// 取相同前缀下的所有key的:后边 比如取群成员列表
func FindSubKeysById(id string) ([]string, error) \{
	var subKeys []string
	prefix := id + ":"

	// 获取所有键
	keys, err := rdb.Keys(ctx, prefix+"*").Result()
	if err != nil \{
		return nil, err
	}

	for _, key := range keys \{
		parts := strings.Split(key, ":")
		if len(parts) == 2 \{
			subKeys = append(subKeys, parts[1])
		}
	}

	return subKeys, nil
}

// FindSubKeysByIdPro 根据1个值获取key中的k:v给出k获取所有v，通过网络调用
func FindSubKeysByIdPro(id string) ([]string, error) \{
	if config.GetLotusValue() && !config.GetLotusWithoutIdmaps() \{
		// 使用网络请求方式
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()

		// 根据portValue确定协议
		protocol := "http"
		if portValue == "443" \{
			protocol = "https"
		}

		// 构建请求URL
		url := fmt.Sprintf("%s://%s:%s/getid?type=14&id=%s", protocol, serverDir, portValue, id)
		resp, err := http.Get(url)
		if err != nil \{
			return nil, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// 解析响应
		var response map[string]interface\{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil \{
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}
		if resp.StatusCode != http.StatusOK \{
			return nil, fmt.Errorf("error response from server: %s", response["error"])
		}

		keys, ok := response["keys"].([]interface\{})
		if !ok \{
			return nil, fmt.Errorf("invalid response format for keys")
		}

		// 将interface\{}类型的keys转换为[]string
		var resultKeys []string
		for _, key := range keys \{
			if strKey, ok := key.(string); ok \{
				resultKeys = append(resultKeys, strKey)
			} else \{
				return nil, fmt.Errorf("invalid key format in response")
			}
		}

		return resultKeys, nil
	}

	// 如果lotus为假，调用本地函数
	return FindSubKeysById(id)
}

// 场景: xxx:yyy zzz:bbb  zzz:bbb xxx:yyy 把xxx(id)替换为newID 比如更换群号(会卡住)
func UpdateKeysWithNewID(id, newID string) error \{
	// 获取所有以id开头的键
	keys, err := rdb.Keys(ctx, id+":*").Result()
	if err != nil \{
		return err
	}

	// 开始事务
	_, err = rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error \{
		for _, key := range keys \{
			value, err := rdb.Get(ctx, key).Result()
			if err != nil \{
				return err
			}

			newKey := strings.Replace(key, id, newID, 1)
			reverseKey := value

			// 获取原反向键的值
			reverseValue, err := rdb.Get(ctx, reverseKey).Result()
			if err != nil \{
				return err
			}

			// 更新原键
			if err := rdb.Del(ctx, key).Err(); err != nil \{
				return err
			}
			if err := rdb.Set(ctx, newKey, reverseKey, 0).Err(); err != nil \{
				return err
			}

			// 更新反向键的值
			newReverseValue := strings.Replace(reverseValue, id, newID, 1)
			if err := rdb.Set(ctx, reverseKey, newReverseValue, 0).Err(); err != nil \{
				return err
			}
		}
		return nil
	})

	return err
}

// StoreUserInfo 存储用户信息
func StoreUserInfo(rawID string, userInfo structs.FriendData) error \{
	key := fmt.Sprintf("%s:%s", rawID, userInfo.UserID) // 创建复合键

	// 检查是否存在重复键
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil \{
		return err
	}
	if exists > 0 \{
		return fmt.Errorf("duplicate key: %s", key)
	}

	// 序列化用户信息作为值
	value, err := json.Marshal(userInfo)
	if err != nil \{
		return fmt.Errorf("could not encode user info: %s", err)
	}

	// 存储键值对
	if err := rdb.Set(ctx, key, value, 0).Err(); err != nil \{
		return fmt.Errorf("could not store user info: %s", err)
	}

	return nil
}

// ListAllUsers 返回数据库中所有用户的信息
func ListAllUsers() ([]structs.FriendData, error) {
	var users []structs.FriendData

	// 获取所有用户信息的键
	keys, err := rdb.Keys(ctx, UserInfoBucket+":*").Result()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve keys: %v", err)
	}

	for _, key := range keys {
		value, err := rdb.Get(ctx, key).Result()
		if err != nil {
			log.Printf("Error retrieving key %s: %v", key, err)
			continue
		}

		var user structs.FriendData
		if err := json.Unmarshal([]byte(value), &user); err != nil {
			log.Printf("Error unmarshaling user data for key %s: %v", key, err)
			continue
		}
		users = append(users, user)
	}

	return users, nil
}
