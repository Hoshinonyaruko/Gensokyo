package images

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/oss"
	"github.com/hoshinonyaruko/gensokyo/protobuf"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
	"google.golang.org/protobuf/proto"
)

// uploadMedia 上传媒体并返回FileInfo
func uploadMedia(ctx context.Context, groupID string, richMediaMessage *dto.RichMediaMessage, apiv2 openapi.OpenAPI) (string, error) {
	// 调用API来上传媒体
	messageReturn, err := apiv2.PostGroupMessage(ctx, groupID, richMediaMessage)
	if err != nil {
		return "", err
	}
	// 返回上传后的FileInfo
	return messageReturn.MediaResponse.FileInfo, nil
}

func isNumeric(s string) bool {
	// 使用正则表达式检查字符串是否只包含数字
	return regexp.MustCompile(`^\d+$`).MatchString(s)
}

// UploadBase64ImageToServer 将base64图片通过lotus转换成url
func UploadBase64ImageToServer(msgid string, base64Image string, groupID string, apiv2 openapi.OpenAPI) (string, uint64, uint32, uint32, error) {
	var picURL string
	var err error
	extraPicAuditingType := config.GetOssType()

	switch extraPicAuditingType {
	case 0:
		picURL, err = originalUploadBehavior(base64Image)
	case 1:
		picURL, err = oss.UploadAndAuditImage(base64Image) // 腾讯
	case 2:
		picURL, err = oss.UploadAndAuditImageB(base64Image) // 百度
	case 3:
		picURL, err = oss.UploadAndAuditImageA(base64Image) // 阿里
	default:
		return "", 0, 0, 0, errors.New("invalid extraPicAuditingType")
	}
	if err != nil {
		return "", 0, 0, 0, err
	}

	if config.GetImgUpApiVtv2() && groupID != "" {

		if msgid == "" {
			msgid = echo.GetLazyMessagesId(groupID)
		}
		if isNumeric(groupID) {
			// 检查groupID是否为纯数字构成
			originalGroupID, err := idmap.RetrieveRowByIDv2(groupID)
			if err != nil {
				log.Printf("Error retrieving original GroupID: %v", err)
				return "", 0, 0, 0, nil
			}
			log.Printf("通过idmap获取的originalGroupID: %v", originalGroupID)

			// 用originalGroupID更新groupID
			groupID = originalGroupID
		}
		richMediaMessage := &dto.RichMediaMessage{
			EventID:    msgid,
			FileType:   1, // 1代表图片
			URL:        picURL,
			Content:    "", // 这个字段文档没有了
			SrvSendMsg: false,
		}
		fileInfo, err := uploadMedia(context.TODO(), groupID, richMediaMessage, apiv2)
		if err != nil {
			return "", 0, 0, 0, err
		}

		// 将Base64字符串解码为二进制
		fileInfoBytes, err := base64.StdEncoding.DecodeString(fileInfo)
		if err != nil {
			log.Fatalf("Failed to decode Base64 string: %v", err)
		}

		// 初始化Proto消息类型
		var mainMessage protobuf.Main

		// 解析二进制数据到Proto消息
		err = proto.Unmarshal(fileInfoBytes, &mainMessage)
		if err != nil {
			log.Fatalf("Failed to unmarshal Proto message: %v", err)
		}

		// 从Proto消息中读取值
		realGroupID := mainMessage.GetA().GetB().GetInfo().GetDetail().GetGroupInfo().GetGroupNumber()
		downloadURL := mainMessage.GetA().GetImageData().GetImageInfo().GetUrl()
		width := mainMessage.GetA().GetImageData().GetWidth()
		height := mainMessage.GetA().GetImageData().GetHeight()

		// 打印读取的值
		log.Printf("RealGroup ID: %d\n", realGroupID)
		log.Printf("Download URL: %s, Width: %d, Height: %d\n", downloadURL, width, height)

		// 根据需要返回适当的值
		return downloadURL, realGroupID, width, height, nil

	}

	return picURL, 0, 0, 0, nil
}

// 将base64语音通过lotus转换成url
func UploadBase64RecordToServer(base64Record string) (string, error) {
	extraPicAuditingType := config.GetOssType()

	// 根据不同的extraPicAuditingType值来调整函数行为
	switch extraPicAuditingType {
	case 0:
		// 原有的函数行为
		return originalUploadBehaviorRecord(base64Record)
	case 1:
		return oss.UploadAndAuditRecord(base64Record) //腾讯
	case 2:
		return oss.UploadAndAuditRecord(base64Record) //百度
	case 3:
		return oss.UploadAndAuditRecord(base64Record) //阿里
	default:
		return "", errors.New("invalid extraPicAuditingType")
	}
}

func originalUploadBehavior(base64Image string) (string, error) {
	// 原有的UploadBase64ImageToServer函数的实现
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

// 将base64语音通过lotus转换成url
func originalUploadBehaviorRecord(base64Image string) (string, error) {
	// 根据serverPort确定协议
	protocol := "http"
	serverPort := config.GetPortValue()
	if serverPort == "443" {
		protocol = "https"
	}

	if config.GetLotusValue() {
		serverDir := config.GetServer_dir()
		url := fmt.Sprintf("%s://%s:%s/uploadrecord", protocol, serverDir, serverPort)

		resp, err := postRecordToServer(base64Image, url)
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
		url := fmt.Sprintf("%s://127.0.0.1:%s/uploadrecord", protocol, serverPort)

		resp, err := postRecordToServer(base64Image, url)
		if err != nil {
			return "", err
		}
		return resp, nil
	}
	return "", errors.New("local server uses a private address; image record failed")
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

// 请求语音床api(图床就是lolus为false的gensokyo)
func postRecordToServer(base64Image, targetURL string) (string, error) {
	data := url.Values{}
	data.Set("base64Record", base64Image) // 修改字段名以与服务器匹配

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
