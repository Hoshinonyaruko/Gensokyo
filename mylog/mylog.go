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
var logChannel = make(chan EnhancedLogEntry, 1000)

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
