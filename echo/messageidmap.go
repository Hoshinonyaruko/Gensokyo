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

const (
	// QQ 群聊中，同一个 message_id 最多用于回复 5 次。
	lazyGroupMessageIDMaxUses = 5

	// QQ C2C 私聊中，同一个 message_id 最多用于回复 4 次。
	lazyC2CMessageIDMaxUses = 4

	// 群聊 message_id 本地有效窗口。
	lazyGroupMessageIDWindow = 4 * time.Minute

	// C2C message_id 官方有效期按 1 小时计算，
	// 本地继续保留 2 分钟安全余量。
	lazyC2CMessageIDWindow = 58 * time.Minute

	// usageCount 至少要覆盖最长的 58 分钟有效期。
	// 多保留几分钟，避免临界清理导致次数被重置。
	lazyUsageRetention = 65 * time.Minute

	lazyUsageCleanupInterval = time.Minute
)

type messageRecord struct {
	messageID string
	timestamp time.Time
}

type messageStore struct {
	records sync.Map

	// sync.Map 只能保证单次 Load/Store 安全，
	// 不能保证 Load -> 选择 -> 增加次数 -> Store 整体原子。
	mu sync.Mutex

	// 必须以 messageID 为维度统一计数。
	// 同一个 messageID 即使同时出现在 groupID 和 groupID.userID 池中，
	// 也只能共享同一个使用次数。
	usageCounts map[string]int

	// 用于定期清理已经过期的次数记录。
	usageTouchedAt map[string]time.Time

	lastUsageCleanup time.Time
}

var instance *messageStore
var once sync.Once

// 惰性初始化
func initInstance() *messageStore {
	once.Do(func() {
		instance = &messageStore{
			usageCounts:    make(map[string]int),
			usageTouchedAt: make(map[string]time.Time),
		}
	})
	return instance
}

// AddLazyMessageId 添加 message_id。
// 这个入口同时用于频道、群聊兼容池以及 QQ C2C，
// 所以按照最长的 C2C 有效窗口清理记录。
func AddLazyMessageId(groupID, messageID string, timestamp time.Time) {
	store := initInstance()
	store.addLazyMessageID(
		groupID,
		messageID,
		timestamp,
		lazyC2CMessageIDWindow,
	)
}

// AddLazyMessageIdv2 添加 groupID + userID 精确匹配的群聊 message_id。
func AddLazyMessageIdv2(
	groupID,
	userID,
	messageID string,
	timestamp time.Time,
) {
	store := initInstance()

	key := groupID + "." + userID
	store.addLazyMessageID(
		key,
		messageID,
		timestamp,
		lazyGroupMessageIDWindow,
	)
}

