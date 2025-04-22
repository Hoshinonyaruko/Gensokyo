package server

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

const (
	MaximumImageSize        = 10 * 1024 * 1024
	AllowedUploadsPerMinute = 100
	RequestInterval         = time.Minute
)

// RateLimiter 使用 sync.Map 来支持并发操作
type RateLimiter struct {
	Counts sync.Map
}

// NewRateLimiter 创建一个新的 RateLimiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

// 闭包,网页后端,图床逻辑,基于gin和www静态文件的简易图床
func UploadBase64ImageHandler(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAddress := c.ClientIP()
		if !rateLimiter.CheckAndUpdateRateLimit(ipAddress) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		base64Image := c.PostForm("base64Image")
		// Print the length of the received base64 data
		mylog.Println("Received base64 data length:", len(base64Image), "characters")

		imageBytes, err := base64.StdEncoding.DecodeString(base64Image)
		if err != nil {
			mylog.Println("Error while decoding base64:", err) // Print error while decoding
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 data"})
			return
		}

		imageFormat, err := getImageFormat(imageBytes)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "undefined picture format1"})
			return
		}

		fileExt := getFileExtensionFromImageFormat(imageFormat)
		if fileExt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported image format2"})
			return
		}

		fileName := getFileMd5(imageBytes) + "." + fileExt
		directoryPath := "./channel_temp/"
		savePath := directoryPath + fileName

		// Create the directory if it doesn't exist
		err = os.MkdirAll(directoryPath, 0755)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating directory"})
			return
		}

		//如果文件存在则跳过
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			err = os.WriteFile(savePath, imageBytes, 0644)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving file"})
				return
			}
		} else {
			mylog.Println("File already exists, skipping save.")
		}

		var serverPort string
		serverAddress := config.GetServer_dir()
		frpport := config.GetFrpPort()
		if frpport != "0" {
			serverPort = frpport
		} else {
			serverPort = config.GetPortValue()
		}
		if serverAddress == "" {
			// Handle the case where the server address is not configured
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server address is not configured"})
			return
		}
		// 根据serverPort确定协议
		protocol := "http"
		if serverPort == "443" || config.GetForceSsl() {
			protocol = "https"
		}
		stun, err := idmap.ReadConfigv2("stun", "addr")
		var imageURL string
		if err == nil && stun != "" {
			imageURL = fmt.Sprintf("http://%s/channel_temp/%s", stun, fileName)
		} else {
			imageURL = fmt.Sprintf("%s://%s:%s/channel_temp/%s", protocol, serverAddress, serverPort, fileName)
		}
		c.JSON(http.StatusOK, gin.H{"url": imageURL})

	}
}

func UploadBase64ImageHandlerV2(rateLimiter *RateLimiter, apiv2 openapi.OpenAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAddress := c.ClientIP()
		if !rateLimiter.CheckAndUpdateRateLimit(ipAddress) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		// 从请求中获取必要的参数
		base64Image := c.PostForm("base64Image")

		var imageURL string
		var width, height int
		var err error

		// 根据参数调用不同的处理逻辑
		if base64Image != "" {
			imageURL, width, height, err = images.UploadBase64ImageToServer(base64Image, apiv2)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "either base64Image or url is required"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 如果上传成功，则返回图片的URL，群组ID，宽度和高度
		c.JSON(http.StatusOK, gin.H{
			"url":    imageURL,
			"width":  width,
			"height": height,
		})
	}
}

func UploadBase64ImageHandlerV3(rateLimiter *RateLimiter, apiv1 openapi.OpenAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAddress := c.ClientIP()
		if !rateLimiter.CheckAndUpdateRateLimit(ipAddress) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		base64Image := c.PostForm("base64Image")
		channelID := c.PostForm("channelID")

		if channelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "channelID is required"})
			return
		}

		fileImageData, err := base64.StdEncoding.DecodeString(base64Image)
		if err != nil {
			mylog.Printf("Base64 解码失败: %v", err)
			return
		}
		// 压缩 只有设置了阈值才会压缩
		compressedData, err := images.CompressSingleImage(fileImageData)
		if err != nil {
			mylog.Printf("Error compressing image: %v", err)
			return
		}

		newMessage := &dto.MessageToCreate{
			Content:   "",
			MsgID:     "1000",
			MsgType:   0,
			Timestamp: time.Now().Unix(),
		}

		if _, err = apiv1.PostMessageMultipart(context.TODO(), channelID, newMessage, compressedData); err != nil {
			mylog.Printf("使用multipart发送图文信息失败: %v message_id %v", err, 1000)
			return
		}

		// 计算压缩数据的MD5值
		md5Hash := md5.Sum(compressedData)
		md5String := strings.ToUpper(hex.EncodeToString(md5Hash[:]))
		imageURL := fmt.Sprintf("https://gchat.qpic.cn/qmeetpic/0/0-0-%s/0", md5String)

		// 获取图片宽高
		height, width, err := images.GetImageDimensions(imageURL)
		if err != nil {
			mylog.Printf("获取图片宽高出错: %v", err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"url":       imageURL,
			"channelID": channelID,
			"width":     width,
			"height":    height,
		})
	}
}

