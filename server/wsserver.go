package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hoshinonyaruko/gensokyo/Processor"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
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
		}
	} else {
		// 如果请求头中没有token，则从URL参数中获取
		token = c.Query("access_token")
	}

	if token == "" {
		log.Printf("Connection failed due to missing token. Headers: %v", c.Request.Header)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// 使用GetWsServerToken()来获取有效的token
	validToken := config.GetWsServerToken()
	if token != validToken {
		log.Printf("Connection failed due to incorrect token. Headers: %v, Provided token: %s", c.Request.Header, tokenFromHeader)
		c.JSON(http.StatusForbidden, gin.H{"error": "Incorrect token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}

	clientIP := c.ClientIP()
	log.Printf("WebSocket client connected. IP: %s", clientIP)

	// 创建WebSocketServerClient实例
	client := &WebSocketServerClient{
		Conn:  conn,
		API:   api,
		APIv2: apiV2,
	}
	// 将此客户端添加到Processor的WsServerClients列表中
	p.WsServerClients = append(p.WsServerClients, client)

	// 获取botID
	botID := config.GetAppID()

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
		log.Printf("Error sending connection success message: %v\n", err)
	}

	defer conn.Close()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
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
		log.Printf("Error unmarshalling message: %v, Original message: %s", err, string(msg))
		return
	}

	fmt.Println("Received from WebSocket onebotv11 client:", wsclient.TruncateMessage(message, 500))
	// 调用callapi
	callapi.CallAPIFromDict(client, client.API, client.APIv2, message)
}

// 发信息给client
func (c *WebSocketServerClient) SendMessage(message map[string]interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}
	return c.Conn.WriteMessage(websocket.TextMessage, msgBytes)
}
