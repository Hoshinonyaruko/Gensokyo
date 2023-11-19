package images

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// 上传图片到 OSS 所需配置信息
type UploadWithOSS struct {

	// OSS 域名
	Endpoint         string
	internalEndpoint string

	// OSS Bucket 名
	OSSBucket string

	// OSS 连接实例
	client *oss.Client
	bucket *oss.Bucket
}

// NewUploadWithOSS: 创建 UploadWithOSS 的新实例
func NewUploadWithOSS(isinternal bool, endpoint, accessKeyID, accessKeySecret, bucketName string) (*UploadWithOSS, error) {

	// AccessKey 和 SecretKey 写入自身环境变量
	os.Setenv("OSS_ACCESS_KEY_ID", accessKeyID)
	os.Setenv("OSS_ACCESS_KEY_SECRET", accessKeySecret)

	// 构建 OSS 内网地址
	internalEndpoint := strings.Replace(endpoint, ".aliyuncs.com", "-internal.aliyuncs.com", 1)

	// 初始化 OSS 获取登录信息
	provider, err := oss.NewEnvironmentVariableCredentialsProvider()
	if err != nil {
		return nil, err
	}

	// 根据情况构建 server_dir
	var server_dir string
	if isinternal {
		server_dir = internalEndpoint
	} else {
		server_dir = endpoint
	}

	// 创建新 OSS 连接
	client, err := oss.New(server_dir, accessKeyID, accessKeySecret, oss.SetCredentialsProvider(&provider))
	if err != nil {
		return nil, err
	}

	// 设定 Bucket 名称
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}

	return &UploadWithOSS{
		Endpoint:         endpoint,
		internalEndpoint: internalEndpoint,
		OSSBucket:        bucketName,
		client:           client,
		bucket:           bucket,
	}, nil
}

// UploadImage: 上传图片到指定的目录
func (ou *UploadWithOSS) UploadImage(image []byte, ext string) (string, error) {

	// 计算图像数据的 MD5 值
	md5Hash := md5.New()
	md5Hash.Write(image)
	md5 := hex.EncodeToString(md5Hash.Sum(nil))

	// 构建图片上传路径
	objectPath := fmt.Sprintf("%s/%s.%s", ou.OSSBucket, md5, ext)

	// 图片上传到 OSS 指定路径
	err := ou.bucket.PutObject(objectPath, bytes.NewReader(image))
	if err != nil {
		return "", err
	}

	return objectPath, nil
}

// GetImageURL: 构建图片的URL
func (ou *UploadWithOSS) GetImageURL(objectPath string) string {
	return fmt.Sprintf("%s/%s", ou.Endpoint, ou.OSSBucket)
}

// DeleteImage: 删除已上传的图片
func (ou *UploadWithOSS) DeleteImage(objectPath string) error {
	err := ou.bucket.DeleteObject(objectPath)
	return err
}
