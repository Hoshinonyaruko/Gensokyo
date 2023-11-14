package echo

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
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
		msgType := GetMessageIDByUseridOrGroupidv2(config.GetAppIDStr(), groupID)
		if strings.HasPrefix(msgType, "guild") {
			randomMessageID = "1000" // 频道主动信息
		} else {
			randomMessageID = ""
		}
	}
	return randomMessageID
}

// 通过user_id获取messageID
func GetMessageIDByUseridOrGroupidv2(appID string, userID interface{}) string {
	// 从appID和userID生成key
	var userIDStr string
	switch u := userID.(type) {
	case int:
		userIDStr = strconv.Itoa(u)
	case int64:
		userIDStr = strconv.FormatInt(u, 10)
	case float64:
		userIDStr = strconv.FormatFloat(u, 'f', 0, 64)
	case string:
		userIDStr = u
	default:
		// 可能需要处理其他类型或报错
		return ""
	}
	//将真实id转为int
	userid64, err := idmap.StoreIDv2(userIDStr)
	if err != nil {
		log.Fatalf("Error storing ID 241: %v", err)
	}
	key := appID + "_" + fmt.Sprint(userid64)
	return GetMsgIDByKey(key)
}
