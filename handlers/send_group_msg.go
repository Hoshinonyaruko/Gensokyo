package handlers

import (
	"context"
	"encoding/base64"
	"os"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_group_msg", handleSendGroupMsg)
}

func handleSendGroupMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}

	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}

	//如果获取不到 就用group_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
	}

	switch msgType {
	case "group":
		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params)

		// 使用 echo 获取消息ID
		var messageID string
		if config.GetLazyMessageId() {
			//由于实现了Params的自定义unmarshell 所以可以类型安全的断言为string
			messageID = echo.GetLazyMessagesId(message.Params.GroupID.(string))
			mylog.Printf("GetLazyMessagesId: %v", messageID)
		}
		if messageID == "" {
			if echoStr, ok := message.Echo.(string); ok {
				messageID = echo.GetMsgIDByKey(echoStr)
				mylog.Println("echo取群组发信息对应的message_id:", messageID)
			}
		}
		//通过bolt数据库还原真实的GroupID
		originalGroupID, err := idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
		if err != nil {
			mylog.Printf("Error retrieving original GroupID: %v", err)
			return
		}
		message.Params.GroupID = originalGroupID
		mylog.Println("群组发信息messageText:", messageText)
		//mylog.Println("foundItems:", foundItems)
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), message.Params.GroupID)
			mylog.Println("通过GetMessageIDByUseridOrGroupid函数获取的message_id:", message.Params.GroupID, messageID)
		}
		//开发环境用
		if config.GetDevMsgID() {
			messageID = "1000"
		}
		// 优先发送文本信息
		if messageText != "" {
			groupReply := generateGroupMessage(messageID, nil, messageText)

			// 进行类型断言
			groupMessage, ok := groupReply.(*dto.MessageToCreate)
			if !ok {
				mylog.Println("Error: Expected MessageToCreate type.")
				return // 或其他错误处理
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			//重新为err赋值
			_, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
			if err != nil {
				mylog.Printf("发送文本群组信息失败: %v", err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			var singleItem = make(map[string][]string)
			singleItem[key] = urls

			groupReply := generateGroupMessage(messageID, singleItem, "")

			// 进行类型断言
			richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
			if !ok {
				mylog.Printf("Error: Expected RichMediaMessage type for key %s.", key)
				continue // 跳过这个项，继续下一个
			}
			mylog.Printf("richMediaMessage: %+v\n", richMediaMessage)
			_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), richMediaMessage)
			if err != nil {
				mylog.Printf("发送 %s 信息失败_send_group_msg: %v", key, err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err := idmap.RetrieveRowByIDv2(message.Params.ChannelID)
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
		message.Params.ChannelID = RChannelID
		//这一句是group_private的逻辑,发频道信息用的是channelid
		//message.Params.GroupID = value
		handleSendGuildChannelMsg(client, api, apiv2, message)
	case "guild_private":
		//用group_id还原出channelid 这是虚拟成群的私聊信息
		message.Params.ChannelID = message.Params.GroupID.(string)
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err := idmap.RetrieveRowByIDv2(message.Params.ChannelID)
		if err != nil {
			mylog.Printf("error retrieving real ChannelID: %v", err)
		}
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(RChannelID, "guild_id")
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return
		}
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, &value, &message.Params.ChannelID)
	case "group_private":
		//用userid还原出openid 这是虚拟成群的群聊私聊信息
		message.Params.UserID = message.Params.GroupID.(string)
		handleSendPrivateMsg(client, api, apiv2, message)
	default:
		mylog.Printf("Unknown message type: %s", msgType)
	}
}

