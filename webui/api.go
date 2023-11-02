package webui

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

//go:embed dist/*
//go:embed dist/css/*
//go:embed dist/fonts/*
//go:embed dist/icons/*
//go:embed dist/js/*
var content embed.FS

// 中间件
func CombinedMiddleware(c *gin.Context) {
	// 检查是否为API请求
	log.Print(c.Param("filepath"))
	if strings.HasPrefix(c.Request.URL.Path, "/webui/api") {
		// 处理API请求
		appIDStr := config.GetAppIDStr()
		// 检查路径是否匹配 `/api/{uin}/process/logs`
		if strings.HasPrefix(c.Param("filepath"), "/api/") && strings.HasSuffix(c.Param("filepath"), "/process/logs") {
			if c.GetHeader("Upgrade") == "websocket" {
				mylog.WsHandlerWithDependencies(c)
			} else {
				getProcessLogs(c)
			}
			return
		}
		// 如果请求路径与appIDStr匹配，并且请求方法为PUT
		if c.Param("filepath") == appIDStr && c.Request.Method == http.MethodPut {
			HandleAppIDRequest(c)
			return
		}
		if c.Param("filepath") == "/api/"+appIDStr+"/process/status" {
			HandleProcessStatusRequest(c)
			return
		}
		if c.Param("filepath") == "/api/accounts" {
			HandleAccountsRequest(c)
			return
		}
		// 如果还有其他API端点，可以在这里继续添加...
	} else {
		// 否则，处理静态文件请求
		// 如果请求是 "/webui/" ，默认为 "index.html"
		filepathRequested := c.Param("filepath")
		if filepathRequested == "" || filepathRequested == "/" {
			filepathRequested = "index.html"
		}

		// 使用 embed.FS 读取文件内容
		filepathRequested = strings.TrimPrefix(filepathRequested, "/")
		data, err := content.ReadFile("dist/" + filepathRequested)
		if err != nil {
			fmt.Println("Error reading file:", err)
			c.Status(http.StatusNotFound)
			return
		}

		mimeType := getContentType(filepathRequested)

		c.Data(http.StatusOK, mimeType, data)
	}
}

func getContentType(path string) string {
	// todo 根据需要增加更多的 MIME 类型
	switch filepath.Ext(path) {
	case ".html":
		return "text/html"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "text/plain"
	}
}

type ResponseData struct {
	UIN      int64  `json:"uin"`
	Password string `json:"password"`
	Protocol int    `json:"protocol"`
}

type RequestData struct {
	Password string `json:"password"`
}

func HandleAccountsRequest(c *gin.Context) {
	responseData := []gin.H{
		{
			"uin":             config.GetAppID(),
			"predefined":      false,
			"process_created": true,
		},
	}

	c.JSON(http.StatusOK, responseData)
}

func HandleProcessStatusRequest(c *gin.Context) {
	responseData := gin.H{
		"status":     "running",
		"total_logs": 0,
		"restarts":   0,
		"qr_uri":     nil,
		"details": gin.H{
			"pid":         0,
			"status":      "running",
			"memory_used": 19361792,          // 示例内存使用量
			"cpu_percent": 0.0,               // 示例CPU使用率
			"start_time":  time.Now().Unix(), // 10位时间戳
		},
	}
	c.JSON(http.StatusOK, responseData)
}

func getProcessLogs(c *gin.Context) {
	c.JSON(200, []interface{}{})
}

func HandleAppIDRequest(c *gin.Context) {
	appIDStr := config.GetAppIDStr()

	// 将 appIDStr 转换为 int64
	uin, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 解析请求体中的JSON数据
	var requestData RequestData
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 创建响应数据
	responseData := ResponseData{
		UIN:      uin,
		Password: requestData.Password,
		Protocol: 5,
	}

	// 发送响应
	c.JSON(http.StatusOK, responseData)
}
