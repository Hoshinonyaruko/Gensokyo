package handlers

import (
	"context"
	"encoding/base64"
	"os"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"

	"github.com/hoshinonyaruko/gensokyo/echo"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_guild_channel_msg", handleSendGuildChannelMsg)
}

func handleSendGuildChannelMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}
	//如果获取不到 就用group_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
	}
	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}
	//新增 内存获取不到从数据库获取
	if msgType == "" {
		msgType = GetMessageTypeByUseridV2(message.Params.UserID)
	}
	if msgType == "" {
		msgType = GetMessageTypeByGroupidV2(message.Params.GroupID)
	}
	switch msgType {
	//原生guild信息
	case "guild":
		params := message.Params
		messageText, foundItems := parseMessageContent(params)

		channelID := params.ChannelID
		// 使用 echo 获取消息ID
		var messageID string
		if config.GetLazyMessageId() {
			//由于实现了Params的自定义unmarshell 所以可以类型安全的断言为string
			messageID = echo.GetLazyMessagesId(channelID)
			mylog.Printf("GetLazyMessagesId: %v", messageID)
		}
		if messageID == "" {
			if echoStr, ok := message.Echo.(string); ok {
				messageID = echo.GetMsgIDByKey(echoStr)
				mylog.Println("echo取频道发信息对应的message_id:", messageID)
			}
		}
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), channelID)
			mylog.Println("通过GetMessageIDByUseridOrGroupid函数获取的message_id:", messageID)
		}
		//开发环境用
		if config.GetDevMsgID() {
			messageID = "1000"
		}
		mylog.Println("频道发信息messageText:", messageText)
		//mylog.Println("foundItems:", foundItems)
		// 优先发送文本信息
		var err error
		if messageText != "" {
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			textMsg, _ := GenerateReplyMessage(messageID, nil, messageText, msgseq+1)
			if _, err = api.PostMessage(context.TODO(), channelID, textMsg); err != nil {
				mylog.Printf("发送文本信息失败: %v", err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			for _, url := range urls {
				var singleItem = make(map[string][]string)
				singleItem[key] = []string{url} // 创建一个只有一个 URL 的 singleItem
				msgseq := echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				reply, isBase64Image := GenerateReplyMessage(messageID, singleItem, "", msgseq+1)

				if isBase64Image {
					// 将base64内容从reply的Content转换回字节
					fileImageData, err := base64.StdEncoding.DecodeString(reply.Content)
					if err != nil {
						mylog.Printf("Base64 解码失败: %v", err)
						continue // 跳过当前项，继续下一个
					}

					// 清除reply的Content
					reply.Content = ""

					// 使用Multipart方法发送
					if _, err = api.PostMessageMultipart(context.TODO(), channelID, reply, fileImageData); err != nil {
						mylog.Printf("使用multipart发送 %s 信息失败: %v message_id %v", key, err, messageID)
					}
					//发送成功回执
					SendResponse(client, err, &message)
				} else {
					if _, err = api.PostMessage(context.TODO(), channelID, reply); err != nil {
						mylog.Printf("发送 %s 信息失败: %v", key, err)
					}
					//发送成功回执
					SendResponse(client, err, &message)
				}
			}
		}
	//频道私信 此时直接取出
	case "guild_private":
		params := message.Params
		channelID := params.ChannelID
		guildID := params.GuildID
		var RChannelID string
		var err error
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err = idmap.RetrieveRowByIDv2(channelID)
		if err != nil {
			mylog.Printf("error retrieving real UserID: %v", err)
		}
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, &guildID, &RChannelID)
	default:
		mylog.Printf("2Unknown message type: %s", msgType)
	}
}

// 组合发频道信息需要的MessageToCreate 支持base64
func GenerateReplyMessage(id string, foundItems map[string][]string, messageText string, msgseq int) (*dto.MessageToCreate, bool) {
	var reply dto.MessageToCreate
	var isBase64 bool

	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// 从本地图路径读取图片
		imageData, err := os.ReadFile(imageURLs[0])
		if err != nil {
			// 读入文件,如果是本地图,应用端和gensokyo需要在一台电脑
			mylog.Printf("Error reading the image from path %s: %v", imageURLs[0], err)
			// 发文本信息，提示图片文件不存在
			reply = dto.MessageToCreate{
				Content: "错误: 图片文件不存在",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}
			return &reply, false
		}
		// 首先压缩图片
		compressedData, err := images.CompressSingleImage(imageData)
		if err != nil {
			mylog.Printf("Error compressing image: %v", err)
			return &dto.MessageToCreate{
				Content: "错误: 压缩图片失败",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}, false
		}
		//base64编码
		base64Encoded := base64.StdEncoding.EncodeToString(compressedData)

		// 当作base64图来处理
		reply = dto.MessageToCreate{
			Content: base64Encoded,
			MsgID:   id,
			MsgSeq:  msgseq,
			MsgType: 0, // Assuming type 0 for images
		}
		isBase64 = true
	} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) > 0 {
		// 发送网络图
		reply = dto.MessageToCreate{
			//EventID: id,           // Use a placeholder event ID for now
			Image:   "http://" + imageURLs[0], // Using the same Image field for external URLs, adjust if needed
			MsgID:   id,
			MsgSeq:  msgseq,
			MsgType: 0, // Assuming type 0 for images
		}
	} else if voiceURLs, ok := foundItems["base64_record"]; ok && len(voiceURLs) > 0 {
		//频道 还不支持发语音
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
			MsgSeq:  msgseq,
			MsgType: 0, // Default type for text
		}
		isBase64 = true
	} else {
		// 发文本信息
		reply = dto.MessageToCreate{
			//EventID: id,
			Content: messageText,
			MsgID:   id,
			MsgSeq:  msgseq,
			MsgType: 0, // Default type for text
		}
	}

	return &reply, isBase64
}
