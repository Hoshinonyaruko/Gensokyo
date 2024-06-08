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
	"github.com/hoshinonyaruko/gensokyo/botstats"
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
	sendFailures   []map[string]interface{} // 存储失败的消息

}

// 发送json信息给onebot应用端
func (client *WebSocketClient) SendMessage(message map[string]interface{}) error {
	client.mutex.Lock()         // 在写操作之前锁定
	defer client.mutex.Unlock() // 确保在函数返回时解锁

	msgBytes, err := json.Marshal(message)
	if err != nil {
		mylog.Println("Error marshalling message:", err)
		return err
	}

	err = client.conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		mylog.Println("Error sending message:", err)
		// 发送失败，将消息添加到切片
		client.sendFailures = append(client.sendFailures, message)
		return err
	}

	return nil
}

// 处理onebotv11应用端发来的信息
func (client *WebSocketClient) handleIncomingMessages(cancel context.CancelFunc) {
	for {
		_, msg, err := client.conn.ReadMessage()
		if err != nil {
			mylog.Println("WebSocket connection closed:", err)
			cancel() // 取消心跳 goroutine
			if !client.isReconnecting {
				go client.Reconnect()
			}
			return // 退出循环，不再尝试读取消息
		}

		go client.recvMessage(msg)
	}
}

// 断线重连
func (client *WebSocketClient) Reconnect() {
	client.isReconnecting = true

	addresses := config.GetWsAddress()
	tokens := config.GetWsToken()

	var token string
	for index, address := range addresses {
		if address == client.urlStr && index < len(tokens) {
			token = tokens[index]
			break
		}
	}

	// 检查URL中是否有access_token参数
	mp := getParamsFromURI(client.urlStr)
	if val, ok := mp["access_token"]; ok {
		token = val
	}

	headers := http.Header{
		"User-Agent":    []string{"CQHttp/4.15.0"},
		"X-Client-Role": []string{"Universal"},
		"X-Self-ID":     []string{fmt.Sprintf("%d", client.botID)},
	}

	if token != "" {
		headers["Authorization"] = []string{"Token " + token}
	}
	mylog.Printf("准备使用token[%s]重新连接到[%s]\n", token, client.urlStr)
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	var conn *websocket.Conn
	var err error

	maxRetryAttempts := config.GetReconnecTimes()
	retryCount := 0
	for {
		mylog.Println("Dialing URL:", client.urlStr)
		conn, _, err = dialer.Dial(client.urlStr, headers)
		if err != nil {
			retryCount++
			if retryCount > maxRetryAttempts {
				mylog.Printf("Exceeded maximum retry attempts for WebSocket[%v]: %v\n", client.urlStr, err)
				return
			}
			mylog.Printf("Failed to connect to WebSocket[%v]: %v, retrying in 5 seconds...\n", client.urlStr, err)
			time.Sleep(5 * time.Second) // sleep for 5 seconds before retrying
		} else {
			mylog.Printf("Successfully connected to %s.\n", client.urlStr) // 输出连接成功提示
			break                                                          // successfully connected, break the loop
		}
	}
	// 复用现有的client完成重连
	client.conn = conn

	// 再次发送元事件
	message := map[string]interface{}{
		"meta_event_type": "lifecycle",
		"post_type":       "meta_event",
		"self_id":         client.botID,
		"sub_type":        "connect",
		"time":            int(time.Now().Unix()),
	}

	mylog.Printf("Message: %+v\n", message)

	err = client.SendMessage(message)
	if err != nil {
		// handle error
		mylog.Printf("Error sending message: %v\n", err)
	}

	//退出老的sendHeartbeat和handleIncomingMessages
	client.cancel()

	// Starting goroutine for heartbeats and another for listening to messages
	ctx, cancel := context.WithCancel(context.Background())

	client.cancel = cancel
	heartbeatinterval := config.GetHeartBeatInterval()
	go client.sendHeartbeat(ctx, client.botID, heartbeatinterval)
	go client.handleIncomingMessages(cancel)

	defer func() {
		client.isReconnecting = false
	}()

	mylog.Printf("Successfully reconnected to WebSocket.")

}

// 处理发送失败的消息
func (client *WebSocketClient) processFailedMessages() {
	for _, failedMessage := range client.sendFailures {
		// 尝试重新发送消息
		err := client.SendMessage(failedMessage)
		if err != nil {
			mylog.Printf("Error resending message: %v\n", err)
		}
	}
	// 清空失败消息列表
	client.sendFailures = []map[string]interface{}{}
}

// 处理信息,调用腾讯api
func (client *WebSocketClient) recvMessage(msg []byte) {
	var message callapi.ActionMessage
	//mylog.Println("Received from onebotv11 server raw:", string(msg))
	err := json.Unmarshal(msg, &message)
	if err != nil {
		mylog.Printf("Error unmarshalling message: %v, Original message: %s", err, string(msg))
		return
	}
	mylog.Println("Received from onebotv11 server:", TruncateMessage(message, 800))
	// 调用callapi
	go callapi.CallAPIFromDict(client, client.api, client.apiv2, message)
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
func (client *WebSocketClient) sendHeartbeat(ctx context.Context, botID uint64, heartbeatinterval int) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(heartbeatinterval) * time.Second):
			messageReceived, messageSent, lastMessageTime, err := botstats.GetStats()
			if err != nil {
				mylog.Printf("心跳错误,获取机器人发信状态错误:%v", err)
			}
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
						"message_received":  messageReceived,
						"message_sent":      messageSent,
						"disconnect_times":  0,
						"lost_times":        0,
						"last_message_time": int(lastMessageTime),
					},
				},
				"interval": 5000, // 以毫秒为单位
			}
			client.SendMessage(message)
			// 重发失败的消息
			client.processFailedMessages()
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
		sendFailures: []map[string]interface{}{},
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
	go client.handleIncomingMessages(cancel)

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
