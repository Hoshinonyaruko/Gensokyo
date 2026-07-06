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
	"github.com/hoshinonyaruko/gensokyo/botstats"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

type WebSocketClient struct {
	conn   *websocket.Conn
	connMu sync.RWMutex

	api    openapi.OpenAPI
	apiv2  openapi.OpenAPI
	botID  uint64
	urlStr string

	cancel   context.CancelFunc
	cancelMu sync.Mutex

	reconnectMu    sync.Mutex
	isReconnecting bool

	sendFailures   [][]byte
	sendFailuresMu sync.Mutex

	writeCh chan writeRequest
	closeCh chan struct{}

	closeOnce sync.Once
}

type writeRequest struct {
	messageType int
	data        []byte
	errCh       chan error
}

func (client *WebSocketClient) getConn() *websocket.Conn {
	client.connMu.RLock()
	defer client.connMu.RUnlock()
	return client.conn
}

func (client *WebSocketClient) swapConn(conn *websocket.Conn) *websocket.Conn {
	client.connMu.Lock()
	defer client.connMu.Unlock()

	oldConn := client.conn
	client.conn = conn
	return oldConn
}

func (client *WebSocketClient) setCancel(cancel context.CancelFunc) {
	client.cancelMu.Lock()
	defer client.cancelMu.Unlock()
	client.cancel = cancel
}

func (client *WebSocketClient) cancelCurrent() {
	client.cancelMu.Lock()
	cancel := client.cancel
	client.cancel = nil
	client.cancelMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (client *WebSocketClient) isClosed() bool {
	select {
	case <-client.closeCh:
		return true
	default:
		return false
	}
}

const (
	wsWriteTimeout     = 10 * time.Second
	sendEnqueueTimeout = 3 * time.Second
	maxSendFailures    = 1000
)

func (client *WebSocketClient) SendMessage(message map[string]interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}

	return client.sendRaw(websocket.TextMessage, msgBytes)
}

func (client *WebSocketClient) sendRaw(messageType int, data []byte) error {
	if client == nil {
		return fmt.Errorf("websocket client is nil")
	}

	errCh := make(chan error, 1)
	req := writeRequest{
		messageType: messageType,
		data:        data,
		errCh:       errCh,
	}

	select {
	case <-client.closeCh:
		return fmt.Errorf("websocket client closed")
	case client.writeCh <- req:
	case <-time.After(sendEnqueueTimeout):
		return fmt.Errorf("websocket write queue timeout")
	}

	select {
	case err := <-errCh:
		return err
	case <-client.closeCh:
		return fmt.Errorf("websocket client closed")
	case <-time.After(wsWriteTimeout + time.Second):
		return fmt.Errorf("websocket write result timeout")
	}
}

// Close 关闭 WebSocketClient，停止写 Goroutine
func (client *WebSocketClient) Close() error {
	if client == nil {
		return nil
	}

	var closeErr error

	client.closeOnce.Do(func() {
		client.cancelCurrent()

		close(client.closeCh)

		conn := client.swapConn(nil)
		if conn != nil {
			closeErr = conn.Close()
		}
	})

	return closeErr
}

func (client *WebSocketClient) startWriter() {
	for {
		select {
		case <-client.closeCh:
			return

		case req := <-client.writeCh:
			if client.isClosed() {
				if req.errCh != nil {
					select {
					case req.errCh <- fmt.Errorf("websocket client closed"):
					default:
					}
				}
				return
			}

			err := client.writeRawNow(req.messageType, req.data)

			if err != nil {
				log.Println("Error sending message:", err)
				client.recordSendFailure(req.data)

				if !client.isClosed() {
					go client.Reconnect()
				}
			}

			if req.errCh != nil {
				select {
				case req.errCh <- err:
				default:
				}
			}
		}
	}
}

func (client *WebSocketClient) writeRawNow(messageType int, data []byte) error {
	conn := client.getConn()
	if conn == nil {
		return fmt.Errorf("websocket connection is nil")
	}

	if wsWriteTimeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
		defer conn.SetWriteDeadline(time.Time{})
	}

	return conn.WriteMessage(messageType, data)
}

// 处理onebotv11应用端发来的信息
func (client *WebSocketClient) handleIncomingMessages(cancel context.CancelFunc) {
	conn := client.getConn()
	if conn == nil {
		mylog.Println("WebSocket connection is nil, reconnecting")
		if cancel != nil {
			cancel()
		}
		go client.Reconnect()
		return
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			mylog.Println("WebSocket connection closed:", err)
			if cancel != nil {
				cancel()
			}
			go client.Reconnect()
			return
		}

		go client.recvMessage(msg)
	}
}

