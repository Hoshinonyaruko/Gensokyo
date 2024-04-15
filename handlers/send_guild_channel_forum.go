package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"mvdan.cc/xurls"

	"github.com/hoshinonyaruko/gensokyo/echo"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_guild_channel_forum", HandleSendGuildChannelForum)
}

func HandleSendGuildChannelForum(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	var retmsg string
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}
	// 检查GroupID是否为0
	checkZeroGroupID := func(id interface{}) bool {
		switch v := id.(type) {
		case int:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != "0" // 检查字符串形式的0
		default:
			return true // 如果不是int、int64或string，假定它不为0
		}
	}

	// 检查UserID是否为0
	checkZeroUserID := func(id interface{}) bool {
		switch v := id.(type) {
		case int:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != "0" // 同样检查字符串形式的0
		default:
			return true // 如果不是int、int64或string，假定它不为0
		}
	}

	if msgType == "" && message.Params.GroupID != nil && checkZeroGroupID(message.Params.GroupID) {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
	}
	if msgType == "" && message.Params.UserID != nil && checkZeroUserID(message.Params.UserID) {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}
	if msgType == "" && message.Params.GroupID != nil && checkZeroGroupID(message.Params.GroupID) {
		msgType = GetMessageTypeByGroupidV2(message.Params.GroupID)
	}
	if msgType == "" && message.Params.UserID != nil && checkZeroUserID(message.Params.UserID) {
		msgType = GetMessageTypeByUseridV2(message.Params.UserID)
	}
	// New checks for UserID and GroupID being nil or 0
	if (message.Params.UserID == nil || !checkZeroUserID(message.Params.UserID)) &&
		(message.Params.GroupID == nil || !checkZeroGroupID(message.Params.GroupID)) {
		mylog.Printf("send_group_msgs接收到错误action: %v", message)
		return "", nil
	}

	//当不转换频道信息时(不支持频道私聊)
	if msgType == "" {
		msgType = "forum"
	}
	switch msgType {
	//原生guild信息
	case "forum":
		params := message.Params
		messageText, foundItems := parseMessageContent(params, message, client, api, apiv2)

		channelID := params.ChannelID
		// 使用 echo 获取消息ID

		mylog.Println("频道发帖子messageText:", messageText)

		Forum, err := GenerateForumMessage(foundItems, messageText, apiv2)
		if err != nil {
			mylog.Printf("组合帖子信息失败: %v", err)
		}
		if _, err = api.PostFourm(context.TODO(), channelID.(string), Forum); err != nil {
			mylog.Printf("发送帖子信息失败: %v", err)
		}

		//发送成功回执
		retmsg, _ = SendResponse(client, err, &message, nil, api, apiv2)

	default:
		mylog.Printf("2Unknown message type: %s", msgType)
	}
	return retmsg, nil
}

// GenerateForumMessage 生成帖子消息，类型是2
// func GenerateForumMessage(foundItems map[string][]string, messageText string) (*dto.FourmToCreate, error) {
// 	var forum dto.FourmToCreate

// 	// 设置标题
// 	title := config.GetBotForumTitle()
// 	forum.Title = title

// 	// 使用提供的messageText作为帖子内容
// 	forum.Content = messageText
// 	forum.Format = 1 // 纯文本

// 	mdImages := []string{}

// 	// 检查是否有图片链接
// 	if imageItems, ok := foundItems["url_image"]; ok && len(imageItems) > 0 {
// 		for _, url := range imageItems {
// 			// 获取图片宽高
// 			height, width, err := images.GetImageDimensions(url)
// 			if err != nil {
// 				mylog.Printf("获取图片宽高出错: %v", err)
// 				continue // 如果无法获取宽高，则跳过此图片
// 			}
// 			// 将图片URL转换为Markdown图片格式，并添加宽高信息
// 			imgDesc := fmt.Sprintf("图片 #%dpx #%dpx", width, height)
// 			mdImages = append(mdImages, fmt.Sprintf("![%s](%s)", imgDesc, url))
// 		}
// 	}

// 	// 处理base64图片
// 	if base64Image, ok := foundItems["base64_image"]; ok && len(base64Image) > 0 {
// 		fileImageData, err := base64.StdEncoding.DecodeString(base64Image[0])
// 		if err != nil {
// 			mylog.Printf("failed to decode base64 image: %v", err)
// 			return nil, fmt.Errorf("failed to decode base64 image: %v", err)
// 		}
// 		compressedData, err := images.CompressSingleImage(fileImageData)
// 		if err != nil {
// 			mylog.Printf("Error compressing image: %v", err)
// 			return nil, fmt.Errorf("error compressing image: %v", err)
// 		}
// 		imageURL, err := images.UploadBase64ImageToServer(base64.StdEncoding.EncodeToString(compressedData))
// 		if err != nil {
// 			mylog.Printf("failed to upload base64 image: %v", err)
// 			return nil, fmt.Errorf("failed to upload base64 image: %v", err)
// 		}
// 		// 获取图片宽高
// 		height, width, err := images.GetImageDimensions(imageURL)
// 		if err != nil {
// 			mylog.Printf("获取图片宽高出错: %v", err)
// 			// 如果无法获取宽高，则使用默认描述
// 			mdImages = append(mdImages, fmt.Sprintf("![(默认图片描述)](%s)", imageURL))
// 		} else {
// 			imgDesc := fmt.Sprintf("图片 #%dpx #%dpx", width, height)
// 			mdImages = append(mdImages, fmt.Sprintf("![%s](%s)", imgDesc, imageURL))
// 		}
// 	}

