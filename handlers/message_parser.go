package handlers

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/dto"
)

var BotID string
var AppID string

// 定义响应结构体
type ServerResponse struct {
	Data struct {
		MessageID int `json:"message_id"`
	} `json:"data"`
	Message string `json:"message"`
	RetCode int    `json:"retcode"`
	Status  string `json:"status"`
	Echo    string `json:"echo"`
}

// 发送成功回执 todo 返回可互转的messageid
func SendResponse(client callapi.Client, err error, message *callapi.ActionMessage) error {
	// 设置响应值
	response := ServerResponse{}
	response.Data.MessageID = 0 // todo 实现messageid转换
	response.Echo = string(message.Echo)
	if err != nil {
		response.Message = err.Error() // 可选：在响应中添加错误消息
		//response.RetCode = -1          // 可以是任何非零值，表示出错
		//response.Status = "failed"
		response.RetCode = 0 //官方api审核异步的 审核中默认返回失败,但其实信息发送成功了
		response.Status = "ok"
	} else {
		response.Message = ""
		response.RetCode = 0
		response.Status = "ok"
	}

	// 转化为map并发送
	outputMap := structToMap(response)

	sendErr := client.SendMessage(outputMap)
	if sendErr != nil {
		log.Printf("Error sending message via client: %v", sendErr)
		return sendErr
	}

	log.Printf("发送成功回执: %+v", outputMap)
	return nil
}

func parseMessageContent(paramsMessage callapi.ParamsContent) (string, map[string][]string) {
	messageText := ""

	switch message := paramsMessage.Message.(type) {
	case string:
		fmt.Printf("params.message is a string\n")
		messageText = message
	case []interface{}:
		//多个映射组成的切片
		fmt.Printf("params.message is a slice (segment_type_koishi)\n")
		for _, segment := range message {
			segmentMap, ok := segment.(map[string]interface{})
			if !ok {
				continue
			}

			segmentType, ok := segmentMap["type"].(string)
			if !ok {
				continue
			}

			segmentContent := ""
			switch segmentType {
			case "text":
				segmentContent, _ = segmentMap["data"].(map[string]interface{})["text"].(string)
			case "image":
				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
				segmentContent = "[CQ:image,file=" + fileContent + "]"
			case "voice":
				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
				segmentContent = "[CQ:record,file=" + fileContent + "]"
			case "at":
				qqNumber, _ := segmentMap["data"].(map[string]interface{})["qq"].(string)
				segmentContent = "[CQ:at,qq=" + qqNumber + "]"
			}

			messageText += segmentContent
		}
	case map[string]interface{}:
		//单个映射
		fmt.Printf("params.message is a map (segment_type_trss)\n")
		messageType, _ := message["type"].(string)
		switch messageType {
		case "text":
			messageText, _ = message["data"].(map[string]interface{})["text"].(string)
		case "image":
			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
			messageText = "[CQ:image,file=" + fileContent + "]"
		case "voice":
			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
			messageText = "[CQ:record,file=" + fileContent + "]"
		case "at":
			qqNumber, _ := message["data"].(map[string]interface{})["qq"].(string)
			messageText = "[CQ:at,qq=" + qqNumber + "]"
		}
	default:
		log.Println("Unsupported message format: params.message field is not a string, map or slice")
	}

	// 正则表达式部分
	localImagePattern := regexp.MustCompile(`\[CQ:image,file=file:///([^\]]+?)\]`)
	urlImagePattern := regexp.MustCompile(`\[CQ:image,file=http://(.+)\]`)
	base64ImagePattern := regexp.MustCompile(`\[CQ:image,file=base64://(.+)\]`)
	base64RecordPattern := regexp.MustCompile(`\[CQ:record,file=base64://(.+)\]`)

	patterns := []struct {
		key     string
		pattern *regexp.Regexp
	}{
		{"local_image", localImagePattern},
		{"url_image", urlImagePattern},
		{"base64_image", base64ImagePattern},
		{"base64_record", base64RecordPattern},
	}

	foundItems := make(map[string][]string)
	for _, pattern := range patterns {
		matches := pattern.pattern.FindAllStringSubmatch(messageText, -1)
		for _, match := range matches {
			if len(match) > 1 {
				foundItems[pattern.key] = append(foundItems[pattern.key], match[1])
				messageText = pattern.pattern.ReplaceAllString(messageText, "")
			}
		}
	}
	//处理at
	messageText = transformMessageText(messageText)

	return messageText, foundItems
}

