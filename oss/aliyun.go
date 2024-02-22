package oss

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/green"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

var (
	aliyunonce   sync.Once
	aliyunclient *oss.Client
)

// 初始化oss单例
func initaliyunClient() {
	aliyunonce.Do(func() {
		aliyunclient, _ = oss.New(config.GetAliyunEndpoint(), config.GetAliyunAccessKeyId(), config.GetAliyunAccessKeySecret())
	})
}

// 初始化Green客户端
func initGreenClient() *green.Client {
	client, err := green.NewClientWithAccessKey(config.GetRegionID(), config.GetAliyunAccessKeyId(), config.GetAliyunAccessKeySecret())
	if err != nil {
		panic(err) // 或者处理错误
	}
	return client
}

// 上传并审核
func UploadAndAuditImageA(base64Data string) (string, error) {
	initaliyunClient()

	// Decode base64 data
	decodedData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	// Create a temporary file to save decoded data
	tmpFile, err := os.CreateTemp("", "upload-*.jpg")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err = tmpFile.Write(decodedData); err != nil {
		return "", err
	}

	// 计算解码数据的 MD5
	h := md5.New()
	if _, err := h.Write(decodedData); err != nil {
		return "", err
	}
	md5Hash := fmt.Sprintf("%x", h.Sum(nil))

	// 使用 MD5 值作为对象键
	objectKey := md5Hash + ".jpg"
	// 上传文件到 OSS
	bucket, err := aliyunclient.Bucket(config.GetAliyunBucketName())
	if err != nil {
		return "", err
	}

	err = bucket.PutObjectFromFile(objectKey, tmpFile.Name())
	if err != nil {
		return "", err
	}

	if config.GetAliyunAudit() {
		// 图片审核
		greenClient := initGreenClient()
		auditResult, err := auditImage(greenClient, objectKey)
		if err != nil {
			return "", err
		}
		if !auditResult {
			return "", fmt.Errorf("image failed to pass audit")
		}
	}

	// 图片正常，返回图片 URL
	imageURL := "https://" + config.GetAliyunBucketName() + ".oss-" + config.GetRegionID() + ".aliyuncs.com/" + objectKey
	return imageURL, nil
}

// 上传语音
func UploadAndAuditRecordA(base64Data string) (string, error) {
	initaliyunClient()

	// Decode base64 data
	decodedData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	// Create a temporary file to save decoded data
	tmpFile, err := os.CreateTemp("", "upload-*.amr")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err = tmpFile.Write(decodedData); err != nil {
		return "", err
	}

	// 计算解码数据的 MD5
	h := md5.New()
	if _, err := h.Write(decodedData); err != nil {
		return "", err
	}
	md5Hash := fmt.Sprintf("%x", h.Sum(nil))

	// 使用 MD5 值作为对象键
	objectKey := md5Hash + ".amr"
	// 上传文件到 OSS
	bucket, err := aliyunclient.Bucket(config.GetAliyunBucketName())
	if err != nil {
		return "", err
	}

	err = bucket.PutObjectFromFile(objectKey, tmpFile.Name())
	if err != nil {
		return "", err
	}

	// 图片正常，返回语音 URL
	imageURL := "https://" + config.GetAliyunBucketName() + ".oss-" + config.GetRegionID() + ".aliyuncs.com/" + objectKey
	return imageURL, nil
}

// auditImage 审核图片
func auditImage(client *green.Client, objectKey string) (bool, error) {
	// 构造审核请求
	request := green.CreateImageSyncScanRequest()
	request.Method = "POST"
	request.AcceptFormat = "JSON"

	bucketName := config.GetAppIDStr()                  // OSS桶名
	imageURL := "oss://" + bucketName + "/" + objectKey // 使用oss://协议指定图片

	task := map[string]interface{}{
		"dataId": uuid.New().String(),
		"url":    imageURL,
	}

	scenes := []string{"porn"} // 审核场景
	tasks := []map[string]interface{}{task}

	bizData := map[string]interface{}{
		"scenes": scenes,
		"tasks":  tasks,
	}

	bizContent, err := json.Marshal(bizData)
	if err != nil {
		return false, err
	}

	request.SetContent(bizContent)

	// 发送审核请求
	response, err := client.ImageSyncScan(request)
	if err != nil {
		return false, err
	}

	// 解析审核结果
	var result struct {
		Code int `json:"code"`
		Data []struct {
			Results []struct {
				Suggestion string `json:"suggestion"`
			} `json:"results"`
		} `json:"data"`
	}

	err = json.Unmarshal(response.GetHttpContentBytes(), &result)
	if err != nil {
		return false, err
	}
	mylog.Printf("阿里云审核结果(596代表未开通-请查看config.yml如何开通):%v", result)
	for _, data := range result.Data {
		for _, r := range data.Results {
			if r.Suggestion != "pass" {
				// 如果审核结果不是"pass"，则图片不合规
				return false, nil
			}
		}
	}

	// 所有审核均通过
	return true, nil
}