// addLazyMessageID 原子地添加记录，并避免重复 message_id 重置次数。
func (store *messageStore) addLazyMessageID(
	key,
	messageID string,
	timestamp time.Time,
	retention time.Duration,
) {
	if key == "" || messageID == "" {
		return
	}

	now := time.Now()
	if timestamp.IsZero() {
		timestamp = now
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	store.cleanupUsageLocked(now)

	var records []messageRecord

	if value, ok := store.records.Load(key); ok {
		if existingRecords, typeOK := value.([]messageRecord); typeOK {
			records = make(
				[]messageRecord,
				0,
				len(existingRecords)+1,
			)

			cutoffTime := now.Add(-retention)
			messageIDExists := false

			for _, record := range existingRecords {
				// 顺便清理此池中的过期记录。
				if !record.timestamp.After(cutoffTime) {
					continue
				}

				if record.messageID == messageID {
					// 防止同一个事件被重复上报后，
					// 添加出多条相同记录或重置使用次数。
					if messageIDExists {
						continue
					}
					messageIDExists = true
				}

				records = append(records, record)
			}

			if !messageIDExists {
				records = append(records, messageRecord{
					messageID: messageID,
					timestamp: timestamp,
				})
			}
		}
	}

	if records == nil {
		records = []messageRecord{
			{
				messageID: messageID,
				timestamp: timestamp,
			},
		}
	}

	store.records.Store(key, records)

	// 不修改 usageCounts。
	// 如果同一个 messageID 已经使用过，重新 Add 也不能重置次数。
	store.usageTouchedAt[messageID] = now
}

// acquireLazyMessageID 从指定池中原子地选择并占用一个 message_id。
//
// maxUses > 0：
//
//	选择时间最新且 usageCount < maxUses 的记录。
//
// preferUnused == true：
//
//	保留旧版频道逻辑，优先选择从未使用过的最新记录；
//	全部使用过后，再选择时间最新的记录。
func (store *messageStore) acquireLazyMessageID(
	key string,
	validWindow time.Duration,
	maxUses int,
	preferUnused bool,
) (string, bool) {
	now := time.Now()
	cutoffTime := now.Add(-validWindow)

	store.mu.Lock()
	defer store.mu.Unlock()

	store.cleanupUsageLocked(now)

	value, ok := store.records.Load(key)
	if !ok {
		return "", false
	}

	records, ok := value.([]messageRecord)
	if !ok || len(records) == 0 {
		store.records.Delete(key)
		return "", false
	}

	validRecords := make([]messageRecord, 0, len(records))

	// 有次数上限时使用 selectedIndex。
	selectedIndex := -1

	// 兼容旧版无硬上限选择方式。
	latestIndex := -1
	latestUnusedIndex := -1

	for _, record := range records {
		if !record.timestamp.After(cutoffTime) {
			continue
		}

		validRecords = append(validRecords, record)
		currentIndex := len(validRecords) - 1
		usageCount := store.usageCounts[record.messageID]

		if maxUses > 0 {
			// 次数已经用完，不能再作为候选。
			if usageCount >= maxUses {
				continue
			}

			// 群聊和 C2C：
			// 始终选择时间最新、且仍有剩余次数的 message_id。
			if selectedIndex == -1 ||
				validRecords[currentIndex].timestamp.After(
					validRecords[selectedIndex].timestamp,
				) {
				selectedIndex = currentIndex
			}

			continue
		}

		// 以下为旧版频道兼容逻辑。
		if latestIndex == -1 ||
			validRecords[currentIndex].timestamp.After(
				validRecords[latestIndex].timestamp,
			) {
			latestIndex = currentIndex
		}

		if usageCount == 0 {
			if latestUnusedIndex == -1 ||
				validRecords[currentIndex].timestamp.After(
					validRecords[latestUnusedIndex].timestamp,
				) {
				latestUnusedIndex = currentIndex
			}
		}
	}

	// 同时保存清理过期记录后的池。
	if len(validRecords) == 0 {
		store.records.Delete(key)
	} else {
		store.records.Store(key, validRecords)
	}

	if maxUses <= 0 {
		if preferUnused && latestUnusedIndex >= 0 {
			selectedIndex = latestUnusedIndex
		} else {
			selectedIndex = latestIndex
		}
	}

	if selectedIndex < 0 {
		// 有有效记录，但全部达到次数上限。
		return "", false
	}

	messageID := validRecords[selectedIndex].messageID

	// 在返回之前先占用一次。
	// 选择和增加次数处于同一把锁内，避免两个并发请求同时拿到第 5 次。
	store.usageCounts[messageID]++
	store.usageTouchedAt[messageID] = now

	return messageID, true
}

func (store *messageStore) cleanupUsageLocked(now time.Time) {
	if !store.lastUsageCleanup.IsZero() &&
		now.Sub(store.lastUsageCleanup) < lazyUsageCleanupInterval {
		return
	}

	cutoffTime := now.Add(-lazyUsageRetention)

	for messageID, touchedAt := range store.usageTouchedAt {
		if touchedAt.After(cutoffTime) {
			continue
		}

		delete(store.usageTouchedAt, messageID)
		delete(store.usageCounts, messageID)
	}

	store.lastUsageCleanup = now
}

// GetLazyMessagesId 保留旧版频道 lazy message_id 的选择规则。
// QQ 群聊应使用 GetLazyGroupMessagesId。
func GetLazyMessagesId(groupID string) string {
	store := initInstance()

	messageID, ok := store.acquireLazyMessageID(
		groupID,
		lazyGroupMessageIDWindow,
		0,
		true,
	)
	if ok {
		return messageID
	}

	return generateDefaultMessageID(groupID)
}

// GetLazyGroupMessagesId 获取群维度最新且仍有剩余次数的 message_id。
// 同一个 message_id 最多占用 5 次。
func GetLazyGroupMessagesId(groupID string) string {
	store := initInstance()

	messageID, ok := store.acquireLazyMessageID(
		groupID,
		lazyGroupMessageIDWindow,
		lazyGroupMessageIDMaxUses,
		false,
	)
	if ok {
		return messageID
	}

	return generateDefaultMessageID(groupID)
}

// GetLazyMessagesIdv2 获取指定群、指定用户最近产生的可用 message_id。
// 同一个 message_id 最多占用 5 次。
func GetLazyMessagesIdv2(groupID, userID string) string {
	store := initInstance()
	key := groupID + "." + userID

	messageID, ok := store.acquireLazyMessageID(
		key,
		lazyGroupMessageIDWindow,
		lazyGroupMessageIDMaxUses,
		false,
	)
	if ok {
		return messageID
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

// GetLazyC2CMessagesId 获取 QQ C2C 私聊用户最近 58 分钟内的可用 message_id。
// 同一个 message_id 最多占用 4 次。
func GetLazyC2CMessagesId(userID string) string {
	store := initInstance()

	messageID, ok := store.acquireLazyMessageID(
		userID,
		lazyC2CMessageIDWindow,
		lazyC2CMessageIDMaxUses,
		false,
	)
	if ok {
		return messageID
	}

	return generateDefaultMessageID(userID)
}
