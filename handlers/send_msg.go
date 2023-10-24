package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_msg", handleSendMsg)
}

func handleSendMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	msgType := echo.GetMsgTypeByKey(message.Echo)

	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(client.GetAppID(), message.Params.UserID)
	}

	//如果获取不到 就用group_id获取信息类型
	if msgType == "" {
		appID := client.GetAppID()
		groupID := message.Params.GroupID
		fmt.Printf("appID: %s, GroupID: %v\n", appID, groupID)

		msgType = GetMessageTypeByGroupid(appID, groupID)
		fmt.Printf("msgType: %s\n", msgType)
	}

	switch msgType {
	case "group":
		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params)

		// 获取 echo 的值
		echostr := message.Echo

		// 使用 echo 获取消息ID
		messageID := echo.GetMsgIDByKey(echostr)
		log.Println("群组发信息对应的message_id:", messageID)
		log.Println("群组发信息messageText:", messageText)
		log.Println("foundItems:", foundItems)

		//通过bolt数据库还原真实的GroupID
		originalGroupID, err := idmap.RetrieveRowByID(message.Params.GroupID.(string))
		if err != nil {
			log.Printf("Error retrieving original GroupID: %v", err)
			return
		}
		message.Params.GroupID = originalGroupID

		// 优先发送文本信息
		if messageText != "" {
			groupReply := generateGroupMessage(messageID, nil, messageText)

			// 进行类型断言
			groupMessage, ok := groupReply.(*dto.MessageToCreate)
			if !ok {
				log.Println("Error: Expected MessageToCreate type.")
				return // 或其他错误处理
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
			if err != nil {
				log.Printf("发送文本群组信息失败: %v", err)
			}
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			var singleItem = make(map[string][]string)
			singleItem[key] = urls

			groupReply := generateGroupMessage(messageID, singleItem, "")

			// 进行类型断言
			richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
			if !ok {
				log.Printf("Error: Expected RichMediaMessage type for key %s.", key)
				continue // 跳过这个项，继续下一个
			}

			//richMediaMessage.Timestamp = time.Now().Unix() // 设置时间戳
			_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), richMediaMessage)
			if err != nil {
				log.Printf("发送 %s 信息失败: %v", key, err)
			}
		}
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		//读取ini 通过ChannelID取回之前储存的guild_id
		// value, err := idmap.ReadConfig(message.Params.ChannelID, "guild_id")
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
			if value, err := idmap.ReadConfig(channelID, "guild_id"); err == nil && value != "" {
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
		//用userid还原出openid 这是虚拟成群的群聊私聊信息
		//todo
		message.Params.ChannelID = message.Params.GroupID.(string)
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfig(message.Params.ChannelID, "guild_id")
		if err != nil {
			log.Printf("Error reading config: %v", err)
			return
		}
		message.Params.GroupID = value
		handleSendGuildChannelMsg(client, api, apiv2, message)
	default:
		log.Printf("1Unknown message type: %s", msgType)
	}
}

func generateMessage(id string, foundItems map[string][]string, messageText string) interface{} {
	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// 本地发图逻辑 todo 适配base64图片
		return &dto.RichMediaMessage{
			EventID:    id,
			FileType:   1, // 1代表图片
			URL:        imageURLs[0],
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
		// 目前不支持发语音 todo 适配base64 slik
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
