package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hoshinonyaruko/gensokyo/Processor"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/wsclient"
	"github.com/tencent-connect/botgo/openapi"
)

type WebSocketServerClient struct {
	Conn  *websocket.Conn
	API   openapi.OpenAPI
	APIv2 openapi.OpenAPI
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 确保WebSocketServerClient实现了interfaces.WebSocketServerClienter接口
var _ callapi.WebSocketServerClienter = &WebSocketServerClient{}

// 使用闭包结构 因为gin需要c *gin.Context固定签名
func WsHandlerWithDependencies(api openapi.OpenAPI, apiV2 openapi.OpenAPI, p *Processor.Processors) gin.HandlerFunc {
	return func(c *gin.Context) {
		wsHandler(api, apiV2, p, c)
	}
}

// 处理正向ws客户端的连接
func wsHandler(api openapi.OpenAPI, apiV2 openapi.OpenAPI, p *Processor.Processors, c *gin.Context) {
	// 先从请求头中尝试获取token
	tokenFromHeader := c.Request.Header.Get("Authorization")
	token := ""
	if tokenFromHeader != "" {
		if strings.HasPrefix(tokenFromHeader, "Token ") {
			// 从 "Token " 后面提取真正的token值
			token = strings.TrimPrefix(tokenFromHeader, "Token ")
		} else if strings.HasPrefix(tokenFromHeader, "Bearer ") {
			// 从 "Bearer " 后面提取真正的token值
			token = strings.TrimPrefix(tokenFromHeader, "Bearer ")
		} else {
			// 直接使用token值
			token = tokenFromHeader
		}
	} else {
		// 如果请求头中没有token，则从URL参数中获取
		token = c.Query("access_token")
	}

	// 获取配置中的有效 token
	validToken := config.GetWsServerToken()

	// 如果配置的 token 不为空，但提供的 token 为空或不匹配
	if validToken != "" && (token == "" || token != validToken) {
		if token == "" {
			mylog.Printf("Connection failed due to missing token. Headers: %v", c.Request.Header)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		} else {
			mylog.Printf("Connection failed due to incorrect token. Headers: %v, Provided token: %s", c.Request.Header, token)
			c.JSON(http.StatusForbidden, gin.H{"error": "Incorrect token"})
		}
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		mylog.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}

	clientIP := c.ClientIP()
	mylog.Printf("WebSocket client connected. IP: %s", clientIP)

	// 创建WebSocketServerClient实例
	client := &WebSocketServerClient{
		Conn:  conn,
		API:   api,
		APIv2: apiV2,
	}
	// 将此客户端添加到Processor的WsServerClients列表中
	p.WsServerClients = append(p.WsServerClients, client)

	// 获取botID

	var botID uint64
	if config.GetUseUin() {
		botID = uint64(config.GetUinint64())
	} else {
		botID = config.GetAppID()
	}

	// 发送连接成功的消息
	message := map[string]interface{}{
		"meta_event_type": "lifecycle",
		"post_type":       "meta_event",
		"self_id":         botID,
		"sub_type":        "connect",
		"time":            int(time.Now().Unix()),
	}
	err = client.SendMessage(message)
	if err != nil {
		mylog.Printf("Error sending connection success message: %v\n", err)
	}

	// 在defer语句之前运行
	defer func() {
		// 移除客户端从WsServerClients
		for i, wsClient := range p.WsServerClients {
			if wsClient == client {
				p.WsServerClients = append(p.WsServerClients[:i], p.WsServerClients[i+1:]...)
				break
			}
		}
	}()
	//退出时候的清理
	defer conn.Close()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			mylog.Printf("Error reading message: %v", err)
			return
		}

		if messageType == websocket.TextMessage {
			processWSMessage(client, p)
		}
	}
}

func processWSMessage(client *WebSocketServerClient, msg []byte) {
	var message callapi.ActionMessage
	err := json.Unmarshal(msg, &message)
	if err != nil {
		mylog.Printf("Error unmarshalling message: %v, Original message: %s", err, string(msg))
		return
	}

	mylog.Println("Received from WebSocket onebotv11 client:", wsclient.TruncateMessage(message, 500))
	// 调用callapi
	callapi.CallAPIFromDict(client, client.API, client.APIv2, message)
}

// 发信息给client
func (c *WebSocketServerClient) SendMessage(message map[string]interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		mylog.Println("Error marshalling message:", err)
		return err
	}
	return c.Conn.WriteMessage(websocket.TextMessage, msgBytes)
}

func (client *WebSocketServerClient) Close() error {
	return client.Conn.Close()
}