// 断线重连
func (client *WebSocketClient) Reconnect() {
	if client == nil || client.isClosed() {
		return
	}

	client.reconnectMu.Lock()
	if client.isReconnecting {
		client.reconnectMu.Unlock()
		return
	}
	client.isReconnecting = true
	client.reconnectMu.Unlock()

	defer func() {
		client.reconnectMu.Lock()
		client.isReconnecting = false
		client.reconnectMu.Unlock()
	}()

	// 先停止旧心跳，并断开旧连接，避免重连期间旧 goroutine 继续写旧连接。
	client.cancelCurrent()

	oldConn := client.swapConn(nil)
	if oldConn != nil {
		_ = oldConn.Close()
	}

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
		if client.isClosed() {
			return
		}

		mylog.Println("Dialing URL:", client.urlStr)
		conn, _, err = dialer.Dial(client.urlStr, headers)
		if err == nil {
			mylog.Printf("Successfully connected to %s.\n", client.urlStr)
			break
		}

		retryCount++
		if retryCount > maxRetryAttempts {
			mylog.Printf("Exceeded maximum retry attempts for WebSocket[%v]: %v\n", client.urlStr, err)
			return
		}

		mylog.Printf("Failed to connect to WebSocket[%v]: %v, retrying in 5 seconds...\n", client.urlStr, err)

		select {
		case <-client.closeCh:
			return
		case <-time.After(5 * time.Second):
		}
	}

	if client.isClosed() {
		if conn != nil {
			_ = conn.Close()
		}
		return
	}

	// 复用现有的 client 完成重连。
	client.swapConn(conn)

	// 再次发送元事件
	message := map[string]interface{}{
		"meta_event_type": "lifecycle",
		"post_type":       "meta_event",
		"self_id":         client.botID,
		"sub_type":        "connect",
		"time":            int(time.Now().Unix()),
	}

	mylog.Printf("Message: %+v\n", message)

	if err := client.SendMessage(message); err != nil {
		mylog.Printf("Error sending reconnect lifecycle message: %v\n", err)

		badConn := client.swapConn(nil)
		if badConn != nil {
			_ = badConn.Close()
		}

		go func() {
			select {
			case <-client.closeCh:
				return
			case <-time.After(2 * time.Second):
				client.Reconnect()
			}
		}()

		return
	}

	if client.isClosed() {
		oldConn := client.swapConn(nil)
		if oldConn != nil {
			_ = oldConn.Close()
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	client.setCancel(cancel)
	heartbeatinterval := config.GetHeartBeatInterval()
	go client.sendHeartbeat(ctx, client.botID, heartbeatinterval)
	go client.handleIncomingMessages(cancel)

	mylog.Printf("Successfully reconnected to WebSocket.")

}

func (client *WebSocketClient) recordSendFailure(data []byte) {
	if config.GetDisableErrorChan() {
		return
	}

	copied := append([]byte(nil), data...)

	client.sendFailuresMu.Lock()
	defer client.sendFailuresMu.Unlock()

	if len(client.sendFailures) >= maxSendFailures {
		copy(client.sendFailures, client.sendFailures[1:])
		client.sendFailures[len(client.sendFailures)-1] = copied
		return
	}

	client.sendFailures = append(client.sendFailures, copied)
}

func (client *WebSocketClient) drainSendFailures() [][]byte {
	client.sendFailuresMu.Lock()
	defer client.sendFailuresMu.Unlock()

	if len(client.sendFailures) == 0 {
		return nil
	}

	failures := client.sendFailures
	client.sendFailures = nil
	return failures
}

func (client *WebSocketClient) processFailedMessages() {
	failures := client.drainSendFailures()
	if len(failures) == 0 {
		return
	}

	for _, data := range failures {
		if len(data) == 0 {
			continue
		}

		if err := client.sendRaw(websocket.TextMessage, data); err != nil {
			mylog.Printf("Error resending websocket message: %v\n", err)
		}
	}
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
	if heartbeatinterval <= 0 {
		heartbeatinterval = 5
	}

	ticker := time.NewTicker(time.Duration(heartbeatinterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
			if err := client.SendMessage(message); err != nil {
				mylog.Printf("Error sending heartbeat: %v", err)
			}
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
		sendFailures: [][]byte{},
		writeCh:      make(chan writeRequest, 5000),
		closeCh:      make(chan struct{}),
	}
	go client.startWriter() // 启动写 Goroutine

	// Sending initial message similar to your setupB function
	message := map[string]interface{}{
		"meta_event_type": "lifecycle",
		"post_type":       "meta_event",
		"self_id":         botID,
		"sub_type":        "connect",
		"time":            int(time.Now().Unix()),
	}

	mylog.Printf("Message: %+v\n", message)

	if err := client.SendMessage(message); err != nil {
		mylog.Printf("Error sending initial lifecycle message: %v\n", err)
		_ = client.Close()
		return nil, err
	}

	// Starting goroutine for heartbeats and another for listening to messages
	ctx, cancel := context.WithCancel(context.Background())

	client.setCancel(cancel)
	heartbeatinterval := config.GetHeartBeatInterval()
	go client.sendHeartbeat(ctx, botID, heartbeatinterval)
	go client.handleIncomingMessages(cancel)

	return client, nil
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
