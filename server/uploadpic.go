package server

import (
	"bytes"
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

const (
	MaximumImageSize        = 10 * 1024 * 1024
	AllowedUploadsPerMinute = 100
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
		if serverPort == "443" {
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
		if serverPort == "443" {
			protocol = "https"
		}

		imageURL := fmt.Sprintf("%s://%s:%s/channel_temp/%s", protocol, serverAddress, serverPort, fileName)
		c.JSON(http.StatusOK, gin.H{"url": imageURL})

	}
}

// 检查是否超过调用频率限制
func (rl *RateLimiter) CheckAndUpdateRateLimit(ipAddress string) bool {
	// 获取 MaxRequests 的当前值
	maxRequests := config.GetImageLimitB()

	now := time.Now()
	rl.Counts[ipAddress] = append(rl.Counts[ipAddress], now)

	// Remove expired entries
	for len(rl.Counts[ipAddress]) > 0 && now.Sub(rl.Counts[ipAddress][0]) > RequestInterval {
		rl.Counts[ipAddress] = rl.Counts[ipAddress][1:]
	}

	return len(rl.Counts[ipAddress]) <= maxRequests
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
