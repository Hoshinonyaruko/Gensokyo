package mylog

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	send chan EnhancedLogEntry
}

// 全局 WebSocket 客户端集合
var wsClients = make(map[*Client]bool)
var lock = sync.RWMutex{}

type EnhancedLogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// 我们的日志频道，所有的 WebSocket 客户端都会在此监听日志事件
var logChannel = make(chan EnhancedLogEntry, 100)

func Println(v ...interface{}) {
	log.Println(v...)
	message := fmt.Sprint(v...)
	emitLog("INFO", message)
}

func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
	message := fmt.Sprintf(format, v...)
	emitLog("INFO", message)
}

func emitLog(level, message string) {
	entry := EnhancedLogEntry{
		Time:    time.Now().Format("2006-01-02T15:04:05"),
		Level:   level,
		Message: message,
	}
	logChannel <- entry
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
	fmt.Println("新的WebSocket客户端已连接!")

	// Start a goroutine for heartbeats
	go func() {
		for {
			time.Sleep(5 * time.Second) // send a heartbeat every 30 seconds
			client.conn.WriteMessage(websocket.PingMessage, nil)
		}
	}()

	go client.writePump()

	for logEntry := range LogChannel() {
		lock.RLock()
		if len(wsClients) == 0 {
			lock.RUnlock()
			continue
		}

		for client := range wsClients {
			client.send <- logEntry
		}
		lock.RUnlock()
	}
}

func (c *Client) writePump() {
	defer func() {
		lock.Lock()
		delete(wsClients, c)
		lock.Unlock()
		c.conn.Close()
	}()

	ticker := time.NewTicker(30 * time.Second) // 每30秒发送一次心跳
	defer ticker.Stop()

	lastActiveTime := time.Now() // 上次活跃的时间

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// 如果send通道已经关闭，那么直接返回
				return
			}
			err := c.conn.WriteJSON(message)
			if err != nil {
				fmt.Println("发送到websocket出错:", err)
				return
			} else {
				// 输出消息发送成功的信息
				fmt.Printf("消息成功发送给客户端: %s\n", message)
			}

		case <-ticker.C: // 定时器触发
			if time.Since(lastActiveTime) > 1*time.Minute {
				// 如果超过1分钟没有收到Pong消息，则关闭连接
				return
			}
			c.conn.WriteMessage(websocket.PingMessage, nil)

		default:
			messageType, _, err := c.conn.ReadMessage()
			if err != nil {
				return
			}
			if messageType == websocket.PongMessage {
				// 更新上次活跃的时间
				lastActiveTime = time.Now()
			}
		}
	}
}
