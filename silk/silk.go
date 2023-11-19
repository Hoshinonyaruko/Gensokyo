package silk

import (
	"bytes"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

//go:embed exec/*
var silkCodecs embed.FS

const (
	// HeaderAmr AMR文件头
	HeaderAmr = "#!AMR"
	// HeaderSilk Silkv3文件头
	HeaderSilk = "\x02#!SILK_V3"
)

const silkCachePath = "data/cache"

const limit = 4 * 1024

func getSilkCodecPath() (string, error) {
	var codecFileName string
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			codecFileName = "silk_codec-windows-static-x64.exe"
		case "386":
			codecFileName = "silk_codec-windows-static-x86.exe"
		default:
			return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			codecFileName = "silk_codec-linux-x64"
		case "arm64":
			codecFileName = "silk_codec-linux-arm64"
		default:
			return "", fmt.Errorf("unsupported architecture for Linux: %s", runtime.GOARCH)
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			codecFileName = "silk_codec-macos"
		case "arm64":
			codecFileName = "silk_codec-macos"
		default:
			return "", fmt.Errorf("unsupported architecture for macOS: %s", runtime.GOARCH)
		}
	case "android":
		switch runtime.GOARCH {
		case "arm64":
			codecFileName = "silk_codec-android-arm64"
		case "x86":
			codecFileName = "silk_codec-android-x86"
		case "x86_64":
			codecFileName = "silk_codec-android-x86_64"
		default:
			return "", fmt.Errorf("unsupported architecture for macOS: %s", runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return "exec/" + codecFileName, nil
}

// EncoderSilk 将音频编码为Silk
func EncoderSilk(data []byte) []byte {
	h := md5.New()
	_, err := h.Write(data)
	if err != nil {
		mylog.Printf("calc md5 failed")
		return nil
	}
	tempName := hex.EncodeToString(h.Sum(nil))
	slk := encode(data, tempName)
	return slk
}

// EncodeMP4 将给定视频文件编码为MP4
func EncodeMP4(src string, dst string) error {
	cmd1 := exec.Command("ffmpeg", "-i", src, "-y", "-c", "copy", "-map", "0", dst)
	if errors.Is(cmd1.Err, exec.ErrDot) {
		cmd1.Err = nil
	}
	err := cmd1.Run()
	if err != nil {
		cmd2 := exec.Command("ffmpeg", "-i", src, "-y", "-c:v", "h264", "-c:a", "mp3", dst)
		if errors.Is(cmd2.Err, exec.ErrDot) {
			cmd2.Err = nil
		}
		mylog.Printf("convert mp4 failed")
		return err
	}
	return err
}

// ExtractCover 获取给定视频文件的Cover
func ExtractCover(src string, target string) error {
	cmd := exec.Command("ffmpeg", "-i", src, "-y", "-ss", "0", "-frames:v", "1", target)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	mylog.Printf("extract video cover failed")
	return nil
}

// createDirectoryIfNotExists 检查目录是否存在，如果不存在则创建它
func createDirectoryIfNotExists(directoryPath string) error {
	_, err := os.Stat(directoryPath)
	if os.IsNotExist(err) {
		// 目录不存在，创建它
		err := os.MkdirAll(directoryPath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

// encode 将音频编码为Silk
func encode(record []byte, tempName string) (silkWav []byte) {
	// 0. 创建缓存目录
	err := createDirectoryIfNotExists(silkCachePath)
	if err != nil {
		fmt.Printf("创建语音缓存目录失败：%v\n", err)
	}
	// 1. 写入缓存文件
	rawPath := path.Join(silkCachePath, tempName+".wav")
	err = os.WriteFile(rawPath, record, os.ModePerm)
	if err != nil {
		mylog.Errorf("write temp file error")
		return nil
	}
	defer os.Remove(rawPath)

	// 2.转换pcm
	sampleRate := config.GetRecordSampleRate() // 获取采样率
	bitRate := config.GetRecordBitRate()       // 获取比特率
	mylog.Printf("sampleRate%v", sampleRate)
	mylog.Printf("bitRate%v", bitRate)
	pcmPath := path.Join(silkCachePath, tempName+".pcm")
	cmd := exec.Command("ffmpeg", "-i", rawPath, "-f", "s16le", "-ar", strconv.Itoa(sampleRate), "-ac", "1", pcmPath)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	if err = cmd.Run(); err != nil {
		mylog.Errorf("convert pcm file error")
		return nil
	}
	defer os.Remove(pcmPath)

	//todo 有大佬可以试试完善go-silk 这部分编码转换
	//努力了很久,都没成功播放

	// 3. 转silk
	// pcm, err := os.ReadFile(pcmPath)
	// if err != nil {
	// 	mylog.Printf("read pcm file err")
	// 	return nil
	// }
	// //silkWav, err = silk.EncodePcmBuffToSilkv2(pcm, sampleRate, bitRate, true, false, 2)
	// silkWav, err = silk.EncodePcmBuffToSilk(pcm, sampleRate, bitRate, true)
	// if err != nil {
	// 	mylog.Printf("silk encode error:%v", err)
	// 	return nil
	// }

	silkPath := path.Join(silkCachePath, tempName+".silk")

	// err = os.WriteFile(silkPath, silkWav, 0o666)
	// if err != nil {
	// 	mylog.Printf("silk encode error2")
	// 	return nil
	// }

	// 调用silk_codec转换为Silk

	// 获取silk_codec文件名
	codecFileName, err := getSilkCodecPath()
	if err != nil {
		mylog.Errorf("failed to get codec path: %v", err)
		return nil
	}

	// 从嵌入的文件系统中读取silk_codec二进制文件
	codecData, err := silkCodecs.ReadFile(codecFileName)
	if err != nil {
		mylog.Errorf("failed to read codec data: %v", err)
		return nil
	}

	// 根据操作系统确定临时文件的扩展名
	tempFilePattern := "silk_codec*"
	if runtime.GOOS == "windows" {
		tempFilePattern += ".exe"
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", tempFilePattern)
	if err != nil {
		mylog.Errorf("failed to create temp file: %v", err)
		return nil
	}
	defer os.Remove(tmpFile.Name()) // 清理临时文件

	// 写入二进制数据到临时文件
	if _, err := tmpFile.Write(codecData); err != nil {
		mylog.Errorf("failed to write codec data to temp file: %v", err)
		return nil
	}
	if err := tmpFile.Close(); err != nil {
		mylog.Errorf("failed to close temp file: %v", err)
		return nil
	}

	// 确保临时文件可执行
	if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
		mylog.Errorf("failed to chmod temp file: %v", err)
		return nil
	}

	// 使用临时文件执行silk_codec
	cmd = exec.Command(tmpFile.Name(), "pts", "-i", pcmPath, "-o", silkPath, "-s", "24000")
	if err := cmd.Run(); err != nil {
		mylog.Errorf("silk encode error: %v", err)
		return nil
	}

	// 读取Silk文件
	silkWav, err = os.ReadFile(silkPath)
	if err != nil {
		mylog.Errorf("read silk file error: %v", err)
		return nil
	}

	return silkWav
}

// IsAMRorSILK 判断给定文件是否为Amr或Silk格式
func IsAMRorSILK(b []byte) bool {
	return bytes.HasPrefix(b, []byte(HeaderAmr)) || bytes.HasPrefix(b, []byte(HeaderSilk))
}

// 扫描格式
func scan(r io.ReadSeeker) string {
	_, _ = r.Seek(0, io.SeekStart)
	defer r.Seek(0, io.SeekStart)
	in := make([]byte, limit)
	_, _ = r.Read(in)
	return http.DetectContentType(in)
}

// CheckAudio 判断给定流是否为合法音频
func CheckAudio(r io.ReadSeeker) (string, bool) {
	t := scan(r)
	// std mime type detection is not full supported for audio
	if strings.Contains(t, "text") || strings.Contains(t, "image") {
		return t, false
	}
	return t, true
}

// // resample 将silk重新编码为 24000 bit rate
// func resample(data []byte) []byte {
// 	pcm, err := silk.DecodeSilkBuffToPcm(data, 24000)
// 	if err != nil {
// 		panic(err)
// 	}
// 	data, err = silk.EncodePcmBuffToSilk(pcm, 24000, 24000, true)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return data
// }
