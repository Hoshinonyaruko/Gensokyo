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
	callapi.RegisterHandler("send_private_msg", HandleSendPrivateMsg)
}

func HandleSendPrivateMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	var retmsg string
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}

	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}
	//顺序,私聊优先从UserID推断类型会更准确
	if msgType == "" {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
	}
	//新增 内存获取不到从数据库获取
	if msgType == "" {
		msgType = GetMessageTypeByUseridV2(message.Params.UserID)
	}
	if msgType == "" {
		msgType = GetMessageTypeByGroupidV2(message.Params.GroupID)
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
			HandleSendPrivateMsg(client, api, apiv2, messageCopy)
		}
	}

	switch msgType {
	case "group_private", "group":
		//私聊信息
		var UserID string
		if config.GetIdmapPro() {
			//还原真实的userid
			//mylog.Printf("group_private:%v", message.Params.UserID.(string))
			_, UserID, err = idmap.RetrieveRowByIDv2Pro("690426430", message.Params.UserID.(string))
			if err != nil {
				mylog.Printf("Error reading config: %v", err)
				return "", nil
			}
			mylog.Printf("测试,通过Proid获取的UserID:%v", UserID)
		} else {
			//还原真实的userid
			UserID, err = idmap.RetrieveRowByIDv2(message.Params.UserID.(string))
			if err != nil {
				mylog.Printf("Error reading config: %v", err)
				return "", nil
			}
		}

		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params, message, client, api, apiv2)

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

		var singleItem = make(map[string][]string)
		var imageType, imageUrl string
		imageCount := 0

		// 检查不同类型的图片并计算数量
		if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) == 1 {
			imageType = "local_image"
			imageUrl = imageURLs[0]
			imageCount++
		} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) == 1 {
			imageType = "url_image"
			imageUrl = imageURLs[0]
			imageCount++
		} else if base64Images, ok := foundItems["base64_image"]; ok && len(base64Images) == 1 {
			imageType = "base64_image"
			imageUrl = base64Images[0]
			imageCount++
		}

		if imageCount == 1 && messageText != "" {
			mylog.Printf("发私聊图文混合信息")
			// 创建包含单个图片的 singleItem
			singleItem[imageType] = []string{imageUrl}
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			groupReply := generateGroupMessage(messageID, singleItem, "", msgseq+1)
			// 进行类型断言
			richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
			if !ok {
				mylog.Printf("Error: Expected RichMediaMessage type for key ")
				return "", nil
			}
			// 上传图片并获取FileInfo
			fileInfo, err := uploadMediaPrivate(context.TODO(), UserID, richMediaMessage, apiv2)
			if err != nil {
				mylog.Printf("上传图片失败: %v", err)
				return "", nil // 或其他错误处理
			}
			// 创建包含文本和图像信息的消息
			msgseq = echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			groupMessage := &dto.MessageToCreate{
				Content: messageText, // 添加文本内容
				Media: dto.Media{
					FileInfo: fileInfo, // 添加图像信息
				},
				MsgID:   messageID,
				MsgSeq:  msgseq,
				MsgType: 7, // 假设7是组合消息类型
			}
			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳

			// 发送组合消息
			_, err = apiv2.PostC2CMessage(context.TODO(), UserID, groupMessage)
			if err != nil {
				mylog.Printf("发送组合消息失败: %v", err)
				return "", nil // 或其他错误处理
			}

			// 发送成功回执
			retmsg, _ = SendResponse(client, err, &message)

			delete(foundItems, imageType) // 从foundItems中删除已处理的图片项
			messageText = ""
		}

		// 优先发送文本信息
		if messageText != "" {
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			groupReply := generateGroupMessage(messageID, nil, messageText, msgseq+1)

			// 进行类型断言
			groupMessage, ok := groupReply.(*dto.MessageToCreate)
			if !ok {
				mylog.Println("Error: Expected MessageToCreate type.")
				return "", nil
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			_, err := apiv2.PostC2CMessage(context.TODO(), UserID, groupMessage)
			if err != nil {
				mylog.Printf("发送文本私聊信息失败: %v", err)
			}
			//发送成功回执
			retmsg, _ = SendResponse(client, err, &message)
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			for _, url := range urls {
				var singleItem = make(map[string][]string)
				singleItem[key] = []string{url} // 创建一个只包含一个 URL 的 singleItem
				//mylog.Println("singleItem:", singleItem)
				msgseq := echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				groupReply := generateGroupMessage(messageID, singleItem, "", msgseq+1)
				// 进行类型断言
				richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
				if !ok {
					mylog.Printf("Error: Expected RichMediaMessage type for key %s.", key)
					continue // 跳过这个项，继续下一个
				}
				message_return, err := apiv2.PostC2CMessage(context.TODO(), UserID, richMediaMessage)
				if err != nil {
					mylog.Printf("发送 %s 信息失败_send_private_msg: %v", key, err)
					if config.GetSendError() { //把报错当作文本发出去
						msgseq := echo.GetMappingSeq(messageID)
						echo.AddMappingSeq(messageID, msgseq+1)
						groupReply := generateGroupMessage(messageID, nil, err.Error(), msgseq+1)
						// 进行类型断言
						groupMessage, ok := groupReply.(*dto.MessageToCreate)
						if !ok {
							mylog.Println("Error: Expected MessageToCreate type.")
							return "", nil // 或其他错误处理
						}
						groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
						//重新为err赋值
						_, err = apiv2.PostC2CMessage(context.TODO(), UserID, groupMessage)
						if err != nil {
							mylog.Printf("发送 %s 私聊信息失败: %v", key, err)
						}
					}
				}
				if message_return != nil && message_return.MediaResponse != nil && message_return.MediaResponse.FileInfo != "" {
					msgseq := echo.GetMappingSeq(messageID)
					echo.AddMappingSeq(messageID, msgseq+1)
					media := dto.Media{
						FileInfo: message_return.MediaResponse.FileInfo,
					}
					groupMessage := &dto.MessageToCreate{
						Content: " ",
						MsgID:   messageID,
						MsgSeq:  msgseq,
						MsgType: 7, // 默认文本类型
						Media:   media,
					}
					groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
					//重新为err赋值
					_, err = apiv2.PostC2CMessage(context.TODO(), UserID, groupMessage)
					if err != nil {
						mylog.Printf("发送 %s 私聊信息失败: %v", key, err)
					}
				}
				//发送成功回执
				retmsg, _ = SendResponse(client, err, &message)
			}
		}
	case "guild_private", "guild":
		//当收到发私信调用 并且来源是频道
		retmsg, _ = HandleSendGuildChannelPrivateMsg(client, api, apiv2, message, nil, nil)
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
		delay := config.GetSendDelay()
		time.Sleep(time.Duration(delay) * time.Millisecond)
		HandleSendPrivateMsg(client, api, apiv2, messageCopy)
	}
	return retmsg, nil
}

