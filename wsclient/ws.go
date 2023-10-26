package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/openapi"
)

type WebSocketClient struct {
	conn  *websocket.Conn
	api   openapi.OpenAPI
	apiv2 openapi.OpenAPI
	appid uint64
}

// 获取appid
func (c *WebSocketClient) GetAppID() uint64 {
	return c.appid
}

// 获取appid的字符串形式
func (c *WebSocketClient) GetAppIDStr() string {
	return fmt.Sprintf("%d", c.appid)
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
			break
		}

		go c.recvMessage(msg)
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

	fmt.Println("Received from onebotv11:", truncateMessage(message, 500))
	// 调用callapi
	callapi.CallAPIFromDict(c, c.api, c.apiv2, message)
}

// 截断信息
func truncateMessage(message callapi.ActionMessage, maxLength int) string {
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

// NewWebSocketClient 创建 WebSocketClient 实例，接受 WebSocket URL、botID 和 openapi.OpenAPI 实例
func NewWebSocketClient(urlStr string, botID uint64, api openapi.OpenAPI, apiv2 openapi.OpenAPI) (*WebSocketClient, error) {
	headers := http.Header{
		"User-Agent":    []string{"CQHttp/4.15.0"},
		"X-Client-Role": []string{"Universal"},
		"X-Self-ID":     []string{fmt.Sprintf("%d", botID)},
	}

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	var conn *websocket.Conn
	var err error

	// Retry mechanism
	for {
		fmt.Println("Dialing URL:", urlStr)
		conn, _, err = dialer.Dial(urlStr, headers)
		if err != nil {
			fmt.Printf("Failed to connect to WebSocket[%v]: %v, retrying in 5 seconds...\n", urlStr, err)
			time.Sleep(5 * time.Second) // sleep for 5 seconds before retrying
		} else {
			fmt.Printf("成功连接到 %s.\n", urlStr) // 输出连接成功提示
			break                             // successfully connected, break the loop
		}
	}

	client := &WebSocketClient{conn: conn, api: api, apiv2: apiv2, appid: botID}

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

	go client.sendHeartbeat(ctx, botID)
	go client.handleIncomingMessages(ctx, cancel) //包含收到信息,调用api部分的代码

	return client, nil
}

func (ws *WebSocketClient) Close() error {
	return ws.conn.Close()
}
