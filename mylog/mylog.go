package mylog

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

type Client struct {
	conn *websocket.Conn
	send chan EnhancedLogEntry
}

type MyLogAdapter struct {
	Level         LogLevel
	EnableFileLog bool
	FileLogPath   string
}

func GetLogLevelFromConfig(logLevel int) LogLevel {
	switch logLevel {
	case 0:
		return LogLevelDebug
	case 1:
		return LogLevelInfo
	case 2:
		return LogLevelWarn
	case 3:
		return LogLevelError
	default:
		return LogLevelInfo // 默认为 Info
	}
}

var logPath string

func init() {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	exeDir := filepath.Dir(exePath)
	logPath = filepath.Join(exeDir, "log")
}

// 全局变量，用于存储日志启用状态
var enableFileLogGlobal bool

// SetEnableFileLog 设置 enableFileLogGlobal 的值
func SetEnableFileLog(value bool) {
	enableFileLogGlobal = value
}

// 接收新参数，并设置文件日志路径
func NewMyLogAdapter(level LogLevel, enableFileLog bool) *MyLogAdapter {
	if enableFileLog {
		SetEnableFileLog(true)
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			err := os.Mkdir(logPath, 0755)
			if err != nil {
				panic(err)
			}
		}
	}

	return &MyLogAdapter{
		Level:         level,
		EnableFileLog: enableFileLog,
		FileLogPath:   logPath,
	}
}

// 文件日志记录函数
func (adapter *MyLogAdapter) logToFile(level, message string) {
	if !adapter.EnableFileLog {
		return
	}
	filename := time.Now().Format("2006-01-02") + ".log" // 按日期命名文件
	filepath := adapter.FileLogPath + "/" + filename

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("2006-01-02T15:04:05"), level, message)
	if _, err := file.WriteString(logEntry); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}

// 独立的文件日志记录函数
func LogToFile(level, message string) {
	if !enableFileLogGlobal {
		return
	}
	filename := time.Now().Format("2006-01-02") + ".log"
	filepath := logPath + "/" + filename

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("2006-01-02T15:04:05"), level, message)
	if _, err := file.WriteString(logEntry); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}

// Debug logs a message at the debug level.
func (adapter *MyLogAdapter) Debug(v ...interface{}) {
	if adapter.Level <= LogLevelDebug {
		message := fmt.Sprint(v...)
		Println(v...)
		adapter.logToFile("DEBUG", message)
		emitLog("DEBUG", message)
	}
}

// Info logs a message at the info level.
func (adapter *MyLogAdapter) Info(v ...interface{}) {
	if adapter.Level <= LogLevelInfo {
		message := fmt.Sprint(v...)
		Println(v...)
		adapter.logToFile("INFO", message)
		emitLog("INFO", message)
	}
}

// Warn logs a message at the warn level.
func (adapter *MyLogAdapter) Warn(v ...interface{}) {
	if adapter.Level <= LogLevelWarn {
		message := fmt.Sprint(v...)
		Printf("WARN: %v\n", v...)
		adapter.logToFile("WARN", message)
		emitLog("WARN", message)
	}
}

// Error logs a message at the error level.
func (adapter *MyLogAdapter) Error(v ...interface{}) {
	if adapter.Level <= LogLevelError {
		message := fmt.Sprint(v...)
		Printf("ERROR: %v\n", v...)
		adapter.logToFile("ERROR", message)
		emitLog("ERROR", message)
	}
}

// Debugf logs a formatted message at the debug level.
func (adapter *MyLogAdapter) Debugf(format string, v ...interface{}) {
	if adapter.Level <= LogLevelDebug {
		message := fmt.Sprintf(format, v...)
		Printf("DEBUG: "+format, v...)
		adapter.logToFile("DEBUG", message)
		emitLog("DEBUG", message)
	}
}

// Infof logs a formatted message at the info level.
func (adapter *MyLogAdapter) Infof(format string, v ...interface{}) {
	if adapter.Level <= LogLevelInfo {
		message := fmt.Sprintf(format, v...)
		Printf("INFO: "+format, v...)
		adapter.logToFile("INFO", message)
		emitLog("INFO", message)
	}
}

// Warnf logs a formatted message at the warn level.
func (adapter *MyLogAdapter) Warnf(format string, v ...interface{}) {
	if adapter.Level <= LogLevelWarn {
		message := fmt.Sprintf(format, v...)
		Printf("WARN: "+format, v...)
		adapter.logToFile("WARN", message)
		emitLog("WARN", message)
	}
}

