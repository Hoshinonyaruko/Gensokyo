package handlers

import (
	"context"
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"

	"github.com/hoshinonyaruko/gensokyo/echo"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_guild_channel_msg", handleSendGuildChannelMsg)
}

func handleSendGuildChannelMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	params := message.Params
	messageText, foundItems := parseMessageContent(params)

	channelID := params.ChannelID
	// 获取 echo 的值
	echostr := message.Echo

	//messageType := echo.GetMsgTypeByKey(echostr)
	messageID := echo.GetMsgIDByKey(echostr)
	log.Println("频道发信息对应的message_id:", messageID)
	log.Println("频道发信息messageText:", messageText)
	log.Println("foundItems:", foundItems)
	// 优先发送文本信息
	if messageText != "" {
		textMsg := generateReplyMessage(messageID, nil, messageText)
		if _, err := api.PostMessage(context.TODO(), channelID, textMsg); err != nil {
			log.Printf("发送文本信息失败: %v", err)
		}
	}

	// 遍历foundItems并发送每种信息
	for key, urls := range foundItems {
		var singleItem = make(map[string][]string)
		singleItem[key] = urls

		reply := generateReplyMessage(messageID, singleItem, "")
		if _, err := api.PostMessage(context.TODO(), channelID, reply); err != nil {
			log.Printf("发送 %s 信息失败: %v", key, err)
		}
	}
}
func generateReplyMessage(id string, foundItems map[string][]string, messageText string) *dto.MessageToCreate {
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
			MsgType: 0, // Assuming type 0 for images
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
