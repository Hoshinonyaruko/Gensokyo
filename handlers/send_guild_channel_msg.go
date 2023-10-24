package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/hoshinonyaruko/gensokyo/callapi"

	"github.com/hoshinonyaruko/gensokyo/echo"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_guild_channel_msg", handleSendGuildChannelMsg)
}

func handleSendGuildChannelMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
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
	//原生guild信息
	case "guild":
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
			textMsg, _ := generateReplyMessage(messageID, nil, messageText)
			if _, err := api.PostMessage(context.TODO(), channelID, textMsg); err != nil {
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
				if _, err := api.PostMessageMultipart(context.TODO(), channelID, reply, fileImageData); err != nil {
					log.Printf("使用multipart发送 %s 信息失败: %v message_id %v", key, err, messageID)
				}
			} else {
				if _, err := api.PostMessage(context.TODO(), channelID, reply); err != nil {
					log.Printf("发送 %s 信息失败: %v", key, err)
				}
			}

		}
	//频道私信 此时直接取出
	case "guild_private":
		params := message.Params
		channelID := params.ChannelID
		guildID := params.GuildID
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, &guildID, &channelID)
	default:
		log.Printf("2Unknown message type: %s", msgType)
	}
}

// 组合发频道信息需要的MessageToCreate
func generateReplyMessage(id string, foundItems map[string][]string, messageText string) (*dto.MessageToCreate, bool) {
	var reply dto.MessageToCreate
	var isBase64 bool

	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// 从本地图路径读取图片
		imageData, err := os.ReadFile(imageURLs[0])
		if err != nil {
			// 读入文件,如果是本地图,应用端和gensokyo需要在一台电脑
			log.Printf("Error reading the image from path %s: %v", imageURLs[0], err)
			return nil, false
		}

		//base64编码
		base64Encoded := base64.StdEncoding.EncodeToString(imageData)

		// 当作base64图来处理
		reply = dto.MessageToCreate{
			Content: base64Encoded,
			MsgID:   id,
			MsgType: 0, // Assuming type 0 for images
		}
		isBase64 = true
	} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) > 0 {
		// 发送网络图
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
	} else if base64_image, ok := foundItems["base64_image"]; ok && len(base64_image) > 0 {
		// base64图片
		reply = dto.MessageToCreate{
			Content: base64_image[0], // 直接使用base64_image[0]作为Content
			MsgID:   id,
			MsgType: 0, // Default type for text
		}
		isBase64 = true
	} else {
		// 发文本信息
		reply = dto.MessageToCreate{
			//EventID: id,
			Content: messageText,
			MsgID:   id,
			MsgType: 0, // Default type for text
		}
	}

	return &reply, isBase64
}
