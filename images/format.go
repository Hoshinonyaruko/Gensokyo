package images

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 宽度 高度
func GetImageDimensions(url string) (int, int, error) {
	// 原有的后缀判断逻辑
	if strings.HasSuffix(url, ".png") {
		return getPNGDimensions(url)
	} else if strings.HasSuffix(url, ".jpg") || strings.HasSuffix(url, ".jpeg") {
		return getJpegDimensions(url)
	} else if strings.HasSuffix(url, ".gif") {
		return getGIFDimensions(url)
	}

	// 如果图片格式不受支持，则尝试其他方法
	methods := []func(string) (int, int, error){getPNGDimensions, getJpegDimensions, getGIFDimensions}
	for _, method := range methods {
		width, height, err := method(url)
		if err == nil && (width != 0 || height != 0) {
			return width, height, nil
		}
	}

	return 0, 0, fmt.Errorf("unsupported image format or failed to get dimensions")
}

func getPNGDimensions(url string) (int, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("error occurred while making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to download image with status code: %d", resp.StatusCode)
	}

	// 只读取前33字节
	buffer := make([]byte, 33)
	_, err = io.ReadFull(resp.Body, buffer)
	if err != nil {
		return 0, 0, fmt.Errorf("error occurred while reading the header: %w", err)
	}

	// 检查PNG文件头
	if bytes.HasPrefix(buffer, []byte("\x89PNG\r\n\x1a\n")) {
		var width, height int32
		reader := bytes.NewReader(buffer[16:24])
		err := binary.Read(reader, binary.BigEndian, &width)
		if err != nil {
			return 0, 0, fmt.Errorf("error occurred while reading width: %w", err)
		}
		err = binary.Read(reader, binary.BigEndian, &height)
		if err != nil {
			return 0, 0, fmt.Errorf("error occurred while reading height: %w", err)
		}
		return int(width), int(height), nil
	}

	width, height, err := getJpegDimensions(url)
	return width, height, err
}

func getJpegDimensions(url string) (int, int, error) {
	response, err := http.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("error occurred while making request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return 0, 0, fmt.Errorf("failed to download image with status code: %d", response.StatusCode)
	}

	bytesRead := 0 // 用于跟踪读取的字节数
	for {
		b := make([]byte, 1)
		_, err := response.Body.Read(b)
		bytesRead++ // 更新读取的字节数
		if err == io.EOF {
			return 0, 0, fmt.Errorf("reached end of file before finding dimensions")
		} else if err != nil {
			return 0, 0, fmt.Errorf("error occurred while reading byte: %w", err)
		}

		if b[0] == 0xFF {
			nextByte := make([]byte, 1)
			_, err := response.Body.Read(nextByte)
			if err == io.EOF {
				return 0, 0, fmt.Errorf("reached end of file before finding SOI marker")
			} else if err != nil {
				return 0, 0, fmt.Errorf("error occurred while reading next byte: %w", err)
			}

			if nextByte[0] == 0xD8 { // SOI
				continue
			} else if (nextByte[0] >= 0xC0 && nextByte[0] <= 0xCF) && nextByte[0] != 0xC4 && nextByte[0] != 0xC8 && nextByte[0] != 0xCC {
				c := make([]byte, 3)
				_, err := response.Body.Read(c)
				if err != nil {
					return 0, 0, fmt.Errorf("error occurred while skipping bytes: %w", err)
				}
				heightBytes := make([]byte, 2)
				widthBytes := make([]byte, 2)
				_, err = response.Body.Read(heightBytes)
				if err != nil {
					return 0, 0, fmt.Errorf("error occurred while reading height: %w", err)
				}
				_, err = response.Body.Read(widthBytes)
				if err != nil {
					return 0, 0, fmt.Errorf("error occurred while reading width: %w", err)
				}
				// 使用binary.BigEndian.Uint16将字节转换为uint16
				height := binary.BigEndian.Uint16(heightBytes)
				width := binary.BigEndian.Uint16(widthBytes)
				return int(height), int(width), nil
			} else {
				time.Sleep(5 * time.Millisecond)
				lengthBytes := make([]byte, 2)
				_, err := response.Body.Read(lengthBytes)
				if err != nil {
					return 0, 0, fmt.Errorf("error occurred while reading segment length: %w", err)
				}

				length := binary.BigEndian.Uint16(lengthBytes)
				bytesToSkip := int(length) - 2

				// 循环读取并跳过指定的字节数
				for bytesToSkip > 0 {
					bufferSize := bytesToSkip
					if bufferSize > 512 { // 可以调整这个值以优化性能和内存使用
						bufferSize = 512 //如果成功率低 就继续减少它
					}
					buffer := make([]byte, bufferSize)
					n, err := response.Body.Read(buffer)
					if err != nil {
						if err == io.EOF {
							break // 数据流结束
						}
						return 0, 0, fmt.Errorf("error occurred while skipping segment: %w", err)
					}
					bytesToSkip -= n
				}
			}
		}
	}
}

func getGIFDimensions(url string) (int, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("error occurred while making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to download image with status code: %d", resp.StatusCode)
	}

	// 仅读取前10字节
	buffer := make([]byte, 10)
	_, err = io.ReadFull(resp.Body, buffer)
	if err != nil {
		return 0, 0, fmt.Errorf("error occurred while reading the header: %w", err)
	}

	if bytes.HasPrefix(buffer, []byte("GIF")) {
		var width, height int16
		reader := bytes.NewReader(buffer[6:10])
		err := binary.Read(reader, binary.LittleEndian, &width)
		if err != nil {
			return 0, 0, fmt.Errorf("error occurred while reading width: %w", err)
		}
		err = binary.Read(reader, binary.LittleEndian, &height)
		if err != nil {
			return 0, 0, fmt.Errorf("error occurred while reading height: %w", err)
		}
		return int(width), int(height), nil
	}

	return 0, 0, fmt.Errorf("not a valid GIF file")
}
