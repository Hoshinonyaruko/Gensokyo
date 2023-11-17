package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_private_msg", handleSendPrivateMsg)
}

func handleSendPrivateMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
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
		mylog.Printf("appID: %s, GroupID: %v\n", appID, groupID)

		msgType = GetMessageTypeByGroupid(appID, groupID)
		mylog.Printf("msgType: %s\n", msgType)
	}

	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}

	var idInt64 int64
	var err error

	if message.Params.UserID != "" {
		idInt64, err = ConvertToInt64(message.Params.UserID)
	} else if message.Params.GroupID != "" {
		idInt64, err = ConvertToInt64(message.Params.GroupID)
	}

	//设置递归 对直接向gsk发送action时有效果
	if msgType == "" {
		messageCopy := message
		if err != nil {
			mylog.Printf("错误：无法转换 ID %v\n", err)
		} else {
			// 递归3次
			echo.AddMapping(idInt64, 4)
			// 递归调用handleSendPrivateMsg，使用设置的消息类型
			echo.AddMsgType(config.GetAppIDStr(), idInt64, "group_private")
			handleSendPrivateMsg(client, api, apiv2, messageCopy)
		}
	}

	switch msgType {
	case "group_private":
		//私聊信息
		//还原真实的userid
		UserID, err := idmap.RetrieveRowByIDv2(message.Params.UserID.(string))
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return
		}

		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params)

		// 使用 echo 获取消息ID
		var messageID string
		if config.GetLazyMessageId() {
			//由于实现了Params的自定义unmarshell 所以可以类型安全的断言为string
			messageID = echo.GetLazyMessagesId(UserID)
			mylog.Printf("GetLazyMessagesId: %v", messageID)
		}
		if messageID == "" {
			if echoStr, ok := message.Echo.(string); ok {
				messageID = echo.GetMsgIDByKey(echoStr)
				mylog.Println("echo取私聊发信息对应的message_id:", messageID)
			}
		}
		// 如果messageID仍然为空，尝试使用config.GetAppID和UserID的组合来获取messageID
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), UserID)
			mylog.Println("通过GetMessageIDByUserid函数获取的message_id:", messageID)
		}
		//开发环境用
		if config.GetDevMsgID() {
			messageID = "1000"
		}
		mylog.Println("私聊发信息messageText:", messageText)
		//mylog.Println("foundItems:", foundItems)

		// 优先发送文本信息
		if messageText != "" {
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			groupReply := generateGroupMessage(messageID, nil, messageText, msgseq+1)

			// 进行类型断言
			groupMessage, ok := groupReply.(*dto.MessageToCreate)
			if !ok {
				mylog.Println("Error: Expected MessageToCreate type.")
				return
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			_, err := apiv2.PostC2CMessage(context.TODO(), UserID, groupMessage)
			if err != nil {
				mylog.Printf("发送文本私聊信息失败: %v", err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			for _, url := range urls {
				var singleItem = make(map[string][]string)
				singleItem[key] = []string{url} // 创建只包含一个 URL 的 singleItem

				// 生成消息
				msgseq := echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				groupReply := generateGroupMessage(messageID, singleItem, "", msgseq+1)

				// 进行类型断言
				richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
				if !ok {
					mylog.Printf("Error: Expected RichMediaMessage type for key %s.", key)
					continue // 如果断言失败，跳过当前 url
				}

				// 发送消息
				_, err := apiv2.PostC2CMessage(context.TODO(), UserID, richMediaMessage)
				if err != nil {
					mylog.Printf("发送 %s 私聊信息失败: %v", key, err)
				}

				// 发送成功回执
				SendResponse(client, err, &message)
			}
		}
	case "guild_private":
		//当收到发私信调用 并且来源是频道
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, nil, nil)
	default:
		mylog.Printf("Unknown message type: %s", msgType)
	}
	//重置递归类型
	if echo.GetMapping(idInt64) <= 0 {
		echo.AddMsgType(config.GetAppIDStr(), idInt64, "")
	}
	echo.AddMapping(idInt64, echo.GetMapping(idInt64)-1)

	//递归3次枚举类型
	if echo.GetMapping(idInt64) > 0 {
		tryMessageTypes := []string{"group", "guild", "guild_private"}
		messageCopy := message // 创建message的副本
		echo.AddMsgType(config.GetAppIDStr(), idInt64, tryMessageTypes[echo.GetMapping(idInt64)-1])
		time.Sleep(300 * time.Millisecond)
		handleSendGroupMsg(client, api, apiv2, messageCopy)
	}
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
			mylog.Printf("获取 guild_id 和 channel_id 出错: %v", err)
			return
		}
	}
	RawUserID := message.Params.UserID.(string)
	// 使用 echo 获取消息ID
	var messageID string
	if config.GetLazyMessageId() {
		//由于实现了Params的自定义unmarshell 所以可以类型安全的断言为string
		messageID = echo.GetLazyMessagesId(RawUserID)
		mylog.Printf("GetLazyMessagesId: %v", messageID)
	}
	if messageID == "" {
		if echoStr, ok := message.Echo.(string); ok {
			messageID = echo.GetMsgIDByKey(echoStr)
			mylog.Println("echo取私聊发信息对应的message_id:", messageID)
		}
	}
	mylog.Println("私聊信息messageText:", messageText)
	if RawUserID != "" { //还原真实的userid todo 太绕了 要精简下逻辑
		UserID, err := idmap.RetrieveRowByIDv2(RawUserID)
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return
		}
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), UserID)
			mylog.Println("通过GetMessageIDByUserid函数获取的message_id:", messageID)
		}
	} else {
		//通过bolt数据库还原真实的GroupID
		originalGroupID, err := idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
		if err != nil {
			mylog.Printf("Error retrieving original GroupID: %v", err)
			return
		}
		message.Params.GroupID = originalGroupID
		mylog.Println("群组(私信虚拟成的)发信息messageText:", messageText)
		//mylog.Println("foundItems:", foundItems)
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), message.Params.GroupID)
			mylog.Println("通过GetMessageIDByUseridOrGroupid函数获取的message_id:", message.Params.GroupID, messageID)
		}
	}
	//开发环境用
	if config.GetDevMsgID() {
		messageID = "1000"
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
		msgseq := echo.GetMappingSeq(messageID)
		echo.AddMappingSeq(messageID, msgseq+1)
		textMsg, _ := GenerateReplyMessage(messageID, nil, messageText, msgseq+1)
		if _, err = apiv2.PostDirectMessage(context.TODO(), dm, textMsg); err != nil {
			mylog.Printf("发送文本信息失败: %v", err)
		}
		//发送成功回执
		SendResponse(client, err, &message)
	}

	// 遍历foundItems并发送每种信息
	for key, urls := range foundItems {
		for _, url := range urls {
			var singleItem = make(map[string][]string)
			singleItem[key] = []string{url} // 创建一个只包含单个 URL 的 singleItem
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			reply, isBase64Image := GenerateReplyMessage(messageID, singleItem, "", msgseq+1)

			if isBase64Image {
				// 处理 Base64 图片的逻辑
				fileImageData, err := base64.StdEncoding.DecodeString(reply.Content)
				if err != nil {
					mylog.Printf("Base64 解码失败: %v", err)
					continue // 跳过当前项，继续处理下一个
				}

				reply.Content = ""

				if _, err = api.PostDirectMessageMultipart(context.TODO(), dm, reply, fileImageData); err != nil {
					mylog.Printf("使用multipart发送 %s 信息失败: %v message_id %v", key, err, messageID)
				}
				SendResponse(client, err, &message)
			} else {
				// 处理非 Base64 图片的逻辑
				if _, err = api.PostDirectMessage(context.TODO(), dm, reply); err != nil {
					mylog.Printf("发送 %s 信息失败: %v", key, err)
				}
				SendResponse(client, err, &message)
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

	// 使用RetrieveRowByIDv2还原真实的UserID
	realUserID, err := idmap.RetrieveRowByIDv2(userID)
	if err != nil {
		return "", "", fmt.Errorf("error retrieving real UserID: %v", err)
	}
	// 使用realUserID作为sectionName从数据库中获取channel_id
	channelID, err := idmap.ReadConfigv2(realUserID, "channel_id")
	if err != nil {
		return "", "", fmt.Errorf("error reading channel_id: %v", err)
	}
	//使用channelID作为sectionName从数据库中获取guild_id
	guildID, err := idmap.ReadConfigv2(channelID, "guild_id")
	if err != nil {
		return "", "", fmt.Errorf("error reading guild_id: %v", err)
	}

	return guildID, channelID, nil
}