// 处理频道私信 最后2个指针参数可空 代表使用userid倒推
func HandleSendGuildChannelPrivateMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage, optionalGuildID *string, optionalChannelID *string) (string, error) {
	params := message.Params
	messageText, foundItems := parseMessageContent(params, message, client, api, apiv2)

	var guildID, channelID string
	var err error
	var UserID string
	var GroupID string
	var retmsg string
	if message.Params.GroupID != nil {
		if gid, ok := message.Params.GroupID.(string); ok {
			GroupID = gid // GroupID 是 string 类型
		} else {
			mylog.Printf(" GroupID 不是 string,304")
		}
	} else {
		mylog.Printf("GroupID 为 nil,信息发送正常可忽略")
	}
	RawUserID := message.Params.UserID.(string)

	if optionalGuildID != nil && optionalChannelID != nil {
		guildID = *optionalGuildID
		channelID = *optionalChannelID
	}

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
	//获取guild和channelid和message id流程(来个大佬简化下)
	if RawUserID != "" {
		if guildID == "" && channelID == "" {
			//频道私信 转 私信 通过userid(author_id)来还原频道私信需要的guildid channelID
			guildID, channelID, err = getGuildIDFromMessage(message)
			if err != nil {
				mylog.Printf("获取 guild_id 和 channel_id 出错,进行重试: %v", err)
				guildID, channelID, err = getGuildIDFromMessagev2(message)
				if err != nil {
					mylog.Printf("获取 guild_id 和 channel_id 出错,重试失败: %v", err)
					return "", nil
				}
			}
			//频道私信 转 私信
			if GroupID != "" && config.GetIdmapPro() {
				_, UserID, err = idmap.RetrieveRowByIDv2Pro(GroupID, RawUserID)
				if err != nil {
					mylog.Printf("Error reading config: %v", err)
					return "", nil
				}
				mylog.Printf("测试,通过Proid获取的UserID:%v", UserID)
			} else {
				UserID, err = idmap.RetrieveRowByIDv2(RawUserID)
				if err != nil {
					mylog.Printf("Error reading config: %v", err)
					return "", nil
				}
			}
			// 如果messageID为空，通过函数获取
			if messageID == "" {
				messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), UserID)
				mylog.Println("通过GetMessageIDByUserid函数获取的message_id:", messageID)
			}
		} else {
			//频道私信 转 私信
			if GroupID != "" && config.GetIdmapPro() {
				_, UserID, err = idmap.RetrieveRowByIDv2Pro(GroupID, RawUserID)
				if err != nil {
					mylog.Printf("Error reading config: %v", err)
					return "", nil
				}
				mylog.Printf("测试,通过Proid获取的UserID:%v", UserID)
			} else {
				UserID, err = idmap.RetrieveRowByIDv2(RawUserID)
				if err != nil {
					mylog.Printf("Error reading config: %v", err)
					return "", nil
				}
			}
			// 如果messageID为空，通过函数获取
			if messageID == "" {
				messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), UserID)
				mylog.Println("通过GetMessageIDByUserid函数获取的message_id:", messageID)
			}
		}
	} else {
		if guildID == "" && channelID == "" {
			//频道私信 转 群聊 通过groupid(author_id)来还原频道私信需要的guildid channelID
			guildID, err = idmap.ReadConfigv2(GroupID, "guild_id")
			if err != nil {
				mylog.Printf("根据GroupID获取guild_id失败: %v", err)
				return "", nil
			}
			channelID, err = idmap.RetrieveRowByIDv2(GroupID)
			if err != nil {
				mylog.Printf("根据GroupID获取channelID失败: %v", err)
				return "", nil
			}
			//频道私信 转 群聊 获取id
			var originalGroupID string
			if config.GetIdmapPro() {
				_, originalGroupID, err = idmap.RetrieveRowByIDv2Pro(channelID, GroupID)
				if err != nil {
					mylog.Printf("Error retrieving original GroupID: %v", err)
					return "", nil
				}
				mylog.Printf("测试,通过Proid获取的originalGroupID:%v", originalGroupID)
			} else {
				originalGroupID, err = idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
				if err != nil {
					mylog.Printf("Error retrieving original GroupID: %v", err)
					return "", nil
				}
			}
			mylog.Println("群组(私信虚拟成的)发信息messageText:", messageText)
			//mylog.Println("foundItems:", foundItems)
			// 如果messageID为空，通过函数获取
			if messageID == "" {
				messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), originalGroupID)
				mylog.Println("通过GetMessageIDByUseridOrGroupid函数获取的message_id:", originalGroupID, messageID)
			}
		} else {
			//频道私信 转 群聊 获取id
			var originalGroupID string
			if config.GetIdmapPro() {
				_, originalGroupID, err = idmap.RetrieveRowByIDv2Pro(GroupID, RawUserID)
				if err != nil {
					mylog.Printf("Error retrieving original GroupID2: %v", err)
				}
				mylog.Printf("测试,通过Proid获取的originalGroupID:%v", originalGroupID)
			}
			//降级重试
			if originalGroupID == "" {
				originalGroupID, err = idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
				if err != nil {
					mylog.Printf("Error retrieving original GroupID: %v", err)
				}
			}
			if messageID == "" {
				messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), originalGroupID)
				mylog.Println("通过GetMessageIDByUseridOrGroupid函数获取的message_id:", originalGroupID, messageID)
			}
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
		retmsg, _ = SendResponse(client, err, &message)
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
				retmsg, _ = SendResponse(client, err, &message)
			} else {
				// 处理非 Base64 图片的逻辑
				if _, err = api.PostDirectMessage(context.TODO(), dm, reply); err != nil {
					mylog.Printf("发送 %s 信息失败: %v", key, err)
				}
				retmsg, _ = SendResponse(client, err, &message)
			}
		}
	}
	return retmsg, nil
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
	var realUserID string
	var err error
	// 使用RetrieveRowByIDv2还原真实的UserID
	realUserID, err = idmap.RetrieveRowByIDv2(userID)
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