// 闭包,网页后端,语音床逻辑,基于gin和www静态文件的简易语音床
func UploadBase64RecordHandler(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAddress := c.ClientIP()
		if !rateLimiter.CheckAndUpdateRateLimit(ipAddress) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		base64REcord := c.PostForm("base64Record")
		// Print the length of the received base64 data
		mylog.Println("Received base64 data length:", len(base64REcord), "characters")

		RecordBytes, err := base64.StdEncoding.DecodeString(base64REcord)
		if err != nil {
			mylog.Println("Error while decoding base64:", err) // Print error while decoding
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 data"})
			return
		}

		fileName := getFileMd5(RecordBytes) + ".silk"
		directoryPath := "./channel_temp/"
		savePath := directoryPath + fileName

		// Create the directory if it doesn't exist
		err = os.MkdirAll(directoryPath, 0755)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating directory"})
			return
		}

		//如果文件存在则跳过
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			err = os.WriteFile(savePath, RecordBytes, 0644)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving file"})
				return
			}
		} else {
			mylog.Println("File already exists, skipping save.")
		}

		serverAddress := config.GetServer_dir()
		serverPort := config.GetPortValue()
		if serverAddress == "" {
			// Handle the case where the server address is not configured
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server address is not configured"})
			return
		}

		// 根据serverPort确定协议
		protocol := "http"
		if serverPort == "443" || config.GetForceSsl() {
			protocol = "https"
		}

		imageURL := fmt.Sprintf("%s://%s:%s/channel_temp/%s", protocol, serverAddress, serverPort, fileName)
		c.JSON(http.StatusOK, gin.H{"url": imageURL})

	}
}

// CheckAndUpdateRateLimit 检查某个 IP 是否超过了请求频率限制
func (rl *RateLimiter) CheckAndUpdateRateLimit(ipAddress string) bool {
	maxRequests := config.GetImageLimitB() // 获取最大请求数

	now := time.Now()
	// 从 sync.Map 获取当前 IP 的请求时间戳列表
	val, _ := rl.Counts.LoadOrStore(ipAddress, []time.Time{})

	// 将现有的请求时间戳列表转化为 []time.Time
	counts := val.([]time.Time)
	// 将当前请求时间戳添加到列表
	counts = append(counts, now)

	// 清理过期的请求时间戳
	expiredCount := 0
	for len(counts) > 0 && now.Sub(counts[0]) > RequestInterval {
		counts = counts[1:]
		expiredCount++
	}

	// 更新 sync.Map 中的 IP 请求时间戳列表
	rl.Counts.Store(ipAddress, counts)

	// 返回是否超过限制
	return len(counts) <= maxRequests
}

// 获取图片类型
func getImageFormat(data []byte) (format string, err error) {
	// Print the size of the data to check if it's being read correctly
	mylog.Println("Received data size:", len(data), "bytes")

	_, format, err = image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// Print additional error information
		mylog.Println("Error while trying to decode image config:", err)
		return "", fmt.Errorf("error decoding image config: %w", err)
	}

	// Print the detected format
	mylog.Println("Detected image format:", format)

	if format == "" {
		return "", errors.New("undefined picture format")
	}
	return format, nil
}

// 判断并返回图片类型
func getFileExtensionFromImageFormat(format string) string {
	switch format {
	case "jpeg":
		return "jpg"
	case "gif":
		return "gif"
	case "png":
		return "png"
	default:
		return ""
	}
}

// 生成随机md5图片名,防止碰撞
func getFileMd5(base64file []byte) string {
	md5Hash := md5.Sum(base64file)
	return hex.EncodeToString(md5Hash[:])
}

func HandleIpupdate(c *gin.Context) {
	reqParam := c.Query("addr")
	idmap.WriteConfigv2("stun", "addr", reqParam)
	c.JSON(http.StatusOK, gin.H{"addr": reqParam})
}
