package url

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo/config"
)

const (
	bucketName = "shortURLs"
)

var (
	db *bolt.DB
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const length = 6

func generateRandomString() string {
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func generateHashedString(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:3]) // 取前3个字节，得到6个字符的16进制表示
}

func init() {
	var err error
	db, err = bolt.Open("gensokyo.db", 0600, nil)
	if err != nil {
		panic(err)
	}

	// Ensure bucket exists
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("failed to create or get the bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("Error initializing the database: %v", err))
	}
}

// 验证链接是否合法
func isValidURL(toTest string) bool {
	parsedURL, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	// 阻止localhost和本地IP地址
	host := parsedURL.Hostname()
	localHostnames := []string{"localhost", "127.0.0.1", "::1"}
	for _, localHost := range localHostnames {
		if host == localHost {
			return false
		}
	}

	// 检查是否是私有IP地址
	return !isPrivateIP(host)
}

// 检查是否是私有IP地址
func isPrivateIP(ipStr string) bool {
	privateIPBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	ip := net.ParseIP(ipStr)
	for _, block := range privateIPBlocks {
		_, ipnet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

// 检查和解码可能的Base64编码的URL
func decodeBase64IfNeeded(input string) string {
	if len(input)%4 == 0 { // 一个简单的检查来看它是否可能是Base64
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err == nil {
			return string(decoded)
		}
	}
	return input
}

// 生成短链接
func GenerateShortURL(longURL string) string {
	if config.GetLotusValue() {
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()
		url := fmt.Sprintf("http://%s:%s/url", serverDir, portValue)

		payload := map[string]string{"longURL": longURL}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshaling payload: %v", err)
			return ""
		}

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Printf("Error while generating short URL: %v", err)
			return ""
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Received non-200 status code: %d from server: %v", resp.StatusCode, url)
			return ""
		}

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Println("Error decoding response")
			return ""
		}

		shortURL, ok := response["shortURL"].(string)
		if !ok {
			log.Println("shortURL not found or not a string in the response")
			return ""
		}

		return shortURL

	} else {
		shortURL := generateHashedString(longURL)

		exists, err := existsInDB(shortURL)
		if err != nil {
			log.Printf("Error checking if shortURL exists in DB: %v", err)
			return "" // 如果有错误, 返回空的短链接
		}
		if exists {
			for {
				shortURL = generateRandomString()
				exists, err := existsInDB(shortURL)
				if err != nil {
					log.Printf("Error checking if shortURL exists in DB: %v", err)
					return "" // 如果有错误, 返回空的短链接
				}
				if !exists {
					break
				}
			}
		}

		// 存储短URL和对应的长URL
		err = storeURL(shortURL, longURL)
		if err != nil {
			log.Printf("Error storing URL in DB: %v", err)
			return ""
		}

		return shortURL
	}
}

func existsInDB(shortURL string) (bool, error) {
	exists := false
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		v := b.Get([]byte(shortURL))
		if v != nil {
			exists = true
		}
		return nil
	})
	if err != nil {
		log.Printf("Error accessing the database: %v", err) // 记录错误
		return false, err
	}
	return exists, nil
}

// 从数据库获取短链接
func getLongURLFromDB(shortURL string) (string, error) {
	if config.GetLotusValue() {
		serverDir := config.GetServer_dir()
		portValue := config.GetPortValue()
		url := fmt.Sprintf("http://%s:%s/url/%s", serverDir, portValue, shortURL)

		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Received non-200 status code: %d while fetching long URL from server: %v", resp.StatusCode, url)
			return "", fmt.Errorf("error fetching long URL from remote server with status code: %d", resp.StatusCode)
		}

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return "", fmt.Errorf("error decoding response from server")
		}
		return response["longURL"].(string), nil
	} else {
		var longURL string
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucketName))
			v := b.Get([]byte(shortURL))
			if v == nil {
				return fmt.Errorf("URL not found")
			}
			longURL = string(v)
			return nil
		})
		return longURL, err
	}
}

// storeURL 存储长URL和对应的短URL
func storeURL(shortURL, longURL string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.Put([]byte(shortURL), []byte(longURL))
	})
}

// 安全性检查
func isMalicious(decoded string) bool {
	lowerDecoded := strings.ToLower(decoded)

	// 检查javascript协议，用于防止XSS
	if strings.HasPrefix(lowerDecoded, "javascript:") {
		return true
	}

	// 检查data协议，可能被用于各种攻击
	if strings.HasPrefix(lowerDecoded, "data:") {
		return true
	}

	// 检查常见的HTML标签，这可能用于指示XSS攻击
	for _, tag := range []string{"<script", "<img", "<iframe", "<link", "<style"} {
		if strings.Contains(lowerDecoded, tag) {
			return true
		}
	}

	return false
}

// 短链接服务handler
func CreateShortURLHandler(c *gin.Context) {
	rawURL := c.PostForm("url")
	longURL := decodeBase64IfNeeded(rawURL)

	if longURL == "" || isMalicious(longURL) || !isValidURL(longURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	// Generate short URL
	shortURL := GenerateShortURL(longURL)

	// Construct baseUrl
	serverDir := config.GetServer_dir()
	portValue := config.GetPortValue()
	protocol := "http"
	if portValue == "443" {
		protocol = "https"
	}
	baseUrl := protocol + "://" + serverDir
	if portValue != "80" && portValue != "443" && portValue != "" {
		baseUrl += ":" + portValue
	}

	c.JSON(http.StatusOK, gin.H{"shortURL": baseUrl + "/url/" + shortURL})
}

// 短链接baseurl
func GetBaseURL() string {
	serverDir := config.GetServer_dir()
	portValue := config.GetPortValue()
	protocol := "http"
	if portValue == "443" {
		protocol = "https"
	}
	baseUrl := protocol + "://" + serverDir
	if portValue != "80" && portValue != "443" && portValue != "" {
		baseUrl += ":" + portValue
	}
	return baseUrl
}

// RedirectFromShortURLHandler
func RedirectFromShortURLHandler(c *gin.Context) {
	shortURL := c.Param("shortURL")

	// Fetch from Bolt DB
	longURL, err := getLongURLFromDB(shortURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// Ensure longURL has a scheme (http or https)
	if !strings.HasPrefix(longURL, "http://") && !strings.HasPrefix(longURL, "https://") {
		// Add default scheme if missing
		longURL = "http://" + longURL
	}

	c.Redirect(http.StatusMovedPermanently, longURL)
}

func CloseDB() {
	db.Close()
}
