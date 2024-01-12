package echo

import (
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/tencent-connect/botgo/dto"
)

type EchoMapping struct {
	mu             sync.Mutex
	msgTypeMapping map[string]string
	msgIDMapping   map[string]string
}

// Int64ToIntMapping 用于存储 int64 到 int 的映射(递归计数器)
type Int64ToIntMapping struct {
	mu      sync.Mutex
	mapping map[int64]int
}

// IntToStringMappingSeq 用于存储 string 到 int 的映射(seq对应)
type StringToIntMappingSeq struct {
	mu      sync.Mutex
	mapping map[string]int
}

// StringToInt64MappingSeq 用于存储 string 到 int64 的映射(file接口频率限制)
type StringToInt64MappingSeq struct {
	mu      sync.Mutex
	mapping map[string]int64
}

// Int64Stack 用于存储 int64 的栈
type Int64Stack struct {
	mu    sync.Mutex
	stack []int64
}

// MessageGroupPair 用于存储 group 和 groupMessage
type MessageGroupPair struct {
	Group        string
	GroupMessage *dto.MessageToCreate
}

// 定义全局栈的结构体
type globalMessageGroup struct {
	mu    sync.Mutex
	stack []MessageGroupPair
}

// 初始化一个全局栈实例
var globalMessageGroupStack = &globalMessageGroup{
	stack: make([]MessageGroupPair, 0),
}

// 定义一个全局的 Int64Stack 实例
var globalInt64Stack = &Int64Stack{
	stack: make([]int64, 0),
}

var globalEchoMapping = &EchoMapping{
	msgTypeMapping: make(map[string]string),
	msgIDMapping:   make(map[string]string),
}
var globalInt64ToIntMapping = &Int64ToIntMapping{
	mapping: make(map[int64]int),
}

var globalStringToIntMappingSeq = &StringToIntMappingSeq{
	mapping: make(map[string]int),
}

var globalStringToInt64MappingSeq = &StringToInt64MappingSeq{
	mapping: make(map[string]int64),
}

func (e *EchoMapping) GenerateKey(appid string, s int64) string {
	return appid + "_" + strconv.FormatInt(s, 10)
}

func (e *EchoMapping) GenerateKeyv2(appid string, groupid int64, userid int64) string {
	return appid + "_" + strconv.FormatInt(groupid, 10) + "_" + strconv.FormatInt(userid, 10)
}

func (e *EchoMapping) GenerateKeyv3(appid string, s string) string {
	return appid + "_" + s
}

// 添加echo对应的类型
func AddMsgType(appid string, s int64, msgType string) {
	key := globalEchoMapping.GenerateKey(appid, s)
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()
	globalEchoMapping.msgTypeMapping[key] = msgType
}

// 添加echo对应的messageid
func AddMsgIDv3(appid string, s string, msgID string) {
	key := globalEchoMapping.GenerateKeyv3(appid, s)
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()
	globalEchoMapping.msgIDMapping[key] = msgID
}

// GetMsgIDv3 返回给定appid和s的msgID
func GetMsgIDv3(appid string, s string) string {
	key := globalEchoMapping.GenerateKeyv3(appid, s)
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()

	return globalEchoMapping.msgIDMapping[key]
}

// 添加group和userid对应的messageid
func AddMsgIDv2(appid string, groupid int64, userid int64, msgID string) {
	key := globalEchoMapping.GenerateKeyv2(appid, groupid, userid)
	globalEchoMapping.mu.Lock()
	defer globalEchoMapping.mu.Unlock()
	globalEchoMapping.msgIDMapping[key] = msgID
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

// AddMapping 添加一个新的映射
func AddMapping(key int64, value int) {
	globalInt64ToIntMapping.mu.Lock()
	defer globalInt64ToIntMapping.mu.Unlock()
	globalInt64ToIntMapping.mapping[key] = value
}

// GetMapping 根据给定的 int64 键获取映射值
func GetMapping(key int64) int {
	globalInt64ToIntMapping.mu.Lock()
	defer globalInt64ToIntMapping.mu.Unlock()
	return globalInt64ToIntMapping.mapping[key]
}

// AddMapping 添加一个新的映射
func AddMappingSeq(key string, value int) {
	globalStringToIntMappingSeq.mu.Lock()
	defer globalStringToIntMappingSeq.mu.Unlock()
	globalStringToIntMappingSeq.mapping[key] = value
}

// GetMappingSeq 根据给定的 string 键获取映射值
func GetMappingSeq(key string) int {
	if config.GetRamDomSeq() {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		return rng.Intn(10000) + 1 // 生成 1 到 10000 的随机数
	} else {
		globalStringToIntMappingSeq.mu.Lock()
		defer globalStringToIntMappingSeq.mu.Unlock()
		return globalStringToIntMappingSeq.mapping[key]
	}
}

// AddMapping 添加一个新的映射
func AddMappingFileTimeLimit(key string, value int64) {
	globalStringToInt64MappingSeq.mu.Lock()
	defer globalStringToInt64MappingSeq.mu.Unlock()
	globalStringToInt64MappingSeq.mapping[key] = value
}

// GetMapping 根据给定的 string 键获取映射值
func GetMappingFileTimeLimit(key string) int64 {
	globalStringToInt64MappingSeq.mu.Lock()
	defer globalStringToInt64MappingSeq.mu.Unlock()
	return globalStringToInt64MappingSeq.mapping[key]
}

// Add 添加一个新的 int64 到全局栈中
func AddFileTimeLimit(value int64) {
	globalInt64Stack.mu.Lock()
	defer globalInt64Stack.mu.Unlock()

	// 添加新元素到栈顶
	globalInt64Stack.stack = append(globalInt64Stack.stack, value)

	// 如果栈的大小超过 10，移除最早添加的元素
	if len(globalInt64Stack.stack) > 10 {
		globalInt64Stack.stack = globalInt64Stack.stack[1:]
	}
}

// Get 获取全局栈顶的元素
func GetFileTimeLimit() int64 {
	globalInt64Stack.mu.Lock()
	defer globalInt64Stack.mu.Unlock()

	if len(globalInt64Stack.stack) == 0 {
		return 0 // 当栈为空时返回 0
	}
	return globalInt64Stack.stack[len(globalInt64Stack.stack)-1]
}

// PushGlobalStack 向全局栈中添加一个新的 MessageGroupPair
func PushGlobalStack(pair MessageGroupPair) {
	globalMessageGroupStack.mu.Lock()
	defer globalMessageGroupStack.mu.Unlock()
	globalMessageGroupStack.stack = append(globalMessageGroupStack.stack, pair)
}

// PopGlobalStackMulti 从全局栈中取出指定数量的 MessageGroupPair，但不删除它们
func PopGlobalStackMulti(count int) []MessageGroupPair {
	globalMessageGroupStack.mu.Lock()
	defer globalMessageGroupStack.mu.Unlock()

	if count == 0 || len(globalMessageGroupStack.stack) == 0 {
		return nil
	}

	if count > len(globalMessageGroupStack.stack) {
		count = len(globalMessageGroupStack.stack)
	}

	return globalMessageGroupStack.stack[:count]
}

// RemoveFromGlobalStack 从全局栈中删除指定下标的元素
func RemoveFromGlobalStack(index int) {
	globalMessageGroupStack.mu.Lock()
	defer globalMessageGroupStack.mu.Unlock()

	if index < 0 || index >= len(globalMessageGroupStack.stack) {
		return // 下标越界检查
	}

	globalMessageGroupStack.stack = append(globalMessageGroupStack.stack[:index], globalMessageGroupStack.stack[index+1:]...)
}
