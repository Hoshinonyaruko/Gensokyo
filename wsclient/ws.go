package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

type WebSocketClient struct {
	conn           *websocket.Conn
	api            openapi.OpenAPI
	apiv2          openapi.OpenAPI
	botID          uint64
	urlStr         string
	cancel         context.CancelFunc
	mutex          sync.Mutex // 用于同步写入和重连操作的互斥锁
	isReconnecting bool
	sendFailures   chan map[string]interface{}
}

// 发送json信息给onebot应用端
func (c *WebSocketClient) SendMessage(message map[string]interface{}) error {
	c.mutex.Lock()         // 在写操作之前锁定
	defer c.mutex.Unlock() // 确保在函数返回时解锁

	msgBytes, err := json.Marshal(message)
	if err != nil {
		mylog.Println("Error marshalling message:", err)
		return err
	}

	err = c.conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		mylog.Println("Error sending message:", err)
		// 发送失败，将消息放入channel
		go func() {
			c.sendFailures <- message
		}()
		return err
	}

	return nil
}

// 处理onebotv11应用端发来的信息
func (c *WebSocketClient) handleIncomingMessages(ctx context.Context, cancel context.CancelFunc) {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			mylog.Println("WebSocket connection closed:", err)
			cancel() // 取消心跳 goroutine
			if !c.isReconnecting {
				go c.Reconnect()
			}
			return // 退出循环，不再尝试读取消息
		}

		go c.recvMessage(msg)
	}
}

// 断线重连
func (client *WebSocketClient) Reconnect() {
	client.mutex.Lock()
	if client.isReconnecting {
		client.mutex.Unlock()
		return // 如果已经有其他携程在重连了，就直接返回
	}

	// 暂存旧的 sendFailures channel，不要关闭它
	oldSendFailures := client.sendFailures

	client.isReconnecting = true
	client.mutex.Unlock()

	defer func() {
		client.mutex.Lock()
		client.isReconnecting = false
		client.mutex.Unlock()
	}()
	reconnecttimes := config.GetReconnecTimes()
	newClient, err := NewWebSocketClient(client.urlStr, client.botID, client.api, client.apiv2, reconnecttimes)
	if err == nil && newClient != nil {
		client.mutex.Lock() // 在替换连接之前锁定
		client.cancel()     // 退出旧的连接
		client.conn = newClient.conn
		client.api = newClient.api
		client.apiv2 = newClient.apiv2
		client.cancel = newClient.cancel // 更新取消函数
		client.mutex.Unlock()
		// 重发失败的消息
		newClient.processFailedMessages(oldSendFailures)
		mylog.Println("Successfully reconnected to WebSocket.")
		return
	}

}

// 处理发送失败的消息
func (client *WebSocketClient) processFailedMessages(failuresChan chan map[string]interface{}) {
	for failedMessage := range failuresChan {
		err := client.SendMessage(failedMessage)
		if err != nil {
			// 处理重发失败的情况，比如重新放入 channel 或记录日志
			mylog.Println("Error re-sending message:", err)
		}
	}
}

// 处理信息,调用腾讯api
func (c *WebSocketClient) recvMessage(msg []byte) {
	var message callapi.ActionMessage
	//mylog.Println("Received from onebotv11 server raw:", string(msg))
	err := json.Unmarshal(msg, &message)
	if err != nil {
		mylog.Printf("Error unmarshalling message: %v, Original message: %s", err, string(msg))
		return
	}
	mylog.Println("Received from onebotv11 server:", TruncateMessage(message, 800))
	// 调用callapi
	callapi.CallAPIFromDict(c, c.api, c.apiv2, message)
}

// 截断信息
func TruncateMessage(message callapi.ActionMessage, maxLength int) string {
	paramsStr, err := json.Marshal(message.Params)
	if err != nil {
		return "Error marshalling Params for truncation."
	}

	// Truncate Params if its length exceeds maxLength
	truncatedParams := string(paramsStr)
	if len(truncatedParams) > maxLength {
		truncatedParams = truncatedParams[:maxLength] + "..."
	}

	return fmt.Sprintf("Action: %s, Params: %s, Echo: %v", message.Action, truncatedParams, message.Echo)
}