// 这个函数可以通过int类型的虚拟groupid反推真实的guild_id和channel_id
func getGuildIDFromMessagev2(message callapi.ActionMessage) (string, string, error) {
	var GroupID string
	//groupID此时是转换后的channelid

	// 判断UserID的类型，并将其转换为string
	switch v := message.Params.GroupID.(type) {
	case int:
		GroupID = strconv.Itoa(v)
	case float64:
		GroupID = strconv.FormatInt(int64(v), 10) // 将float64先转为int64，然后再转为string
	case string:
		GroupID = v
	default:
		return "", "", fmt.Errorf("unexpected type for UserID: %T", v) // 使用%T来打印具体的类型
	}

	var err error
	//使用channelID作为sectionName从数据库中获取guild_id
	guildID, err := idmap.ReadConfigv2(GroupID, "guild_id")
	if err != nil {
		return "", "", fmt.Errorf("error reading guild_id: %v", err)
	}

	return guildID, GroupID, nil
}

// uploadMedia 上传媒体并返回FileInfo
func uploadMediaPrivate(ctx context.Context, UserID string, richMediaMessage *dto.RichMediaMessage, apiv2 openapi.OpenAPI) (string, error) {
	// 调用API来上传媒体
	messageReturn, err := apiv2.PostC2CMessage(ctx, UserID, richMediaMessage)
	if err != nil {
		return "", err
	}
	// 返回上传后的FileInfo
	return messageReturn.MediaResponse.FileInfo, nil
}
