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
	"strconv"
	"strings"
	"sync"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/oss"
	"github.com/hoshinonyaruko/gensokyo/protobuf"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
	"google.golang.org/protobuf/proto"
)

// 包级私有变量，用于存储当前URL索引
var (
	currentURLIndex int
	urlsMutex       sync.Mutex
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

// uploadMedia 上传媒体并返回FileInfo
func uploadMediaPrivate(ctx context.Context, UserID string, richMediaMessage *dto.RichMediaMessage, apiv2 openapi.OpenAPI) (string, error) {
	// 调用API来上传媒体
	messageReturn, err := apiv2.PostC2CMessage(ctx, UserID, richMediaMessage)
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
	// 检查是否应该使用全局服务器临时QQ群的特殊上传行为
	if config.GetGlobalServerTempQQguild() {
		// 直接调用UploadBehaviorV3
		downloadURL, width, height, err := UploadBehaviorV3(base64Image)
		if err != nil {
			log.Printf("Error UploadBehaviorV3: %v", err)
			return "", 0, 0, 0, nil
		}
		return downloadURL, 0, width, height, nil
	}

	//v2接口是否使用base64
	if !config.GetUploadPicV2Base64() {
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
	}

	if config.GetImgUpApiVtv2() && groupID != "" {

		if isNumeric(groupID) {
			//用转换前的群号获取msgid
			if msgid == "" {
				msgid = echo.GetLazyMessagesId(groupID)
			}
			// 检查groupID是否为纯数字构成 RetrieveRowByIDv2是通用逻辑，也可以将userid还原为32位数originaluserid
			// 但没有私信权限，故没有测试
			originalGroupID, err := idmap.RetrieveRowByIDv2(groupID)
			if err != nil {
				log.Printf("Error retrieving original GroupID: %v", err)
				return picURL, 0, 0, 0, nil
			}
			log.Printf("通过idmap获取的originalGroupID: %v", originalGroupID)

			// 用originalGroupID更新groupID
			groupID = originalGroupID
		} else {
			// 映射str的GroupID到int
			GroupID64, err := idmap.StoreIDv2(groupID)
			if err != nil {
				log.Printf("failed to convert ChannelID to int: %v", err)
				return picURL, 0, 0, 0, nil
			}
			groupIDTemp := strconv.FormatInt(GroupID64, 10)
			//用数字的群号获取msgid
			if msgid == "" {
				msgid = echo.GetLazyMessagesId(groupIDTemp)
			}
		}
		var richMediaMessage *dto.RichMediaMessage
		if !config.GetUploadPicV2Base64() {
			richMediaMessage = &dto.RichMediaMessage{
				EventID:    msgid,
				FileType:   1, // 1代表图片
				URL:        picURL,
				Content:    "", // 这个字段文档没有了
				SrvSendMsg: false,
			}
		} else {
			richMediaMessage = &dto.RichMediaMessage{
				EventID:    msgid,
				FileType:   1, // 1代表图片
				FileData:   base64Image,
				Content:    "", // 这个字段文档没有了
				SrvSendMsg: false,
			}
		}

		var fileInfo string
		//尝试群聊发图
		fileInfo, err = uploadMedia(context.TODO(), groupID, richMediaMessage, apiv2)
		if err != nil {
			//尝试私信发图
			fileInfo, err = uploadMediaPrivate(context.TODO(), groupID, richMediaMessage, apiv2)
			if err != nil {
				//返回原始图片url
				return picURL, 0, 0, 0, nil
			}
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
		//https的地址不能放到md里
		//downloadURL = "https://multimedia.nt.qq.com.cn" + downloadURL
		// 将 downloadURL 中的所有下划线 "_" 替换为 "%5f"
		downloadURL = strings.Replace(downloadURL, "_", "%5f", -1)
		downloadURL = "http://multimedia.nt.qq.com" + downloadURL
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

// TransferUrlToServerUrlr 将url转化为ntv2链接
func TransferUrlToServerUrl(msgid string, url string, groupID string, apiv2 openapi.OpenAPI) (string, uint64, uint32, uint32, error) {
	var err error

	if isNumeric(groupID) {
		//用转换前的群号获取msgid
		if msgid == "" {
			msgid = echo.GetLazyMessagesId(groupID)
		}
		// 检查groupID是否为纯数字构成 RetrieveRowByIDv2是通用逻辑，也可以将userid还原为32位数originaluserid
		// 但没有私信权限，故没有测试
		originalGroupID, err := idmap.RetrieveRowByIDv2(groupID)
		if err != nil {
			log.Printf("Error retrieving original GroupID: %v", err)
			return url, 0, 0, 0, nil
		}
		log.Printf("通过idmap获取的originalGroupID: %v", originalGroupID)

		// 用originalGroupID更新groupID
		groupID = originalGroupID
	} else {
		// 映射str的GroupID到int
		GroupID64, err := idmap.StoreIDv2(groupID)
		if err != nil {
			log.Printf("failed to convert ChannelID to int: %v", err)
			return url, 0, 0, 0, nil
		}
		groupIDTemp := strconv.FormatInt(GroupID64, 10)
		//用数字的群号获取msgid
		if msgid == "" {
			msgid = echo.GetLazyMessagesId(groupIDTemp)
		}
	}

	richMediaMessage := &dto.RichMediaMessage{
		EventID:    msgid,
		FileType:   1, // 1代表图片
		URL:        url,
		Content:    "", // 这个字段文档没有了
		SrvSendMsg: false,
	}

	var fileInfo string
	//尝试群聊发图
	fileInfo, err = uploadMedia(context.TODO(), groupID, richMediaMessage, apiv2)
	if err != nil {
		//尝试私信发图
		fileInfo, err = uploadMediaPrivate(context.TODO(), groupID, richMediaMessage, apiv2)
		if err != nil {
			//返回原始图片url
			return url, 0, 0, 0, nil
		}
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
	//https的地址不能放到md里
	//downloadURL = "https://multimedia.nt.qq.com.cn" + downloadURL
	// 将 downloadURL 中的所有下划线 "_" 替换为 "%5f"
	downloadURL = strings.Replace(downloadURL, "_", "%5f", -1)
	downloadURL = "http://multimedia.nt.qq.com" + downloadURL
	width := mainMessage.GetA().GetImageData().GetWidth()
	height := mainMessage.GetA().GetImageData().GetHeight()

	// 打印读取的值
	log.Printf("RealGroup ID: %d\n", realGroupID)
	log.Printf("Download URL: %s, Width: %d, Height: %d\n", downloadURL, width, height)

	// 根据需要返回适当的值
	return downloadURL, realGroupID, width, height, nil

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

func UploadBehaviorV3(base64Image string) (string, uint32, uint32, error) {
	urls := config.GetServerTempQQguildPool()
	if len(urls) > 0 {
		urlsMutex.Lock()
		url := urls[currentURLIndex]
		currentURLIndex = (currentURLIndex + 1) % len(urls)
		urlsMutex.Unlock()

		resp, width, height, err := postImageToServerV3(base64Image, url)
		if err != nil {
			return "", 0, 0, err
		}
		return resp, width, height, nil
	} else {
		protocol := "http"
		serverPort := config.GetPortValue()
		if serverPort == "443" {
			protocol = "https"
		}

		serverDir := config.GetServer_dir()
		url := fmt.Sprintf("%s://%s:%s/uploadpicv3", protocol, serverDir, serverPort)

		if config.GetLotusValue() {
			resp, width, height, err := postImageToServerV3(base64Image, url)
			if err != nil {
				return "", 0, 0, err
			}
			return resp, width, height, nil
		} else {
			if serverPort == "443" {
				protocol = "http"
				serverPort = "444"
			}
			url = fmt.Sprintf("%s://127.0.0.1:%s/uploadpicv3", protocol, serverPort)

			resp, width, height, err := postImageToServerV3(base64Image, url)
			if err != nil {
				return "", 0, 0, err
			}
			return resp, width, height, nil
		}
	}
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

// 请求图床api(图床就是lolus为false的gensokyo)
func postImageToServerV3(base64Image, targetURL string) (string, uint32, uint32, error) {
	data := url.Values{}
	channelID := config.GetServerTempQQguild()
	data.Set("base64Image", base64Image) // 修改字段名以与服务器匹配
	data.Set("channelID", channelID)     // 修改字段名以与服务器匹配

	resp, err := http.PostForm(targetURL, data)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, 0, fmt.Errorf("error response from server: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to read response body: %v", err)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return "", 0, 0, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	url, okURL := responseMap["url"].(string)
	width, okWidth := responseMap["width"].(float64) // JSON numbers are decoded as float64
	height, okHeight := responseMap["height"].(float64)
	if !okURL {
		return "", 0, 0, fmt.Errorf("uRL not found in response")
	}
	if !okWidth || !okHeight {
		return "", 0, 0, fmt.Errorf("width or Height not found in response")
	}

	return url, uint32(width), uint32(height), nil
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
