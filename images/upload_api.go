package images

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/hoshinonyaruko/gensokyo/config"
)

// 将base64图片通过lotus转换成url
func UploadBase64ImageToServer(base64Image string) (string, error) {
	// 根据serverPort确定协议
	protocol := "http"
	serverPort := config.GetPortValue()
	if serverPort == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() {
		serverDir := config.GetServer_dir()
		url := fmt.Sprintf("%s://%s:%s/uploadpic", protocol, serverDir, serverPort)

		resp, err := postImageToServer(base64Image, url)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	serverDir := config.GetServer_dir()
	// 当端口是443时，使用HTTP和444端口
	if serverPort == "443" {
		protocol = "http"
		serverPort = "444"
	}

	if isPublicAddress(serverDir) {
		url := fmt.Sprintf("%s://127.0.0.1:%s/uploadpic", protocol, serverPort)

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

	body, err := io.ReadAll(resp.Body)
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
