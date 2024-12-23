package echo

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

type messageRecord struct {
	messageID  string
	timestamp  time.Time
	usageCount int // 新增字段，记录使用次数，默认为0
}

type messageStore struct {
	records sync.Map // 使用 sync.Map 代替普通 map
}

var instance *messageStore
var once sync.Once

// 惰性初始化
func initInstance() *messageStore {
	once.Do(func() {
		instance = &messageStore{}
	})
	return instance
}

// AddLazyMessageId 添加 message_id 和它的时间戳到指定群号
func AddLazyMessageId(groupID, messageID string, timestamp time.Time) {
	store := initInstance()

	// 初始化 messageRecord
	record := messageRecord{
		messageID:  messageID,
		timestamp:  timestamp,
		usageCount: 0, // 默认使用次数为 0
	}

	// 从 sync.Map 获取现有记录
	value, ok := store.records.Load(groupID)
	if ok {
		// 如果已有记录，追加新记录
		records := value.([]messageRecord)
		store.records.Store(groupID, append(records, record))
	} else {
		// 如果没有记录，初始化新记录
		store.records.Store(groupID, []messageRecord{record})
	}
}

// AddLazyMessageIdv2 添加 message_id 和它的时间戳到指定群号
func AddLazyMessageIdv2(groupID, userID, messageID string, timestamp time.Time) {
	store := initInstance()

	// 组合键
	key := groupID + "." + userID

	// 初始化 messageRecord，并设置 usageCount 为 0
	record := messageRecord{
		messageID:  messageID,
		timestamp:  timestamp,
		usageCount: 0, // 默认使用次数为0
	}

	// 通过 sync.Map 读取现有数据
	value, ok := store.records.Load(key)
	if ok {
		// 如果已存在，追加新记录
		existingRecords := value.([]messageRecord)
		store.records.Store(key, append(existingRecords, record))
	} else {
		// 如果不存在，初始化记录
		store.records.Store(key, []messageRecord{record})
	}
}

// GetLazyMessagesId 获取指定群号中最近 4 分钟内的 message_id
func GetLazyMessagesId(groupID string) string {
	store := initInstance()

	// 获取当前时间和时间窗口
	now := time.Now()
	fourMinutesAgo := now.Add(-4 * time.Minute)

	// 从 sync.Map 获取记录
	value, ok := store.records.Load(groupID)
	if !ok {
		return generateDefaultMessageID(groupID)
	}

	// 类型断言并筛选最近 4 分钟的消息
	records, ok := value.([]messageRecord)
	if !ok || len(records) == 0 {
		return generateDefaultMessageID(groupID)
	}

	var selectedRecord *messageRecord
	var validRecords []messageRecord

	// 筛选最近 4 分钟的消息，同时找出使用次数为0且时间最近的记录
	for i := range records {
		record := records[i]
		if record.timestamp.After(fourMinutesAgo) {
			// 添加到有效记录中
			validRecords = append(validRecords, record)

			// 优先选择 usageCount == 0 且时间最近的记录
			if record.usageCount == 0 {
				if selectedRecord == nil || record.timestamp.After(selectedRecord.timestamp) {
					selectedRecord = &validRecords[len(validRecords)-1] // 指向新增的有效记录
				}
			} else if selectedRecord == nil {
				// 如果没有 usageCount == 0 的，选择时间最近的
				selectedRecord = &validRecords[len(validRecords)-1]
			}
		}
	}

	// 如果找到合适的记录，更新其 usageCount
	if selectedRecord != nil {
		selectedRecord.usageCount++
	}

	// 更新有效记录到 sync.Map（仅更新一次）
	store.records.Store(groupID, validRecords)

	// 返回选中的 messageID 或生成默认消息ID
	if selectedRecord != nil {
		return selectedRecord.messageID
	}

	return generateDefaultMessageID(groupID)
}

func GetLazyMessagesIdv2(groupID, userID string) string {
	store := initInstance()
	now := time.Now() // 统一时间基准
	fourMinutesAgo := now.Add(-4 * time.Minute)
	key := groupID + "." + userID

	// 获取记录
	value, ok := store.records.Load(key)
	if !ok {
		// 如果没有找到记录，生成默认消息ID
		return generateDefaultMessageID(groupID)
	}

	// 类型断言并检查记录是否为空
	records, ok := value.([]messageRecord)
	if !ok || len(records) == 0 {
		return generateDefaultMessageID(groupID)
	}

	// 筛选最近 4 分钟的记录并找最优记录，同时清理过期记录
	var selectedRecord *messageRecord
	var validRecords []messageRecord

	for i := range records {
		record := records[i]
		if record.timestamp.After(fourMinutesAgo) {
			// 保留有效记录
			validRecords = append(validRecords, record)

			// 优先选择 usageCount == 0 且时间最近的
			if record.usageCount == 0 {
				if selectedRecord == nil || record.timestamp.After(selectedRecord.timestamp) {
					selectedRecord = &validRecords[len(validRecords)-1] // 指向新增的有效记录
				}
			} else if selectedRecord == nil {
				// 如果没有 usageCount == 0 的，选择时间最近的
				selectedRecord = &validRecords[len(validRecords)-1] // 指向新增的有效记录
			}
		}
	}

	// 如果找到合适的记录，更新其 usageCount
	if selectedRecord != nil {
		selectedRecord.usageCount++ // 更新选中记录的 usageCount
	}

	// 更新有效记录到 sync.Map（仅更新一次）
	store.records.Store(key, validRecords)

	// 返回选中的 messageID 或生成默认消息ID
	if selectedRecord != nil {
		return selectedRecord.messageID
	}
	return generateDefaultMessageID(groupID)
}

// 生成默认消息ID的逻辑拆分为独立函数
func generateDefaultMessageID(groupID string) string {
	groupIDint64, err := idmap.StoreIDv2(groupID)
	if err != nil {
		mylog.Printf("Error storing ID: %v", err)
		return "2000"
	}
	msgType := GetMessageTypeByGroupidv2(config.GetAppIDStr(), groupIDint64)
	if strings.HasPrefix(msgType, "guild") {
		return "1000"
	}
	return "2000"
}

// 通过group_id获取类型
func GetMessageTypeByGroupidv2(appID string, GroupID interface{}) string { //2
	// 从appID和userID生成key
	var GroupIDStr string
	switch u := GroupID.(type) {
	case int:
		GroupIDStr = strconv.Itoa(u)
	case int64:
		GroupIDStr = strconv.FormatInt(u, 10)
	case string:
		GroupIDStr = u
	default:
		// 可能需要处理其他类型或报错
		return ""
	}

	key := appID + "_" + GroupIDStr
	return GetMsgTypeByKey(key)
}