// Errorf logs a formatted message at the error level.
func (adapter *MyLogAdapter) Errorf(format string, v ...interface{}) {
	if adapter.Level <= LogLevelError {
		message := fmt.Sprintf(format, v...)
		Printf("ERROR: "+format, v...)
		adapter.logToFile("ERROR", message)
		emitLog("ERROR", message)
	}
}

// Sync 实现 Botgo SDK 的 Sync 方法
func (adapter *MyLogAdapter) Sync() error {
	// 如果日志系统有需要执行的同步操作，在这里添加相应的代码
	// 例如，可能需要刷新或关闭文件，或执行其他清理操作
	// 如果没有特别的同步需求，可以直接返回 nil
	return nil
}

// 全局 WebSocket 客户端集合
var wsClients = make(map[*Client]bool)
var lock = sync.RWMutex{}

type EnhancedLogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// 日志频道，所有的 WebSocket 客户端都会在此监听日志事件
var logChannel = make(chan EnhancedLogEntry, 1000)

func Println(v ...interface{}) {
	log.Println(v...)
	message := fmt.Sprint(v...)
	emitLog("INFO", message)
	LogToFile("INFO", message)
}

func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
	message := fmt.Sprintf(format, v...)
	emitLog("INFO", message)
	LogToFile("INFO", message)
}

func Errorf(format string, v ...interface{}) {
	log.Printf(format, v...)
	message := fmt.Sprintf(format, v...)
	emitLog("ERROR", message)
	LogToFile("ERROR", message)
}

func Fatalf(format string, v ...interface{}) {
	log.Printf(format, v...)
	message := fmt.Sprintf(format, v...)
	emitLog("Fatal", message)
	LogToFile("Fatal", message)
}

func emitLog(level, message string) {
	entry := EnhancedLogEntry{
		Time:    time.Now().Format("2006-01-02T15:04:05"),
		Level:   level,
		Message: message,
	}
	// 非阻塞发送，如果通道满了就尝试备份日志。
	select {
	case logChannel <- entry:
		// 日志成功发送到通道。
	default:
		// 通道满了，备份日志到文件或数据库。
		//backupLog(entry)
	}
}

// 返回日志通道，以便我们的 WebSocket 服务端可以监听和广播日志事件
func LogChannel() chan EnhancedLogEntry {
	return logChannel
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WsHandlerWithDependencies(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("无法升级为websocket:", err)
		return
	}

	client := &Client{conn: ws, send: make(chan EnhancedLogEntry)}

	lock.Lock()
	wsClients[client] = true
	lock.Unlock()

	// 输出新的 WebSocket 客户端连接信息
	fmt.Println("新的webui用户已连接!")

	go client.writePump()
	go client.readPump()

	for logEntry := range LogChannel() {
		lock.RLock()
		// 去掉对wsClients长度的检查，已经在emitLog里面做了防阻塞处理
		for client := range wsClients {
			select {
			case client.send <- logEntry:
				// 成功发送日志到客户端
			default:
				// 客户端的send通道满了，可以选择断开客户端连接或者其他处理
			}
		}
		lock.RUnlock()
	}
}

func (c *Client) readPump() {
	defer func() {
		lock.Lock()
		delete(wsClients, c) // 从客户端集合中移除当前客户端
		lock.Unlock()
		c.conn.Close() // 关闭WebSocket连接
	}()

	// 设置读取超时时间
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("websocket closed unexpectedly: %v", err)
			} else {
				fmt.Println("读取websocket出错:", err)
			}
			break
		}

		// 检查收到的消息是否为心跳
		if string(message) == "heartbeat" {
			//fmt.Println("收到心跳，客户端活跃")
			c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			// 更新客户端的活跃时间，或执行其它心跳相关逻辑
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		lock.Lock()
		delete(wsClients, c) // 从客户端集合中移除当前客户端
		lock.Unlock()
		c.conn.Close() // 关闭WebSocket连接
	}()

	// 设置心跳发送间隔
	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// 如果send通道已经关闭，那么直接退出
				return
			}
			// 更新写入超时时间
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.conn.WriteJSON(message)
			if err != nil {
				// 如果写入websocket出错，输出错误并退出
				fmt.Println("发送到websocket出错:", err)
				return
			}
		case <-heartbeatTicker.C:
			// 发送心跳消息
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// 如果写入心跳失败，输出错误并退出
				fmt.Println("发送心跳失败:", err)
				return
			}
			//fmt.Println("发送心跳，维持连接活跃")
		}
	}
}
