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
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/server"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_group_msg", handleSendGroupMsg)
}

func handleSendGroupMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	msgType := echo.GetMsgTypeByKey(string(message.Echo))

	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(client.GetAppID(), message.Params.UserID)
	}

	//如果获取不到 就用group_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByGroupid(client.GetAppID(), message.Params.GroupID)
	}

	switch msgType {
	case "group":
		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params)

		// 获取 echo 的值
		echostr := string(message.Echo)

		// 使用 echo 获取消息ID
		messageID := echo.GetMsgIDByKey(echostr)
		log.Println("群组发信息对应的message_id:", messageID)
		log.Println("群组发信息messageText:", messageText)
		log.Println("foundItems:", foundItems)
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(client.GetAppID(), message.Params.GroupID)
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
			_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
			if err != nil {
				log.Printf("发送文本群组信息失败: %v", err)
			}
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
			_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), richMediaMessage)
			if err != nil {
				log.Printf("发送 %s 信息失败_send_group_msg: %v", key, err)
			}
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
		//这一句是group_private的逻辑,发频道信息用的是channelid,只有频道private需要guildid才需要这些逻辑
		//message.Params.GroupID = value
		handleSendGuildChannelMsg(client, api, apiv2, message)
	case "guild_private":
		//用group_id还原出channelid 这是虚拟成群的私聊信息
		message.Params.ChannelID = message.Params.GroupID.(string)
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(message.Params.ChannelID, "guild_id")
		if err != nil {
			log.Printf("Error reading config: %v", err)
			return
		}
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, &value, &message.Params.ChannelID)
	case "group_private":
		//用userid还原出openid 这是虚拟成群的群聊私聊信息
		//todo
		message.Params.ChannelID = message.Params.GroupID.(string)
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(message.Params.ChannelID, "guild_id")
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

func generateGroupMessage(id string, foundItems map[string][]string, messageText string) interface{} {
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
		imageURL, err := server.UploadBase64ImageToServer(base64Encoded)
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
				log.Printf("failed to decode base64 image: %v", err)
				return nil
			}
			// 将解码的图片数据转换回base64格式并上传
			imageURL, err := server.UploadBase64ImageToServer(base64.StdEncoding.EncodeToString(fileImageData))
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
