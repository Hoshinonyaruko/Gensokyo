package handlers

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/keyboard"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_group_msg_raw", HandleSendGroupMsgRaw)
}

func HandleSendGroupMsgRaw(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}
	// 检查GroupID是否为0
	checkZeroGroupID := func(id interface{}) bool {
		switch v := id.(type) {
		case int:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != "0" // 检查字符串形式的0
		default:
			return true // 如果不是int、int64或string，假定它不为0
		}
	}

	// 检查UserID是否为0
	checkZeroUserID := func(id interface{}) bool {
		switch v := id.(type) {
		case int:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != "0" // 同样检查字符串形式的0
		default:
			return true // 如果不是int、int64或string，假定它不为0
		}
	}

	if msgType == "" && message.Params.GroupID != nil && checkZeroGroupID(message.Params.GroupID) {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
	}
	if msgType == "" && message.Params.UserID != nil && checkZeroUserID(message.Params.UserID) {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}
	if msgType == "" && message.Params.GroupID != nil && checkZeroGroupID(message.Params.GroupID) {
		msgType = GetMessageTypeByGroupidV2(message.Params.GroupID)
	}
	if msgType == "" && message.Params.UserID != nil && checkZeroUserID(message.Params.UserID) {
		msgType = GetMessageTypeByUseridV2(message.Params.UserID)
	}
	// New checks for UserID and GroupID being nil or 0
	if (message.Params.UserID == nil || !checkZeroUserID(message.Params.UserID)) &&
		(message.Params.GroupID == nil || !checkZeroGroupID(message.Params.GroupID)) {
		mylog.Printf("send_group_msgs接收到错误action: %v", message)
		return "", nil
	}
	mylog.Printf("send_group_msg获取到信息类型:%v", msgType)
	var idInt64 int64
	var err error
	var retmsg string

	if message.Params.GroupID != "" {
		idInt64, err = ConvertToInt64(message.Params.GroupID)
	} else if message.Params.UserID != "" {
		idInt64, err = ConvertToInt64(message.Params.UserID)
	}

	//设置递归 对直接向gsk发送action时有效果
	if msgType == "" {
		messageCopy := message
		if err != nil {
			mylog.Printf("错误：无法转换 ID %v\n", err)
		} else {
			// 递归3次
			echo.AddMapping(idInt64, 4)
			// 递归调用handleSendGroupMsg，使用设置的消息类型
			echo.AddMsgType(config.GetAppIDStr(), idInt64, "group_private")
			retmsg, _ = HandleSendGroupMsg(client, api, apiv2, messageCopy)
		}
	} else if echo.GetMapping(idInt64) <= 0 {
		// 特殊值代表不递归
		echo.AddMapping(idInt64, 10)
	}

	switch msgType {
	case "group":
		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params, message, client, api, apiv2)
		var SSM bool

		var originalGroupID, originalUserID string
		// 检查UserID是否为nil
		if message.Params.UserID != nil && config.GetIdmapPro() {
			// 如果UserID不是nil且配置为使用Pro版本，则调用RetrieveRowByIDv2Pro
			originalGroupID, originalUserID, err = idmap.RetrieveRowByIDv2Pro(message.Params.GroupID.(string), message.Params.UserID.(string))
			if err != nil {
				mylog.Printf("Error1 retrieving original GroupID: %v", err)
			}
			mylog.Printf("测试,通过idmaps-pro获取的originalGroupID:%v", originalGroupID)
			if originalGroupID == "" {
				originalGroupID, err = idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
				if err != nil {
					mylog.Printf("Error2 retrieving original GroupID: %v", err)
					return "", nil
				}
				mylog.Printf("测试,通过idmaps获取的originalGroupID:%v", originalGroupID)
			}
		} else {
			// 如果UserID是nil或配置不使用Pro版本，则调用RetrieveRowByIDv2
			originalGroupID, err = idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
			if err != nil {
				mylog.Printf("Error retrieving original GroupID: %v", err)
			}
			// 检查 message.Params.UserID 是否为 nil
			if message.Params.UserID == nil {
				//mylog.Println("UserID is nil")
			} else {
				// 进行类型断言，确认 UserID 不是 nil
				userID, ok := message.Params.UserID.(string)
				if !ok {
					mylog.Println("UserID is not a string")
					// 处理类型断言失败的情况
				} else {
					originalUserID, err = idmap.RetrieveRowByIDv2(userID)
					if err != nil {
						mylog.Printf("Error retrieving original UserID: %v", err)
					}
				}
			}
		}
		message.Params.GroupID = originalGroupID
		message.Params.UserID = originalUserID

		// 检查字符串是否仅包含数字
		isNumeric := func(s string) bool {
			return regexp.MustCompile(`^\d+$`).MatchString(s)
		}

		messageID := message.Params.MessageID.(string)

		if isNumeric(messageID) && messageID != "0" {
			// 当messageID是字符串形式的数字时，执行转换
			RealMsgID, err := idmap.RetrieveRowByIDv2(messageID)
			if err != nil {
				mylog.Printf("error retrieving real MessageID: %v", err)
			} else {
				// 重新赋值，RealMsgID的类型与message.Params.MessageID兼容
				messageID = RealMsgID
			}
		}

		//2000是群主动 此时不能被动转主动
		if SSM {
			//mylog.Printf("正在使用Msgid:%v 补发之前失败的主动信息,请注意AtoP不要设置超过3,否则可能会影响正常信息发送", messageID)
			//mylog.Printf("originalGroupID:%v ", originalGroupID)
			SendStackMessages(apiv2, messageID, message.Params.GroupID.(string))
		}
		mylog.Println("群组发信息messageText:", messageText)

		mylog.Printf("群组发信息使用messageID:[%v]", messageID)
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
		} else if imageURLs, ok := foundItems["url_images"]; ok && len(imageURLs) == 1 {
			imageType = "url_images"
			imageUrl = imageURLs[0]
			imageCount++
		} else if base64Images, ok := foundItems["base64_image"]; ok && len(base64Images) == 1 {
			imageType = "base64_image"
			imageUrl = base64Images[0]
			imageCount++
		}

		if imageCount == 1 && messageText != "" {
			mylog.Printf("发图文混合信息-群")
			// 创建包含单个图片的 singleItem
			singleItem[imageType] = []string{imageUrl}
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			groupReply := generateGroupMessage(messageID, singleItem, "", msgseq+1, apiv2, message.Params.GroupID.(string))
			// 进行类型断言
			richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
			if !ok {
				mylog.Printf("Error: Expected RichMediaMessage type for key ")
				return "", nil
			}
			var groupMessage *dto.MessageToCreate
			var transmd bool
			var md *dto.Markdown
			var kb *keyboard.MessageKeyboard
			//判断是否需要自动转换md
			if config.GetTwoWayEcho() {
				md, kb, transmd = auto_md(message, messageText, richMediaMessage)
			}

			//如果没有转换成md发送
			if !transmd {
				// 上传图片并获取FileInfo
				fileInfo, err := uploadMedia(context.TODO(), message.Params.GroupID.(string), richMediaMessage, apiv2)
				if err != nil {
					mylog.Printf("上传图片失败: %v", err)
					return "", nil // 或其他错误处理
				}
				// 创建包含文本和图像信息的消息
				msgseq = echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				groupMessage = &dto.MessageToCreate{
					Content: messageText, // 添加文本内容
					Media: dto.Media{
						FileInfo: fileInfo, // 添加图像信息
					},
					MsgID:   messageID,
					MsgSeq:  msgseq,
					MsgType: 7, // 假设7是组合消息类型
				}
				groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			} else {
				//将kb和md组合成groupMessage并用MsgType=2发送

				msgseq = echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				groupMessage = &dto.MessageToCreate{
					Content:  "markdown", // 添加文本内容
					MsgID:    messageID,
					MsgSeq:   msgseq,
					Markdown: md,
					Keyboard: kb,
					MsgType:  2, // 假设7是组合消息类型
				}
				groupMessage.Timestamp = time.Now().Unix() // 设置时间戳

			}
			// 发送组合消息
			resp, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
			if err != nil {
				mylog.Printf("发送组合消息失败: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), `"code":22009`) {
				mylog.Printf("信息发送失败,加入到队列中,下次被动信息进行发送")
				var pair echo.MessageGroupPair
				pair.Group = message.Params.GroupID.(string)
				pair.GroupMessage = groupMessage
				echo.PushGlobalStack(pair)
			}

			// 发送成功回执
			retmsg, _ = SendResponse(client, err, &message, resp, api, apiv2)

			delete(foundItems, imageType) // 从foundItems中删除已处理的图片项
			messageText = ""
		}

		// 优先发送文本信息
		if messageText != "" {
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			groupReply := generateGroupMessage(messageID, nil, messageText, msgseq+1, apiv2, message.Params.GroupID.(string))

			// 进行类型断言
			groupMessage, ok := groupReply.(*dto.MessageToCreate)
			if !ok {
				mylog.Println("Error: Expected MessageToCreate type.")
				return "", nil // 或其他错误处理
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			//重新为err赋值
			resp, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
			if err != nil {
				mylog.Printf("发送文本群组信息失败: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), `"code":22009`) {
				mylog.Printf("信息发送失败,加入到队列中,下次被动信息进行发送")
				var pair echo.MessageGroupPair
				pair.Group = message.Params.GroupID.(string)
				pair.GroupMessage = groupMessage
				echo.PushGlobalStack(pair)
			}
			//发送成功回执
			retmsg, _ = SendResponse(client, err, &message, resp, api, apiv2)
		}
		var resp *dto.GroupMessageResponse
		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			for _, url := range urls {
				var singleItem = make(map[string][]string)
				singleItem[key] = []string{url} // 创建一个只包含一个 URL 的 singleItem
				//mylog.Println("singleItem:", singleItem)
				msgseq := echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				groupReply := generateGroupMessage(messageID, singleItem, "", msgseq+1, apiv2, message.Params.GroupID.(string))
				// 进行类型断言
				richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
				if !ok {
					mylog.Printf("Error: Expected RichMediaMessage type for key %s.", key)
					if key == "markdown" || key == "qqmusic" {
						// 进行类型断言
						groupMessage, ok := groupReply.(*dto.MessageToCreate)
						if !ok {
							mylog.Println("Error: Expected MessageToCreate type.")
							return "", nil // 或其他错误处理
						}
						//重新为err赋值
						resp, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
						if err != nil {
							mylog.Printf("发送md信息失败: %v", err)
						}
						if err != nil && strings.Contains(err.Error(), `"code":22009`) {
							mylog.Printf("信息发送失败,加入到队列中,下次被动信息进行发送")
							var pair echo.MessageGroupPair
							pair.Group = message.Params.GroupID.(string)
							pair.GroupMessage = groupMessage
							echo.PushGlobalStack(pair)
						}
						//发送成功回执
						retmsg, _ = SendResponse(client, err, &message, resp, api, apiv2)
					}
					continue // 跳过这个项，继续下一个
				}
				message_return, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), richMediaMessage)
				if err != nil {
					mylog.Printf("发送 %s 信息失败_send_group_msg: %v", key, err)
					if config.GetSendError() { //把报错当作文本发出去
						msgseq := echo.GetMappingSeq(messageID)
						echo.AddMappingSeq(messageID, msgseq+1)
						groupReply := generateGroupMessage(messageID, nil, err.Error(), msgseq+1, apiv2, message.Params.GroupID.(string))
						// 进行类型断言
						groupMessage, ok := groupReply.(*dto.MessageToCreate)
						if !ok {
							mylog.Println("Error: Expected MessageToCreate type.")
							return "", nil // 或其他错误处理
						}
						groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
						//重新为err赋值
						resp, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
						if err != nil {
							mylog.Printf("发送文本报错信息失败: %v", err)
						}
						if err != nil && strings.Contains(err.Error(), `"code":22009`) {
							mylog.Printf("信息发送失败,加入到队列中,下次被动信息进行发送")
							var pair echo.MessageGroupPair
							pair.Group = message.Params.GroupID.(string)
							pair.GroupMessage = groupMessage
							echo.PushGlobalStack(pair)
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
					resp, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
					if err != nil {
						mylog.Printf("发送图片失败: %v", err)
					}
					if err != nil && strings.Contains(err.Error(), `"code":22009`) {
						mylog.Printf("信息发送失败,加入到队列中,下次被动信息进行发送")
						var pair echo.MessageGroupPair
						pair.Group = message.Params.GroupID.(string)
						pair.GroupMessage = groupMessage
						echo.PushGlobalStack(pair)
					}
				}
				//发送成功回执
				retmsg, _ = SendResponse(client, err, &message, resp, api, apiv2)
			}
		}
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		var RChannelID string
		if message.Params.UserID != nil && config.GetIdmapPro() {
			RChannelID, _, err = idmap.RetrieveRowByIDv2Pro(message.Params.ChannelID.(string), message.Params.UserID.(string))
			mylog.Printf("测试,通过Proid获取的RChannelID:%v", RChannelID)
		}
		if RChannelID == "" {
			// 使用RetrieveRowByIDv2还原真实的ChannelID
			RChannelID, err = idmap.RetrieveRowByIDv2(message.Params.ChannelID.(string))
		}
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
		message.Params.ChannelID = RChannelID
		//这一句是group_private的逻辑,发频道信息用的是channelid
		//message.Params.GroupID = value
		retmsg, _ = HandleSendGuildChannelMsg(client, api, apiv2, message)
	case "guild_private":
		//用group_id还原出channelid 这是虚拟成群的私聊信息
		var RChannelID string
		var Vuserid string
		message.Params.ChannelID = message.Params.GroupID.(string)
		Vuserid, ok := message.Params.UserID.(string)
		if !ok {
			mylog.Printf("Error illegal UserID")
			return "", nil
		}
		if Vuserid != "" && config.GetIdmapPro() {
			RChannelID, _, err = idmap.RetrieveRowByIDv2Pro(message.Params.ChannelID.(string), Vuserid)
			mylog.Printf("测试,通过Proid获取的RChannelID:%v", RChannelID)
		} else {
			// 使用RetrieveRowByIDv2还原真实的ChannelID
			RChannelID, err = idmap.RetrieveRowByIDv2(message.Params.ChannelID.(string))
		}
		if err != nil {
			mylog.Printf("error retrieving real ChannelID: %v", err)
		}
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(RChannelID, "guild_id")
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return "", nil
		}
		retmsg, _ = HandleSendGuildChannelPrivateMsg(client, api, apiv2, message, &value, &RChannelID)
	case "group_private":
		//用userid还原出openid 这是虚拟成群的群聊私聊信息
		message.Params.UserID = message.Params.GroupID.(string)
		retmsg, _ = HandleSendPrivateMsg(client, api, apiv2, message)
	case "forum":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		var RChannelID string
		if message.Params.UserID != nil && config.GetIdmapPro() {
			RChannelID, _, err = idmap.RetrieveRowByIDv2Pro(message.Params.ChannelID.(string), message.Params.UserID.(string))
			mylog.Printf("测试,通过Proid获取的RChannelID:%v", RChannelID)
		}
		if RChannelID == "" {
			// 使用RetrieveRowByIDv2还原真实的ChannelID
			RChannelID, err = idmap.RetrieveRowByIDv2(message.Params.ChannelID.(string))
		}
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
		message.Params.ChannelID = RChannelID
		//这一句是group_private的逻辑,发频道信息用的是channelid
		//message.Params.GroupID = value
		retmsg, _ = HandleSendGuildChannelForum(client, api, apiv2, message)
	default:
		mylog.Printf("Unknown message type: %s", msgType)
	}

	// 如果递归id不是10(不递归特殊值)
	if echo.GetMapping(idInt64) != 10 {
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
			retmsg, _ = HandleSendGroupMsg(client, api, apiv2, messageCopy)
		}
	}

	return retmsg, nil
}
