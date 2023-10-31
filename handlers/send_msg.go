package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_msg", handleSendMsg)
}

func handleSendMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}

	//如果获取不到 就用group_id获取信息类型
	if msgType == "" {
		appID := config.GetAppIDStr()
		groupID := message.Params.GroupID
		fmt.Printf("appID: %s, GroupID: %v\n", appID, groupID)

		msgType = GetMessageTypeByGroupid(appID, groupID)
		fmt.Printf("msgType: %s\n", msgType)
	}

	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}

	switch msgType {
	case "group":
		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params)

		// 使用 echo 获取消息ID
		var messageID string
		if echoStr, ok := message.Echo.(string); ok {
			messageID = echo.GetMsgIDByKey(echoStr)
			log.Println("echo取群组发信息对应的message_id:", messageID)
		}
		log.Println("群组发信息messageText:", messageText)
		log.Println("foundItems:", foundItems)
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), message.Params.GroupID)
			log.Println("通过GetMessageIDByUserid函数获取的message_id:", messageID)
		}

		//通过bolt数据库还原真实的GroupID
		originalGroupID, err := idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
		if err != nil {
			log.Printf("Error retrieving original GroupID: %v", err)
			return
		}
		message.Params.GroupID = originalGroupID

		// 优先发送文本信息
		if messageText != "" {
			groupReply := generateMessage(messageID, nil, messageText)

			// 进行类型断言
			groupMessage, ok := groupReply.(*dto.MessageToCreate)
			if !ok {
				log.Println("Error: Expected MessageToCreate type.")
				return // 或其他错误处理
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			_, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
			if err != nil {
				log.Printf("发送文本群组信息失败: %v", err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			var singleItem = make(map[string][]string)
			singleItem[key] = urls

			groupReply := generateMessage(messageID, singleItem, "")

			// 进行类型断言
			richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
			if !ok {
				log.Printf("Error: Expected RichMediaMessage type for key %s.", key)
				continue // 跳过这个项，继续下一个
			}

			fmt.Printf("richMediaMessage: %+v\n", richMediaMessage)
			_, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), richMediaMessage)
			if err != nil {
				log.Printf("发送 %s 信息失败_send_msg: %v", key, err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		//读取ini 通过ChannelID取回之前储存的guild_id
		// value, err := idmap.ReadConfigv2(message.Params.ChannelID, "guild_id")
		// if err != nil {
		// 	log.Printf("Error reading config: %v", err)
		// 	return
		// }
		// message.Params.GroupID = value
		handleSendGuildChannelMsg(client, api, apiv2, message)
	case "guild_private":
		//send_msg比具体的send_xxx少一层,其包含的字段类型在虚拟化场景已经失去作用
		//根据userid绑定得到的具体真实事件类型,这里也有多种可能性
		//1,私聊(但虚拟成了群),这里用群号取得需要的id
		//2,频道私聊(但虚拟成了私聊)这里传递2个nil,用user_id去推测channel_id和guild_id

		var channelIDPtr *string
		var GuildidPtr *string

		// 先尝试将GroupID断言为字符串
		if channelID, ok := message.Params.GroupID.(string); ok && channelID != "" {
			channelIDPtr = &channelID
			// 读取bolt数据库 通过ChannelID取回之前储存的guild_id
			if value, err := idmap.ReadConfigv2(channelID, "guild_id"); err == nil && value != "" {
				GuildidPtr = &value
			} else {
				log.Printf("Error reading config: %v", err)
			}
		}

		if channelIDPtr == nil || GuildidPtr == nil {
			log.Printf("Value or ChannelID is empty or in error. Value: %v, ChannelID: %v", GuildidPtr, channelIDPtr)
		}

		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, GuildidPtr, channelIDPtr)

	case "group_private":
		//私聊信息
		//还原真实的userid
		UserID, err := idmap.RetrieveRowByIDv2(message.Params.UserID.(string))
		if err != nil {
			log.Printf("Error reading config: %v", err)
			return
		}

		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params)

		// 使用 echo 获取消息ID
		var messageID string
		if echoStr, ok := message.Echo.(string); ok {
			messageID = echo.GetMsgIDByKey(echoStr)
			log.Println("echo取私聊发信息对应的message_id:", messageID)
		}
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), UserID)
			log.Println("通过GetMessageIDByUserid函数获取的message_id:", messageID)
		}
		log.Println("私聊发信息messageText:", messageText)
		log.Println("foundItems:", foundItems)

		// 优先发送文本信息
		if messageText != "" {
			groupReply := generateMessage(messageID, nil, messageText)

			// 进行类型断言
			groupMessage, ok := groupReply.(*dto.MessageToCreate)
			if !ok {
				log.Println("Error: Expected MessageToCreate type.")
				return
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			_, err = apiv2.PostC2CMessage(context.TODO(), UserID, groupMessage)
			if err != nil {
				log.Printf("发送文本私聊信息失败: %v", err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}

		// 遍历 foundItems 并发送每种信息
		for key, urls := range foundItems {
			var singleItem = make(map[string][]string)
			singleItem[key] = urls

			groupReply := generateMessage(messageID, singleItem, "")

			// 进行类型断言
			richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
			if !ok {
				log.Printf("Error: Expected RichMediaMessage type for key %s.", key)
				continue
			}
			_, err = apiv2.PostC2CMessage(context.TODO(), UserID, richMediaMessage)
			if err != nil {
				log.Printf("发送 %s 私聊信息失败: %v", key, err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}
	default:
		log.Printf("1Unknown message type: %s", msgType)
	}
}

func generateMessage(id string, foundItems map[string][]string, messageText string) interface{} {
	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// 从本地路径读取图片
		imageData, err := os.ReadFile(imageURLs[0])
		if err != nil {
			// 读入文件，如果是本地图,应用端和gensokyo需要在一台电脑
			log.Printf("Error reading the image from path %s: %v", imageURLs[0], err)
			return nil
		}

		// base64编码
		base64Encoded := base64.StdEncoding.EncodeToString(imageData)

		// 上传base64编码的图片并获取其URL
		imageURL, err := images.UploadBase64ImageToServer(base64Encoded)
		if err != nil {
			log.Printf("Error uploading base64 encoded image: %v", err)
			return nil
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
			FileType:   1, // 1代表图片
			URL:        "http://" + imageURLs[0],
			Content:    "", // 这个字段文档没有了
			SrvSendMsg: true,
		}
	} else if base64_image, ok := foundItems["base64_image"]; ok && len(base64_image) > 0 {
		// todo 适配base64图片
		//因为QQ群没有 form方式上传,所以在gensokyo内置了图床,需公网,或以lotus方式连接位于公网的gensokyo
		//要正确的开放对应的端口和设置正确的ip地址在config,这对于一般用户是有一些难度的
		if base64Image, ok := foundItems["base64_image"]; ok && len(base64Image) > 0 {
			// 解码base64图片数据
			fileImageData, err := base64.StdEncoding.DecodeString(base64Image[0])
			if err != nil {
				log.Printf("failed to decode base64 image: %v", err)
				return nil
			}
			// 将解码的图片数据转换回base64格式并上传
			imageURL, err := images.UploadBase64ImageToServer(base64.StdEncoding.EncodeToString(fileImageData))
			if err != nil {
				log.Printf("failed to upload base64 image: %v", err)
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
	} else if voiceURLs, ok := foundItems["base64_record"]; ok && len(voiceURLs) > 0 {
		// 目前不支持发语音 todo 适配base64 slik
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

// 通过user_id获取messageID
func GetMessageIDByUseridOrGroupid(appID string, userID interface{}) string {
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
	return echo.GetMsgIDByKey(key)
}
