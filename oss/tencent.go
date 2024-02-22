package oss

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var (
	once   sync.Once
	client *cos.Client
)

// 初始化oss单例
func initClient() {
	once.Do(func() {
		bucketURL, _ := url.Parse(config.GetTencentBucketURL())
		b := &cos.BaseURL{BucketURL: bucketURL}
		c := cos.NewClient(b, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  config.GetTencentCosSecretid(),
				SecretKey: config.GetTencentSecretKey(),
			},
		})
		client = c
	})
}

// 上传并审核
func UploadAndAuditImage(base64Data string) (string, error) {
	initClient()

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

	// 上传文件到 COS
	_, err = client.Object.PutFromFile(context.Background(), objectKey, tmpFile.Name(), nil)
	if err != nil {
		return "", err
	}

	if config.GetTencentAudit() {
		// Call ImageRecognition
		res, _, err := client.CI.ImageRecognition(context.Background(), objectKey, "")
		if err != nil {
			return "", err
		}

		// 使用 mylog 输出审核结果
		mylog.Printf("Image Recognition Results: %+v\n", res)

		// 检查图片审核结果
		if res.Result != 0 {
			// 如果图片不正常，记录日志并返回空字符串
			mylog.Printf("Image is not normal. Result code: %d\n", res.Result)
			return "", nil
		}
	}

	// 图片正常，返回图片 URL
	bucketURL, _ := url.Parse(config.GetTencentBucketURL()) // 确保这里的 GetTencentBucketURL 是正确的函数调用
	imageURL := bucketURL.String() + "/" + objectKey
	return imageURL, nil
}

// 上传语音
func UploadAndAuditRecord(base64Data string) (string, error) {
	initClient()

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

	// 上传文件到 COS
	_, err = client.Object.PutFromFile(context.Background(), objectKey, tmpFile.Name(), nil)
	if err != nil {
		return "", err
	}

	// 语音正常，返回语音 URL
	bucketURL, _ := url.Parse(config.GetTencentBucketURL()) // 确保这里的 GetTencentBucketURL 是正确的函数调用
	imageURL := bucketURL.String() + "/" + objectKey
	return imageURL, nil
}
