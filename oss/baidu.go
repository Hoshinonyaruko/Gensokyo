package oss

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Baidu-AIP/golang-sdk/aip/censor"
	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

var (
	onceBaidu   sync.Once
	clientBaidu *censor.ContentCensorClient
	clientBos   *bos.Client
)

// 初始化百度云内容审核客户端单例
func initClientB() {
	onceBaidu.Do(func() {
		// 获取BOS的AK/SK
		bosAK := config.GetBaiduBCEAK()
		bosSK := config.GetBaiduBCESK()
		if bosAK == "" || bosSK == "" {
			mylog.Println("Error: BOS AK/SK are empty")
			return
		}

		// 获取BOS Endpoint，此处用Bucket Name代替
		bosEndpoint := config.GetBaiduBOSBucketName()
		if bosEndpoint == "" {
			mylog.Println("warring: BOS Endpoint (Bucket Name) is empty")
		}

		var err error
		clientBos, err = bos.NewClient(bosAK, bosSK, bosEndpoint)
		if err != nil {
			mylog.Printf("Error initializing BOS client: %v\n", err)
			return
		}

		// 获取Censor的AK/SK
		censorAK := config.GetBaiduBCEAK()
		censorSK := config.GetBaiduBCESK()
		if censorAK == "" || censorSK == "" {
			mylog.Println("Error: Censor AK/SK are empty")
			return
		}
		clientBaidu = censor.NewCloudClient(censorAK, censorSK)
	})
}

type AuditResponse struct {
	LogID      int64  `json:"log_id"`
	Conclusion string `json:"conclusion"`
	ErrorCode  int    `json:"error_code"`
	ErrorMsg   string `json:"error_msg"`
	Data       []struct {
		Msg         string  `json:"msg"`
		Probability float64 `json:"probability"`
		Type        int     `json:"type"`
	} `json:"data"`
}

// UploadAndAuditImageB 上传并根据配置审核图片
func UploadAndAuditImageB(base64Data string) (string, error) {
	initClientB()

	// 解码base64数据
	decodedData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	// 根据配置获取审核模式
	auditMode := config.GetBaiduAudit()

	// 根据审核模式进行操作
	switch auditMode {
	case 0:
		// 审核模式0: 仅上传图片
		return uploadImageBOS(decodedData)
	case 1:
		// 审核模式1: 使用BOS上传并审核图片
		imageURL, err := uploadImageBOS(decodedData)
		if err != nil {
			return "", err
		}
		return auditUsingBOS(imageURL)
	case 2:
		// 审核模式2: 使用原始上传逻辑并审核图片
		imageURL, err := originalUploadBehavior(base64Data)
		if err != nil {
			return "", err
		}
		if !AuditImageContentAIP(imageURL) {
			return "", fmt.Errorf("image failed the audit")
		}
		return imageURL, nil
	default:
		return "", fmt.Errorf("invalid audit mode")
	}
}

// UploadAndAuditRecordB 上传语音
func UploadAndAuditRecordB(base64Data string) (string, error) {
	initClientB()

	// 解码base64数据
	decodedData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	return uploadRecordBOS(decodedData)

}

func originalUploadBehavior(base64Image string) (string, error) {
	// 原有的UploadBase64ImageToServer函数的实现
	protocol := "http"
	serverPort := config.GetPortValue()
	if serverPort == "443" ||config.GetForceSsl(){
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
	if serverPort == "443" ||config.GetForceSsl(){
		protocol = "http"
		serverPort = config.GetHttpPortAfterSsl()
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

// uploadImageBOS 使用BOS进行图片上传
func uploadImageBOS(data []byte) (string, error) {
	// 计算MD5以用作文件名
	md5Hash := md5.New()
	md5Hash.Write(data)
	md5String := hex.EncodeToString(md5Hash.Sum(nil))

	// 创建临时文件
	picname := fmt.Sprintf("qqbot-upload-%s-*.jpg", md5String)
	tmpFile, err := os.CreateTemp("", picname)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name()) // 清理临时文件
	}()

	// 写入数据到临时文件
	if _, err = tmpFile.Write(data); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %v", err)
	}

	// 确保数据写入磁盘
	if err = tmpFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync temp file: %v", err)
	}

	// 获取临时文件的实际文件名
	actualFileName := filepath.Base(tmpFile.Name())

	// 上传到BOS
	bucketDomain := config.GetBaiduBOSBucketName() //bucket的域名
	appid := config.GetAppIDStr()
	_, err = clientBos.PutObjectFromFile(appid, actualFileName, tmpFile.Name(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload to BOS: %v", err)
	}

	// 返回图片URL
	return fmt.Sprintf("https://%s/%s/%s", bucketDomain, appid, actualFileName), nil
}

// uploadRecordBOS 使用BOS进行语音上传
func uploadRecordBOS(data []byte) (string, error) {
	// 计算MD5以用作文件名
	md5Hash := md5.New()
	md5Hash.Write(data)
	md5String := hex.EncodeToString(md5Hash.Sum(nil))

	// 创建临时文件
	picname := fmt.Sprintf("qqbot-upload-%s-*.amr", md5String)
	tmpFile, err := os.CreateTemp("", picname)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name()) // 清理临时文件
	}()

	// 写入数据到临时文件
	if _, err = tmpFile.Write(data); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %v", err)
	}

	// 确保数据写入磁盘
	if err = tmpFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync temp file: %v", err)
	}

	// 获取临时文件的实际文件名
	actualFileName := filepath.Base(tmpFile.Name())

	// 上传到BOS
	bucketDomain := config.GetBaiduBOSBucketName() //bucket的域名
	appid := config.GetAppIDStr()
	_, err = clientBos.PutObjectFromFile(appid, actualFileName, tmpFile.Name(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload to BOS: %v", err)
	}

	// 返回图片URL
	return fmt.Sprintf("https://%s/%s/%s", bucketDomain, appid, actualFileName), nil
}

// auditUsingBOS 使用BOS进行图片审核
func auditUsingBOS(imageURL string) (string, error) {
	ok := AuditImageContentAIP(imageURL)
	if !ok {
		return "", fmt.Errorf("image failed the audit 图片未通过百度云审核,可到百度云控制台自定义审核维度")
	}
	return imageURL, nil
}

// AuditImageContent 检查图片内容是否合规 AIP方式
func AuditImageContentAIP(imageURL string) bool {
	options := map[string]interface{}{} // 根据需要设置选项
	response := clientBaidu.ImgCensorUrl(imageURL, options)
	var auditResp AuditResponse
	err := json.Unmarshal([]byte(response), &auditResp)
	if err != nil {
		mylog.Printf("failed to parse audit response: %v", err)
		return false
	}
	//显示图片的审核结论
	mylog.Printf("AIP Audit result for image '%s': %v", imageURL, auditResp)

	return auditResp.Conclusion == "合规"
}
