//go:build (linux || (windows && !arm && !arm64) || darwin) && (386 || amd64 || arm || arm64) && !race && !nosilk
// +build linux windows,!arm,!arm64 darwin
// +build 386 amd64 arm arm64
// +build !race
// +build !nosilk

package silk

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/wdvxdr1123/go-silk"
)

const (
	// HeaderAmr AMR文件头
	HeaderAmr = "#!AMR"
	// HeaderSilk Silkv3文件头
	HeaderSilk = "\x02#!SILK_V3"
)

const silkCachePath = "data/cache"

const limit = 4 * 1024

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

	// 3. 转silk
	pcm, err := os.ReadFile(pcmPath)
	if err != nil {
		mylog.Printf("read pcm file err")
		return nil
	}
	silkWav, err = silk.EncodePcmBuffToSilkv2(pcm, sampleRate, bitRate, true, false, 2)
	if err != nil {
		mylog.Printf("silk encode error:%v", err)
		return nil
	}
	silkPath := path.Join(silkCachePath, tempName+".silk")
	err = os.WriteFile(silkPath, silkWav, 0o666)
	if err != nil {
		mylog.Printf("silk encode error2")
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
