package images

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
)

// 上传图片到 OSSFS 所需配置信息
type UploadWithOSSFS struct {

	// OSS 域名
	Domain string

	// OSSFS 本地映射路径
	OSSFSPath string
}

// NewUploadWithOSSFS: 创建 UploadWithOSSFS 的新实例
func NewUploadWithOSSFS(config map[string]string) *UploadWithOSSFS {

	return &UploadWithOSSFS{

		Domain: config["domain"],

		OSSFSPath: config["ossfs_path"],
	}

}

// SavePicture: 保存给定的图像数据以及指定的扩展名
func (u *UploadWithOSSFS) SavePicture(image []byte, ext string) (string, error) {

	// 计算图像数据的 MD5 值
	md5Hash := md5.New()
	md5Hash.Write(image)
	md5 := hex.EncodeToString(md5Hash.Sum(nil))

	// 构建图片存储路径
	path := fmt.Sprintf("%s/%s.%s", u.OSSFSPath, md5, ext)

	// 图像文件上传至 OSSFS 挂载文件夹
	err := os.WriteFile(path, image, 0644)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", md5, ext), nil
}

// GetURL: 构建图片直链 URL
func (u *UploadWithOSSFS) GetURL(name string) string {
	return fmt.Sprintf("%s/%s", u.Domain, name)
}

// DelPicture: 删除上传完成的图片
func (u *UploadWithOSSFS) DelPicture(name string) error {

	// 删除上传完成的图片文件
	err := os.Remove(fmt.Sprintf("%s/%s", u.OSSFSPath, name))
	if err != nil {
		return err
	}

	return nil
}