// 不支持base64
func generateGroupMessage(id string, foundItems map[string][]string, messageText string) interface{} {
	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// 从本地路径读取图片
		imageData, err := os.ReadFile(imageURLs[0])
		if err != nil {
			// 读入文件失败
			mylog.Printf("Error reading the image from path %s: %v", imageURLs[0], err)
			// 返回文本信息，提示图片文件不存在
			return &dto.MessageToCreate{
				Content: "错误: 图片文件不存在",
				MsgID:   id,
				MsgType: 0, // 默认文本类型
			}
		}
		// 首先压缩图片 默认不压缩
		compressedData, err := images.CompressSingleImage(imageData)
		if err != nil {
			mylog.Printf("Error compressing image: %v", err)
			return &dto.MessageToCreate{
				Content: "错误: 压缩图片失败",
				MsgID:   id,
				MsgType: 0, // 默认文本类型
			}
		}

		// base64编码
		base64Encoded := base64.StdEncoding.EncodeToString(compressedData)

		// 上传base64编码的图片并获取其URL
		imageURL, err := images.UploadBase64ImageToServer(base64Encoded)
		if err != nil {
			mylog.Printf("Error uploading base64 encoded image: %v", err)
			// 如果上传失败，也返回文本信息，提示上传失败
			return &dto.MessageToCreate{
				Content: "错误: 上传图片失败",
				MsgID:   id,
				MsgType: 0, // 默认文本类型
			}
		}

		// 创建RichMediaMessage并返回，当作URL图片处理
		return &dto.RichMediaMessage{
			EventID:    id,
			FileType:   1, // 1代表图片
			URL:        imageURL,
			Content:    "", // 这个字段文档没有了
			SrvSendMsg: true,
		}
	} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) > 0 {
		// 发链接图片
		return &dto.RichMediaMessage{
			EventID:    id,
			FileType:   1,                        // 1代表图片
			URL:        "http://" + imageURLs[0], //url在base64时候被截断了,在这里补全
			Content:    "",                       // 这个字段文档没有了
			SrvSendMsg: true,
		}
	} else if voiceURLs, ok := foundItems["base64_record"]; ok && len(voiceURLs) > 0 {
		// 目前不支持发语音 todo 适配base64 slik
	} else if base64_image, ok := foundItems["base64_image"]; ok && len(base64_image) > 0 {
		// todo 适配base64图片
		//因为QQ群没有 form方式上传,所以在gensokyo内置了图床,需公网,或以lotus方式连接位于公网的gensokyo
		//要正确的开放对应的端口和设置正确的ip地址在config,这对于一般用户是有一些难度的
		if base64Image, ok := foundItems["base64_image"]; ok && len(base64Image) > 0 {
			// 解码base64图片数据
			fileImageData, err := base64.StdEncoding.DecodeString(base64Image[0])
			if err != nil {
				mylog.Printf("failed to decode base64 image: %v", err)
				return nil
			}
			// 首先压缩图片 默认不压缩
			compressedData, err := images.CompressSingleImage(fileImageData)
			if err != nil {
				mylog.Printf("Error compressing image: %v", err)
				return &dto.MessageToCreate{
					Content: "错误: 压缩图片失败",
					MsgID:   id,
					MsgType: 0, // 默认文本类型
				}
			}
			// 将解码的图片数据转换回base64格式并上传
			imageURL, err := images.UploadBase64ImageToServer(base64.StdEncoding.EncodeToString(compressedData))
			if err != nil {
				mylog.Printf("failed to upload base64 image: %v", err)
				return nil
			}
			// 创建RichMediaMessage并返回
			return &dto.RichMediaMessage{
				EventID:    id,
				FileType:   1, // 1代表图片
				URL:        imageURL,
				Content:    "", // 这个字段文档没有了
				SrvSendMsg: true,
			}
		}
	} else {
		// 返回文本信息
		return &dto.MessageToCreate{
			Content: messageText,
			MsgID:   id,
			MsgType: 0, // 默认文本类型
		}
	}
	return nil
}

// 通过user_id获取类型
func GetMessageTypeByUserid(appID string, userID interface{}) string {
	// 从appID和userID生成key
	var userIDStr string
	switch u := userID.(type) {
	case int:
		userIDStr = strconv.Itoa(u)
	case int64:
		userIDStr = strconv.FormatInt(u, 10)
	case float64:
		userIDStr = strconv.FormatFloat(u, 'f', 0, 64)
	case string:
		userIDStr = u
	default:
		// 可能需要处理其他类型或报错
		return ""
	}

	key := appID + "_" + userIDStr
	return echo.GetMsgTypeByKey(key)
}

// 通过group_id获取类型
func GetMessageTypeByGroupid(appID string, GroupID interface{}) string {
	// 从appID和userID生成key
	var GroupIDStr string
	switch u := GroupID.(type) {
	case int:
		GroupIDStr = strconv.Itoa(u)
	case int64:
		GroupIDStr = strconv.FormatInt(u, 10)
	case string:
		GroupIDStr = u
	default:
		// 可能需要处理其他类型或报错
		return ""
	}

	key := appID + "_" + GroupIDStr
	return echo.GetMsgTypeByKey(key)
}