func transformMessageText(messageText string) string {
	// 使用正则表达式来查找所有[CQ:at,qq=数字]的模式
	re := regexp.MustCompile(`\[CQ:at,qq=(\d+)\]`)
	// 使用正则表达式来替换找到的模式为<@!数字>
	return re.ReplaceAllStringFunc(messageText, func(m string) string {
		submatches := re.FindStringSubmatch(m)
		if len(submatches) > 1 {
			return "<@!" + submatches[1] + ">"
		}
		return m
	})
}

// 处理at和其他定形文到onebotv11格式(cq码)
func RevertTransformedText(data interface{}) string {
	var msg *dto.Message
	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSDirectMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSC2CMessageData:
		msg = (*dto.Message)(v)
	default:
		return ""
	}
	messageText := strings.TrimSpace(msg.Content)

	// 将messageText里的BotID替换成AppID
	messageText = strings.ReplaceAll(messageText, BotID, AppID)

	// 使用正则表达式来查找所有<@!数字>的模式
	re := regexp.MustCompile(`<@!(\d+)>`)
	// 使用正则表达式来替换找到的模式为[CQ:at,qq=数字]
	messageText = re.ReplaceAllStringFunc(messageText, func(m string) string {
		submatches := re.FindStringSubmatch(m)
		if len(submatches) > 1 {
			return "[CQ:at,qq=" + submatches[1] + "]"
		}
		return m
	})

	// 处理图片附件
	for _, attachment := range msg.Attachments {
		if strings.HasPrefix(attachment.ContentType, "image/") {
			// 获取文件的后缀名
			ext := filepath.Ext(attachment.FileName)
			md5name := strings.TrimSuffix(attachment.FileName, ext)
			imageCQ := "[CQ:image,file=" + md5name + ".image,subType=0,url=" + attachment.URL + "]"
			messageText += imageCQ
		}
	}

	return messageText
}

// 将收到的data.content转换为message segment todo,群场景不支持受图片,频道场景的图片可以拼一下
func ConvertToSegmentedMessage(data interface{}) []map[string]interface{} {
	// 强制类型转换，获取Message结构
	var msg *dto.Message
	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSDirectMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSC2CMessageData:
		msg = (*dto.Message)(v)
	default:
		return nil
	}

	var messageSegments []map[string]interface{}

	// 处理Attachments字段来构建图片消息
	for _, attachment := range msg.Attachments {
		imageFileMD5 := attachment.FileName
		for _, ext := range []string{"{", "}", ".png", ".jpg", ".gif", "-"} {
			imageFileMD5 = strings.ReplaceAll(imageFileMD5, ext, "")
		}
		imageSegment := map[string]interface{}{
			"type": "image",
			"data": map[string]interface{}{
				"file":    imageFileMD5 + ".image",
				"subType": "0",
				"url":     attachment.URL,
			},
		}
		messageSegments = append(messageSegments, imageSegment)

		// 在msg.Content中替换旧的图片链接
		newImagePattern := "[CQ:image,file=" + attachment.URL + "]"
		msg.Content = msg.Content + newImagePattern
	}

	// 使用正则表达式查找所有的[@数字]格式
	r := regexp.MustCompile(`<@!(\d+)>`)
	atMatches := r.FindAllStringSubmatch(msg.Content, -1)

	for _, match := range atMatches {
		// 构建at部分的映射并加入到messageSegments
		atSegment := map[string]interface{}{
			"type": "at",
			"data": map[string]interface{}{
				"qq": match[1],
			},
		}
		messageSegments = append(messageSegments, atSegment)

		// 从原始内容中移除at部分
		msg.Content = strings.Replace(msg.Content, match[0], "", 1)
	}

	// 如果还有其他内容，那么这些内容被视为文本部分
	if msg.Content != "" {
		textSegment := map[string]interface{}{
			"type": "text",
			"data": map[string]interface{}{
				"text": msg.Content,
			},
		}
		messageSegments = append(messageSegments, textSegment)
	}

	return messageSegments
}
