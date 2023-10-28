package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/tencent-connect/botgo/openapi"
)

type WebSocketClient struct {
	conn           *websocket.Conn
	api            openapi.OpenAPI
	apiv2          openapi.OpenAPI
	botID          uint64
	urlStr         string
	cancel         context.CancelFunc // Add this
	mutex          sync.Mutex         // Mutex for reconnecting
	isReconnecting bool
}

// 发送json信息给onebot应用端
func (c *WebSocketClient) SendMessage(message map[string]interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}
	err = c.conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		log.Println("Error sending message:", err)
		return err
	}
	return nil
}

// 处理onebotv11应用端发来的信息
func (c *WebSocketClient) handleIncomingMessages(ctx context.Context, cancel context.CancelFunc) {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket connection closed:", err)
			cancel() // cancel heartbeat goroutine

			if !c.isReconnecting {
				go c.Reconnect()
			}
			return
		}

		go c.recvMessage(msg)
	}
}

// 断线重连
func (client *WebSocketClient) Reconnect() {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.cancel != nil {
		client.cancel() // Stop current goroutines
	}

	client.isReconnecting = true
	defer func() {
		client.isReconnecting = false
	}()

	for {
		time.Sleep(5 * time.Second)

		newClient, err := NewWebSocketClient(client.urlStr, client.botID, client.api, client.apiv2)
		if err == nil && newClient != nil {
			client.conn = newClient.conn
			client.api = newClient.api
			client.apiv2 = newClient.apiv2
			client.cancel = newClient.cancel // Update cancel function

			log.Println("Successfully reconnected to WebSocket.")
			return
		}
		log.Println("Failed to reconnect to WebSocket. Retrying in 5 seconds...")
	}
}

// 处理信息,调用腾讯api
func (c *WebSocketClient) recvMessage(msg []byte) {
	var message callapi.ActionMessage
	err := json.Unmarshal(msg, &message)
	if err != nil {
		log.Printf("Error unmarshalling message: %v, Original message: %s", err, string(msg))
		return
	}

	fmt.Println("Received from onebotv11 server:", TruncateMessage(message, 500))
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
func (c *WebSocketClient) sendHeartbeat(ctx context.Context, botID uint64) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
			message := map[string]interface{}{
				"meta_event_type": "heartbeat",
				"post_type":       "meta_event",
				"self_id":         botID,
				"status":          "ok",
				"time":            int(time.Now().Unix()),
			}
			c.SendMessage(message)
		}
	}
}

const maxRetryAttempts = 5

// NewWebSocketClient 创建 WebSocketClient 实例，接受 WebSocket URL、botID 和 openapi.OpenAPI 实例
func NewWebSocketClient(urlStr string, botID uint64, api openapi.OpenAPI, apiv2 openapi.OpenAPI) (*WebSocketClient, error) {
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
	fmt.Printf("准备使用token[%s]连接到[%s]\n", token, urlStr)
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	var conn *websocket.Conn
	var err error

	retryCount := 0
	for {
		fmt.Println("Dialing URL:", urlStr)
		conn, _, err = dialer.Dial(urlStr, headers)
		if err != nil {
			retryCount++
			if retryCount > maxRetryAttempts {
				log.Printf("Exceeded maximum retry attempts for WebSocket[%v]: %v\n", urlStr, err)
				return nil, err
			}
			fmt.Printf("Failed to connect to WebSocket[%v]: %v, retrying in 5 seconds...\n", urlStr, err)
			time.Sleep(5 * time.Second) // sleep for 5 seconds before retrying
		} else {
			fmt.Printf("Successfully connected to %s.\n", urlStr) // 输出连接成功提示
			break                                                 // successfully connected, break the loop
		}
	}

	client := &WebSocketClient{
		conn:   conn,
		api:    api,
		apiv2:  apiv2,
		botID:  botID,
		urlStr: urlStr,
	}

	// Sending initial message similar to your setupB function
	message := map[string]interface{}{
		"meta_event_type": "lifecycle",
		"post_type":       "meta_event",
		"self_id":         botID,
		"sub_type":        "connect",
		"time":            int(time.Now().Unix()),
	}

	fmt.Printf("Message: %+v\n", message)

	err = client.SendMessage(message)
	if err != nil {
		// handle error
		fmt.Printf("Error sending message: %v\n", err)
	}

	// Starting goroutine for heartbeats and another for listening to messages
	ctx, cancel := context.WithCancel(context.Background())

	client.cancel = cancel

	go client.sendHeartbeat(ctx, botID)
	go client.handleIncomingMessages(ctx, cancel)

	return client, nil
}

func (ws *WebSocketClient) Close() error {
	return ws.conn.Close()
}

// getParamsFromURI 解析给定URI中的查询参数，并返回一个映射（map）
func getParamsFromURI(uriStr string) map[string]string {
	params := make(map[string]string)

	u, err := url.Parse(uriStr)
	if err != nil {
		fmt.Printf("Error parsing the URL: %v\n", err)
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
