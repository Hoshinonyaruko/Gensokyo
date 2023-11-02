package server

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

const (
	MaximumImageSize        = 10 * 1024 * 1024
	AllowedUploadsPerMinute = 100
	MaxRequests             = 30
	RequestInterval         = time.Minute
)

type RateLimiter struct {
	Counts map[string][]time.Time
}

// 频率限制器
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		Counts: make(map[string][]time.Time),
	}
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

		fileName := generateRandomMd5() + "." + fileExt
		directoryPath := "./channel_temp/"
		savePath := directoryPath + fileName

		// Create the directory if it doesn't exist
		err = os.MkdirAll(directoryPath, 0755)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating directory"})
			return
		}

		err = os.WriteFile(savePath, imageBytes, 0644)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving file"})
			return
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
		if serverPort == "443" {
			protocol = "https"
		}

		imageURL := fmt.Sprintf("%s://%s:%s/channel_temp/%s", protocol, serverAddress, serverPort, fileName)
		c.JSON(http.StatusOK, gin.H{"url": imageURL})

	}
}

// 检查是否超过调用频率限制
// 默认1分钟30次 todo 允许用户自行在config编辑限制次数
func (rl *RateLimiter) CheckAndUpdateRateLimit(ipAddress string) bool {
	now := time.Now()
	rl.Counts[ipAddress] = append(rl.Counts[ipAddress], now)

	// Remove expired entries
	for len(rl.Counts[ipAddress]) > 0 && now.Sub(rl.Counts[ipAddress][0]) > RequestInterval {
		rl.Counts[ipAddress] = rl.Counts[ipAddress][1:]
	}

	return len(rl.Counts[ipAddress]) <= MaxRequests
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
func generateRandomMd5() string {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return ""
	}

	md5Hash := md5.Sum(randomBytes)
	return hex.EncodeToString(md5Hash[:])
}