// 	// 如果有图片，则更新内容和格式
// 	if len(mdImages) > 0 {
// 		// 如果已经有文本内容，则在文本后添加图片
// 		if forum.Content != "" {
// 			forum.Content += "\n\n" + strings.Join(mdImages, "\n\n")
// 		} else {
// 			forum.Content = strings.Join(mdImages, "\n\n")
// 		}
// 		forum.Format = 3 // Markdown，因为包含文本和/或图片
// 	}

// 	// 如果没有找到任何内容，返回错误
// 	if forum.Content == "" {
// 		return nil, fmt.Errorf("no valid content found")
// 	}

// 	return &forum, nil
// }

// GenerateForumMessage 生成帖子消息
func GenerateForumMessage(foundItems map[string][]string, messageText string, apiv2 openapi.OpenAPI) (*dto.FourmToCreate, error) {
	var forum dto.FourmToCreate

	// 设置标题
	title := config.GetBotForumTitle()
	forum.Title = title

	// 初始化富文本内容结构
	var richText struct {
		Paragraphs []struct {
			Elems []interface{} `json:"elems"`
		} `json:"paragraphs"`
	}

	// 使用xurls正则表达式查找所有的URL
	foundURLs := xurls.Relaxed.FindAllString(messageText, -1)

	// 移除文本中的URL
	messageText = xurls.Relaxed.ReplaceAllStringFunc(messageText, func(originalURL string) string {
		return ""
	})

	// 处理文本消息，除了URL
	if messageText != "" {
		richText.Paragraphs = append(richText.Paragraphs, struct {
			Elems []interface{} `json:"elems"`
		}{
			Elems: []interface{}{
				map[string]interface{}{
					"text": map[string]interface{}{
						"text": messageText,
					},
					"type": 1,
				},
			},
		})
	}

	// 为每个URL创建ELEM_TYPE_URL元素
	for _, url := range foundURLs {
		richText.Paragraphs[0].Elems = append(richText.Paragraphs[0].Elems, map[string]interface{}{
			"url": map[string]interface{}{
				"url":  url,
				"desc": "点我跳转",
			},
			"type": 4,
		})
	}

	if len(richText.Paragraphs) == 0 {
		// 初始化一个空段落
		richText.Paragraphs = append(richText.Paragraphs, struct {
			Elems []interface{} "json:\"elems\""
		}{})
	}

	// 处理图片链接
	for _, url := range foundItems["url_image"] {
		richText.Paragraphs[0].Elems = append(richText.Paragraphs[0].Elems, map[string]interface{}{
			"image": map[string]interface{}{
				"third_url":     url,
				"width_percent": 1.0, // 设置图片宽度比例为1.0
			},
			"type": 2,
		})
	}

	// 处理base64图片
	for _, base64Image := range foundItems["base64_image"] {
		fileImageData, err := base64.StdEncoding.DecodeString(base64Image)
		if err != nil {
			mylog.Printf("failed to decode base64 image: %v", err)
			return nil, fmt.Errorf("failed to decode base64 image: %v", err)
		}
		compressedData, err := images.CompressSingleImage(fileImageData)
		if err != nil {
			mylog.Printf("Error compressing image: %v", err)
			return nil, fmt.Errorf("error compressing image: %v", err)
		}
		imageURL, _, _, _, err := images.UploadBase64ImageToServer("", base64.StdEncoding.EncodeToString(compressedData), "", apiv2)
		if err != nil {
			mylog.Printf("failed to upload base64 image: %v", err)
			return nil, fmt.Errorf("failed to upload base64 image: %v", err)
		}
		richText.Paragraphs[0].Elems = append(richText.Paragraphs[0].Elems, map[string]interface{}{
			"image": map[string]interface{}{
				"third_url":     imageURL,
				"width_percent": 1.0, // 设置图片宽度比例为1.0
			},
			"type": 2,
		})
	}

	// 将富文本内容结构转换为JSON字符串
	contentJSON, err := json.Marshal(richText)
	if err != nil {
		mylog.Printf("Error marshalling rich text content: %v", err)
		return nil, fmt.Errorf("error marshalling rich text content: %v", err)
	}

	forum.Content = string(contentJSON)
	forum.Format = 4 // 设置格式为富文本

	return &forum, nil
}