// 发送心跳包
func (c *WebSocketClient) sendHeartbeat(ctx context.Context, botID uint64, heartbeatinterval int) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(heartbeatinterval) * time.Second):
			message := map[string]interface{}{
				"post_type":       "meta_event",
				"meta_event_type": "heartbeat",
				"time":            int(time.Now().Unix()),
				"self_id":         botID,
				"status": map[string]interface{}{
					"app_enabled":     true,
					"app_good":        true,
					"app_initialized": true,
					"good":            true,
					"online":          true,
					"plugins_good":    nil,
					"stat": map[string]int{
						"packet_received":   34933,
						"packet_sent":       8513,
						"packet_lost":       0,
						"message_received":  24674,
						"message_sent":      1663,
						"disconnect_times":  0,
						"lost_times":        0,
						"last_message_time": int(time.Now().Unix()) - 10, // 假设最后一条消息是10秒前收到的
					},
				},
				"interval": 10000, // 以毫秒为单位
			}
			c.SendMessage(message)
		}
	}
}

// NewWebSocketClient 创建 WebSocketClient 实例，接受 WebSocket URL、botID 和 openapi.OpenAPI 实例
func NewWebSocketClient(urlStr string, botID uint64, api openapi.OpenAPI, apiv2 openapi.OpenAPI, maxRetryAttempts int) (*WebSocketClient, error) {
	addresses := config.GetWsAddress()
	tokens := config.GetWsToken()

	var token string
	for index, address := range addresses {
		if address == urlStr && index < len(tokens) {
			token = tokens[index]
			break
		}
	}

	// 检查URL中是否有access_token参数
	mp := getParamsFromURI(urlStr)
	if val, ok := mp["access_token"]; ok {
		token = val
	}

	headers := http.Header{
		"User-Agent":    []string{"CQHttp/4.15.0"},
		"X-Client-Role": []string{"Universal"},
		"X-Self-ID":     []string{fmt.Sprintf("%d", botID)},
	}

	if token != "" {
		headers["Authorization"] = []string{"Token " + token}
	}
	mylog.Printf("准备使用token[%s]连接到[%s]\n", token, urlStr)
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	var conn *websocket.Conn
	var err error

	retryCount := 0
	for {
		mylog.Println("Dialing URL:", urlStr)
		conn, _, err = dialer.Dial(urlStr, headers)
		if err != nil {
			retryCount++
			if retryCount > maxRetryAttempts {
				mylog.Printf("Exceeded maximum retry attempts for WebSocket[%v]: %v\n", urlStr, err)
				return nil, err
			}
			mylog.Printf("Failed to connect to WebSocket[%v]: %v, retrying in 5 seconds...\n", urlStr, err)
			time.Sleep(5 * time.Second) // sleep for 5 seconds before retrying
		} else {
			mylog.Printf("Successfully connected to %s.\n", urlStr) // 输出连接成功提示
			break                                                   // successfully connected, break the loop
		}
	}
	client := &WebSocketClient{
		conn:         conn,
		api:          api,
		apiv2:        apiv2,
		botID:        botID,
		urlStr:       urlStr,
		sendFailures: make(chan map[string]interface{}, 100),
	}

	// Sending initial message similar to your setupB function
	message := map[string]interface{}{
		"meta_event_type": "lifecycle",
		"post_type":       "meta_event",
		"self_id":         botID,
		"sub_type":        "connect",
		"time":            int(time.Now().Unix()),
	}

	mylog.Printf("Message: %+v\n", message)

	err = client.SendMessage(message)
	if err != nil {
		// handle error
		mylog.Printf("Error sending message: %v\n", err)
	}

	// Starting goroutine for heartbeats and another for listening to messages
	ctx, cancel := context.WithCancel(context.Background())

	client.cancel = cancel
	heartbeatinterval := config.GetHeartBeatInterval()
	go client.sendHeartbeat(ctx, botID, heartbeatinterval)
	go client.handleIncomingMessages(ctx, cancel)

	return client, nil
}

func (ws *WebSocketClient) Close() error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.Close()
}

// getParamsFromURI 解析给定URI中的查询参数，并返回一个映射（map）
func getParamsFromURI(uriStr string) map[string]string {
	params := make(map[string]string)

	u, err := url.Parse(uriStr)
	if err != nil {
		mylog.Printf("Error parsing the URL: %v\n", err)
		return params
	}

	// 遍历查询参数并将其添加到返回的映射中
	for key, values := range u.Query() {
		if len(values) > 0 {
			params[key] = values[0] // 如果一个参数有多个值，这里只选择第一个。可以根据需求进行调整。
		}
	}

	return params
}
