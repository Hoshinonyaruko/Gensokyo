package echo

import (
	"math/rand"
	"sync"
	"time"
)

type messageRecord struct {
	messageID string
	timestamp time.Time
}

type messageStore struct {
	mu      sync.RWMutex
	records map[string][]messageRecord
}

var instance *messageStore
var once sync.Once

// 惰性初始化
func initInstance() *messageStore {
	once.Do(func() {
		instance = &messageStore{
			records: make(map[string][]messageRecord),
		}
	})
	return instance
}

// AddLazyMessageId 添加 message_id 和它的时间戳到指定群号
func AddLazyMessageId(groupID, messageID string, timestamp time.Time) {
	store := initInstance()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.records[groupID] = append(store.records[groupID], messageRecord{messageID: messageID, timestamp: timestamp})
}

// GetRecentMessages 获取指定群号中最近5分钟内的 message_id
func GetLazyMessagesId(groupID string) string {
	store := initInstance()
	store.mu.RLock()
	defer store.mu.RUnlock()

	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	var recentMessages []string
	for _, record := range store.records[groupID] {
		if record.timestamp.After(fiveMinutesAgo) {
			recentMessages = append(recentMessages, record.messageID)
		}
	}
	var randomMessageID string
	if len(recentMessages) > 0 {
		randomIndex := rand.Intn(len(recentMessages))
		randomMessageID = recentMessages[randomIndex]
	} else {
		randomMessageID = ""
	}
	return randomMessageID
}
