package images

import (
	"bytes"

	"github.com/hoshinonyaruko/gensokyo/config"
)

// 默认压缩参数
const (
	defaultQualityStep = 10
	defaultMinQuality  = 25
	defaultMaxQuality  = 75
)

// CompressSingleImage 接收一个图片的 []byte 数据，并根据设定阈值返回压缩后的数据或原始数据。
func CompressSingleImage(imageBytes []byte) ([]byte, error) {
	// 获取压缩阈值
	thresholdKB := config.GetImageLimit()

	// 如果阈值为0，则直接返回原始图片数据，不进行压缩
	if thresholdKB == 0 {
		return imageBytes, nil
	}

	// 创建压缩器实例
	compressor := NewCompressor(thresholdKB, defaultQualityStep, defaultMinQuality, defaultMaxQuality)

	// 创建一个读取器来读取 imageBytes 数据
	reader := bytes.NewReader(imageBytes)

	// 调用 CompressImage 方法来压缩图片
	compressedImage, err := compressor.CompressImage(reader)
	if err != nil {
		return nil, err // 压缩出错时返回错误
	}

	return compressedImage, nil // 返回压缩后的图片数据
}
