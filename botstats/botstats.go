package botstats

import (
	"log"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

var db *bbolt.DB

const (
	bucketName = "stats"
)

func InitializeDB() {
	var err error
	db, err = bbolt.Open("botstats.db", 0600, nil)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return nil
	})
}

const (
	messageReceivedKey = "messageReceived"
	messageSentKey     = "messageSent"
	lastMessageTimeKey = "lastMessageTime"
)

func RecordMessageReceived() {
	recordStats(1, 0)
}

func RecordMessageSent() {
	recordStats(0, 1)
}

// 收到增量 发出增量
func recordStats(receivedIncrement int, sentIncrement int) {
	db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		now := time.Now()
		today := now.Format("2006-01-02")

		// Reset stats if not the current day
		lastTimeBytes := b.Get([]byte(lastMessageTimeKey))
		if lastTimeBytes != nil && !strings.HasPrefix(string(lastTimeBytes), today) {
			b.Put([]byte(messageReceivedKey), []byte("0"))
			b.Put([]byte(messageSentKey), []byte("0"))
		}

		updateCounter(b, messageReceivedKey, receivedIncrement)
		updateCounter(b, messageSentKey, sentIncrement)
		b.Put([]byte(lastMessageTimeKey), []byte(today+" "+now.Format(time.RFC3339)))

		return nil
	})
}

func updateCounter(b *bbolt.Bucket, key string, increment int) {
	currentValueBytes := b.Get([]byte(key))
	currentValue := 0
	if currentValueBytes != nil {
		currentValue, _ = strconv.Atoi(string(currentValueBytes))
	}
	newValue := currentValue + increment
	b.Put([]byte(key), []byte(strconv.Itoa(newValue)))
}

// 获取 收到 发出 上次收到Time 错误
func GetStats() (int, int, int64, error) {
	var messageReceived, messageSent int
	var lastMessageTime int64
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		messageReceived = getInt(b, messageReceivedKey)
		messageSent = getInt(b, messageSentKey)
		lastMessageTime = getLastMessageTime(b)
		return nil
	})
	return messageReceived, messageSent, lastMessageTime, err
}

func getInt(b *bbolt.Bucket, key string) int {
	valueBytes := b.Get([]byte(key))
	value, _ := strconv.Atoi(string(valueBytes))
	return value
}

func getLastMessageTime(b *bbolt.Bucket) int64 {
	lastTimeBytes := b.Get([]byte("lastMessageTimeKey")) // 确保使用正确的键
	if lastTimeBytes == nil {
		return 0 // 如果键不存在或值为空，直接返回0
	}

	// 将字节切片转换为字符串，并尝试按空格分割
	splitResult := strings.Split(string(lastTimeBytes), " ")
	if len(splitResult) < 2 {
		return 0 // 如果没有足够的分割结果，返回0
	}

	// 安全地解析时间
	lastTime, err := time.Parse(time.RFC3339, splitResult[1])
	if err != nil {
		return 0 // 如果解析时间出错，返回0
	}

	return lastTime.Unix() // 返回Unix时间戳
}

func CloseDB() {
	db.Close()
}
