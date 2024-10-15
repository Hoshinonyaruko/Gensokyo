package server

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/websocket/client"
)

// Payload 定义请求载荷结构
type Payload struct {
	D  ValidationRequest `json:"d"`
	Op int               `json:"op"`
}

// ValidationRequest 定义鉴权请求结构
type ValidationRequest struct {
	PlainToken string `json:"plain_token"`
	EventTs    string `json:"event_ts"`
}

// WebhookPayload 定义Webhook消息结构
type WebhookPayload struct {
	PlainToken string `json:"plain_token"`
	EventTs    string `json:"event_ts"`
	RawMessage []byte // 保存原始消息内容
}

// WebhookHandler 负责处理 Webhook 的接收和消息处理
type WebhookHandler struct {
	messageQueue chan *WebhookPayload
	closeChan    chan error
}

// NewWebhookHandler 创建新的 WebhookHandler 实例
func NewWebhookHandler(queueSize int) *WebhookHandler {
	return &WebhookHandler{
		messageQueue: make(chan *WebhookPayload, queueSize),
		closeChan:    make(chan error),
	}
}

// CreateHandleValidation 创建用于签名验证和消息入队的处理函数
func CreateHandleValidation(botSecret string, wh *WebhookHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Println("Failed to read HTTP body:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}

		var payload Payload
		if err := json.Unmarshal(httpBody, &payload); err != nil {
			log.Println("Failed to parse HTTP payload:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse payload"})
			return
		}

		// 生成种子并创建私钥
		seed := botSecret
		for len(seed) < ed25519.SeedSize {
			seed = strings.Repeat(seed, 2)
		}
		seed = seed[:ed25519.SeedSize]
		reader := strings.NewReader(seed)
		_, privateKey, err := ed25519.GenerateKey(reader)
		if err != nil {
			log.Println("Failed to generate ed25519 key:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate ed25519 key"})
			return
		}

		// 拼接消息并生成签名
		var msg bytes.Buffer
		msg.WriteString(payload.D.EventTs)
		msg.WriteString(payload.D.PlainToken)
		signature := hex.EncodeToString(ed25519.Sign(privateKey, msg.Bytes()))

		// 推送验证成功消息到队列
		webhookPayload := &WebhookPayload{
			PlainToken: payload.D.PlainToken,
			EventTs:    payload.D.EventTs,
			RawMessage: httpBody,
		}
		wh.messageQueue <- webhookPayload

		// 返回签名验证响应
		c.JSON(http.StatusOK, gin.H{
			"plain_token": payload.D.PlainToken,
			"signature":   signature,
		})
	}
}

// listenAndProcessMessages 启动协程处理队列中的消息
func (wh *WebhookHandler) ListenAndProcessMessages() {
	for payload := range wh.messageQueue {
		go func(p *WebhookPayload) {
			log.Printf("Processing Webhook event with token: %s", p.PlainToken)
			// 业务逻辑处理的地方
			payload := &dto.WSPayload{}
			if err := json.Unmarshal(p.RawMessage, payload); err != nil {
				log.Printf("%s json failed, %v", p.EventTs, err)
				return
			}
			// 更新 global_s 的值
			atomic.StoreInt64(&client.Global_s, payload.S)

			payload.RawMessage = p.RawMessage
			log.Printf("%s receive %s message, %s", p.EventTs, dto.OPMeans(payload.OPCode), string(p.RawMessage))

			// 性能不够 报错也没用 就扬了
			go event.ParseAndHandle(payload)
		}(payload)
	}
	log.Println("Message queue is closed")
}

// Close 关闭消息队列
func (wh *WebhookHandler) Close() {
	close(wh.messageQueue)
}
