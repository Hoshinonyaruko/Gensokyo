package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hoshinonyaruko/gensokyo/callapi"
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

// 使用闭包结构 因为gin需要c *gin.Context固定签名
func WsHandlerWithDependencies(api openapi.OpenAPI, apiV2 openapi.OpenAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		wsHandler(api, apiV2, c)
	}
}

func wsHandler(api openapi.OpenAPI, apiV2 openapi.OpenAPI, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}

	clientIP := c.ClientIP()
	headers := c.Request.Header
	log.Printf("WebSocket client connected. IP: %s, Headers: %v", clientIP, headers)

	// 创建WebSocketServerClient实例
	client := &WebSocketServerClient{
		Conn:  conn,
		API:   api,
		APIv2: apiV2,
	}

	defer conn.Close()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		if messageType == websocket.TextMessage {
			processWSMessage(client, p) // 使用WebSocketServerClient而不是直接使用连接
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
