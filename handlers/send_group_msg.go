package handlers

import (
	"context"
	"log"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

var compatibilityMapping *idmap.IniMapping
var err error

func init() {
	callapi.RegisterHandler("send_group_msg", handleSendGroupMsg)
	compatibilityMapping, err = idmap.NewIniMapping()
	if err != nil {
		log.Fatalf("Failed to initialize IniMapping: %v", err)
	}
}

func handleSendGroupMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	msgType := echo.GetMsgTypeByKey(message.Echo)

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
			groupMessage := generateGroupMessage(messageID, nil, messageText)
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
			groupReply.Timestamp = time.Now().Unix() // 设置时间戳
			_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupReply)
			if err != nil {
				log.Printf("发送 %s 信息失败: %v", key, err)
			}
		}
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := compatibilityMapping.ReadConfig(message.Params.ChannelID, "guild_id")
		if err != nil {
			log.Printf("Error reading config: %v", err)
			return
		}
		message.Params.GroupID = value
		handleSendGuildChannelMsg(client, api, apiv2, message)
	default:
		log.Printf("Unknown message type: %s", msgType)
	}
}

func generateGroupMessage(id string, foundItems map[string][]string, messageText string) *dto.MessageToCreate {
	var reply dto.MessageToCreate

	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// todo 完善本地文件上传 发送机制
		reply = dto.MessageToCreate{
			//EventID: id, // Use a placeholder event ID for now
			Image:   imageURLs[0],
			MsgID:   id,
			MsgType: 0, // Assuming type 0 for images
		}
	} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) > 0 {
		// Sending an external image
		reply = dto.MessageToCreate{
			//EventID: id,           // Use a placeholder event ID for now
			Image:   "http://" + imageURLs[0], // Using the same Image field for external URLs, adjust if needed
			MsgID:   id,
			MsgType: 2, // Assuming type 0 for images
		}
	} else if voiceURLs, ok := foundItems["base64_record"]; ok && len(voiceURLs) > 0 {
		//还不支持发语音
		// Sending a voice message
		// reply = dto.MessageToCreate{
		// 	EventID: id,
		// 	Embed: &dto.Embed{
		// 		URL: voiceURLs[0], // Assuming voice is embedded using a URL
		// 	},
		// 	MsgID:   id,
		// 	MsgType: 0, // Adjust type as needed for voice
		// }
	} else {
		// 发文本信息
		reply = dto.MessageToCreate{
			//EventID: id,
			Content: messageText,
			MsgID:   id,
			MsgType: 0, // Default type for text
		}
	}

	return &reply
}
