package echo

import (
	"strconv"
	"sync"
)

type EchoMapping struct {
	mu             sync.Mutex
	msgTypeMapping map[string]string
	msgIDMapping   map[string]string
}

var globalEchoMapping = &EchoMapping{
	msgTypeMapping: make(map[string]string),
	msgIDMapping:   make(map[string]string),
}

func (e *EchoMapping) GenerateKey(appid string, s int64) string {
	return appid + "_" + strconv.FormatInt(s, 10)
}

// 添加echo对应的类型
func AddMsgType(appid string, s int64, msgType string) {
	key := globalEchoMapping.GenerateKey(appid, s)
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()
	globalEchoMapping.msgTypeMapping[key] = msgType
}

// 添加echo对应的messageid
func AddMsgID(appid string, s int64, msgID string) {
	key := globalEchoMapping.GenerateKey(appid, s)
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()
	globalEchoMapping.msgIDMapping[key] = msgID
}

// 根据给定的key获取消息类型
func GetMsgTypeByKey(key string) string {
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()
	return globalEchoMapping.msgTypeMapping[key]
}

// 根据给定的key获取消息ID
func GetMsgIDByKey(key string) string {
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()
	return globalEchoMapping.msgIDMapping[key]
}
