package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_private_msg", handleSendPrivateMsg)
}

func handleSendPrivateMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	msgType := echo.GetMsgTypeByKey(message.Echo)

	switch msgType {
	case "group_private":
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
			groupReply := generatePrivateMessage(messageID, nil, messageText)

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

			groupReply := generatePrivateMessage(messageID, singleItem, "")

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
	case "guild_private":
		//当收到发私信调用 并且来源是频道
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, nil, nil)
	default:
		log.Printf("Unknown message type: %s", msgType)
	}
}

func generatePrivateMessage(id string, foundItems map[string][]string, messageText string) interface{} {
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

// 处理频道私信 最后2个指针参数可空 代表使用userid倒推
func handleSendGuildChannelPrivateMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage, optionalGuildID *string, optionalChannelID *string) {
	params := message.Params
	messageText, foundItems := parseMessageContent(params)

	var guildID, channelID string
	var err error

	if optionalGuildID != nil && optionalChannelID != nil {
		guildID = *optionalGuildID
		channelID = *optionalChannelID
	} else {
		//默认私信场景 通过仅有的userid来还原频道私信需要的guildid
		guildID, channelID, err = getGuildIDFromMessage(message)
		if err != nil {
			log.Printf("获取 guild_id 和 channel_id 出错: %v", err)
			return
		}
	}

	// 获取 echo 的值
	echostr := message.Echo
	messageID := echo.GetMsgIDByKey(echostr)
	log.Println("私聊信息对应的message_id:", messageID)
	log.Println("私聊信息messageText:", messageText)
	log.Println("foundItems:", foundItems)
	// 如果messageID为空，通过函数获取
	if messageID == "" {
		messageID = GetMessageIDByUseridOrGroupid(client.GetAppID(), message.Params.UserID)
		log.Println("通过GetMessageIDByUserid函数获取的message_id:", messageID)
	}

	timestamp := time.Now().Unix()
	timestampStr := fmt.Sprintf("%d", timestamp)

	// 构造 dm (dms 私信事件)
	dm := &dto.DirectMessage{
		GuildID:    guildID,
		ChannelID:  channelID,
		CreateTime: timestampStr,
	}

	// 优先发送文本信息
	if messageText != "" {
		textMsg, _ := generateReplyMessage(messageID, nil, messageText)
		if _, err := apiv2.PostDirectMessage(context.TODO(), dm, textMsg); err != nil {
			log.Printf("发送文本信息失败: %v", err)
		}
	}

	// 遍历foundItems并发送每种信息
	for key, urls := range foundItems {
		var singleItem = make(map[string][]string)
		singleItem[key] = urls

		reply, isBase64Image := generateReplyMessage(messageID, singleItem, "")

		if isBase64Image {
			// 将base64内容从reply的Content转换回字节
			fileImageData, err := base64.StdEncoding.DecodeString(reply.Content)
			if err != nil {
				log.Printf("Base64 解码失败: %v", err)
				return // 或其他的错误处理方式
			}

			// 清除reply的Content
			reply.Content = ""

			// 使用Multipart方法发送
			if _, err := api.PostDirectMessageMultipart(context.TODO(), dm, reply, fileImageData); err != nil {
				log.Printf("使用multipart发送 %s 信息失败: %v message_id %v", key, err, messageID)
			}
		} else {
			if _, err := api.PostDirectMessage(context.TODO(), dm, reply); err != nil {
				log.Printf("发送 %s 信息失败: %v", key, err)
			}
		}

	}
}

// 这个函数可以通过int类型的虚拟userid反推真实的guild_id和channel_id
func getGuildIDFromMessage(message callapi.ActionMessage) (string, string, error) {
	var userID string

	// 判断UserID的类型，并将其转换为string
	switch v := message.Params.UserID.(type) {
	case int:
		userID = strconv.Itoa(v)
	case float64:
		userID = strconv.FormatInt(int64(v), 10) // 将float64先转为int64，然后再转为string
	case string:
		userID = v
	default:
		return "", "", fmt.Errorf("unexpected type for UserID: %T", v) // 使用%T来打印具体的类型
	}

	// 使用RetrieveRowByID还原真实的UserID
	realUserID, err := idmap.RetrieveRowByID(userID)
	if err != nil {
		return "", "", fmt.Errorf("error retrieving real UserID: %v", err)
	}
	// 使用realUserID作为sectionName从数据库中获取channel_id
	channelID, err := idmap.ReadConfig(realUserID, "channel_id")
	if err != nil {
		return "", "", fmt.Errorf("error reading channel_id: %v", err)
	}

	// 使用channelID作为sectionName从数据库中获取guild_id
	guildID, err := idmap.ReadConfig(channelID, "guild_id")
	if err != nil {
		return "", "", fmt.Errorf("error reading guild_id: %v", err)
	}

	return guildID, channelID, nil
}
