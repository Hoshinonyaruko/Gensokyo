package server

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo/config"
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
		fmt.Println("Received base64 data length:", len(base64Image), "characters")

		imageBytes, err := base64.StdEncoding.DecodeString(base64Image)
		if err != nil {
			fmt.Println("Error while decoding base64:", err) // Print error while decoding
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

		imageURL := fmt.Sprintf("http://%s:%s/channel_temp/%s", serverAddress, serverPort, fileName)
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
	fmt.Println("Received data size:", len(data), "bytes")

	_, format, err = image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// Print additional error information
		fmt.Println("Error while trying to decode image config:", err)
		return "", fmt.Errorf("error decoding image config: %w", err)
	}

	// Print the detected format
	fmt.Println("Detected image format:", format)

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

// 将base64图片通过lotus转换成url
func UploadBase64ImageToServer(base64Image string) (string, error) {
	if config.GetLotusValue() {
		serverDir := config.GetServer_dir()
		serverPort := config.GetPortValue()
		url := fmt.Sprintf("http://%s:%s/uploadpic", serverDir, serverPort)

		resp, err := postImageToServer(base64Image, url)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	serverDir := config.GetServer_dir()
	if isPublicAddress(serverDir) {
		url := fmt.Sprintf("http://127.0.0.1:%s/uploadpic", config.GetPortValue())

		resp, err := postImageToServer(base64Image, url)
		if err != nil {
			return "", err
		}
		return resp, nil
	}
	return "", errors.New("local server uses a private address; image upload failed")
}

// 请求图床api(图床就是lolus为false的gensokyo)
func postImageToServer(base64Image, targetURL string) (string, error) {
	data := url.Values{}
	data.Set("base64Image", base64Image) // 修改字段名以与服务器匹配

	resp, err := http.PostForm(targetURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error response from server: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if value, ok := responseMap["url"]; ok {
		return fmt.Sprintf("%v", value), nil
	}

	return "", fmt.Errorf("URL not found in response")
}

// 判断是否公网ip 填写域名也会被认为是公网,但需要用户自己确保域名正确解析到gensokyo所在的ip地址
func isPublicAddress(addr string) bool {
	if strings.Contains(addr, "localhost") || strings.HasPrefix(addr, "127.") || strings.HasPrefix(addr, "192.168.") {
		return false
	}
	if net.ParseIP(addr) != nil {
		return true
	}
	// If it's not a recognized IP address format, consider it a domain name (public).
	return true
}
