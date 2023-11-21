package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/silk"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_group_msg", handleSendGroupMsg)
}

func handleSendGroupMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}

	//如果获取不到 就用user_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}

	//如果获取不到 就用group_id获取信息类型
	if msgType == "" {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
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
			// 递归调用handleSendGroupMsg，使用设置的消息类型
			echo.AddMsgType(config.GetAppIDStr(), idInt64, "group_private")
			handleSendGroupMsg(client, api, apiv2, messageCopy)
		}
	}

	switch msgType {
	case "group":
		// 解析消息内容
		messageText, foundItems := parseMessageContent(message.Params)

		// 使用 echo 获取消息ID
		var messageID string
		if config.GetLazyMessageId() {
			//由于实现了Params的自定义unmarshell 所以可以类型安全的断言为string
			messageID = echo.GetLazyMessagesId(message.Params.GroupID.(string))
			mylog.Printf("GetLazyMessagesId: %v", messageID)
		}
		if messageID == "" {
			if echoStr, ok := message.Echo.(string); ok {
				messageID = echo.GetMsgIDByKey(echoStr)
				mylog.Println("echo取群组发信息对应的message_id:", messageID)
			}
		}
		//通过bolt数据库还原真实的GroupID
		originalGroupID, err := idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
		if err != nil {
			mylog.Printf("Error retrieving original GroupID: %v", err)
			return
		}
		message.Params.GroupID = originalGroupID
		mylog.Println("群组发信息messageText:", messageText)
		//mylog.Println("foundItems:", foundItems)
		// 如果messageID为空，通过函数获取
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), message.Params.GroupID)
			mylog.Println("通过GetMessageIDByUseridOrGroupid函数获取的message_id:", message.Params.GroupID, messageID)
		}
		//开发环境用
		if config.GetDevMsgID() {
			messageID = "1000"
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
				return // 或其他错误处理
			}

			groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
			//重新为err赋值
			_, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
			if err != nil {
				mylog.Printf("发送文本群组信息失败: %v", err)
			}
			//发送成功回执
			SendResponse(client, err, &message)
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			for _, url := range urls {
				var singleItem = make(map[string][]string)
				singleItem[key] = []string{url} // 创建一个只包含一个 URL 的 singleItem
				//mylog.Println("singleItem:", singleItem)
				msgseq := echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				//时间限制
				lastSendTimestamp := echo.GetMappingFileTimeLimit(messageID)
				if lastSendTimestamp == 0 {
					lastSendTimestamp = echo.GetFileTimeLimit()
				}
				now := time.Now()
				millis := now.UnixMilli()
				diff := millis - lastSendTimestamp
				groupReply := generateGroupMessage(messageID, singleItem, "", msgseq+1)
				// 进行类型断言
				richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
				if !ok {
					mylog.Printf("Error: Expected RichMediaMessage type for key %s.", key)
					continue // 跳过这个项，继续下一个
				}
				mylog.Printf("richMediaMessage: %+v\n", richMediaMessage)
				richMediaMessageCopy := *richMediaMessage // 创建 richMediaMessage 的副本
				mylog.Printf("上次发图(ms): %+v\n", diff)
				if diff < 1000 {
					waitDuration := time.Duration(1200-diff) * time.Millisecond
					mylog.Printf("等待 %v...\n", waitDuration)
					time.AfterFunc(waitDuration, func() {
						mylog.Println("延迟完成")
						_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), richMediaMessageCopy)
						echo.AddMappingFileTimeLimit(messageID, millis)
						echo.AddFileTimeLimit(millis)
						if err != nil {
							mylog.Printf("发送 %s 信息失败_send_group_msg: %v", key, err)
							if config.GetSendError() { //把报错当作文本发出去
								msgseq := echo.GetMappingSeq(messageID)
								echo.AddMappingSeq(messageID, msgseq+1)
								groupReply := generateGroupMessage(messageID, nil, err.Error(), msgseq+1)
								// 进行类型断言
								groupMessage, ok := groupReply.(*dto.MessageToCreate)
								if !ok {
									mylog.Println("Error: Expected MessageToCreate type.")
									return // 或其他错误处理
								}
								groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
								//重新为err赋值
								mylog.Printf("准备发送文本报错信息: %v", groupMessage)
								_, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
								mylog.Printf("发送文本报错信息成功: %v", groupMessage)
								if err != nil {
									mylog.Printf("发送文本报错信息失败: %v", err)
								}
							}
						}
					})
				} else {
					_, err := apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), richMediaMessage)
					echo.AddMappingFileTimeLimit(messageID, millis)
					echo.AddFileTimeLimit(millis)
					if err != nil {
						mylog.Printf("发送 %s 信息失败_send_group_msg: %v", key, err)
						if config.GetSendError() { //把报错当作文本发出去
							msgseq := echo.GetMappingSeq(messageID)
							echo.AddMappingSeq(messageID, msgseq+1)
							groupReply := generateGroupMessage(messageID, nil, err.Error(), msgseq+1)
							// 进行类型断言
							groupMessage, ok := groupReply.(*dto.MessageToCreate)
							if !ok {
								mylog.Println("Error: Expected MessageToCreate type.")
								return // 或其他错误处理
							}
							groupMessage.Timestamp = time.Now().Unix() // 设置时间戳
							//重新为err赋值
							_, err = apiv2.PostGroupMessage(context.TODO(), message.Params.GroupID.(string), groupMessage)
							if err != nil {
								mylog.Printf("发送文本报错信息失败: %v", err)
							}
						}
					}
				}
				//发送成功回执
				SendResponse(client, err, &message)
			}
		}
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err := idmap.RetrieveRowByIDv2(message.Params.ChannelID)
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
		message.Params.ChannelID = RChannelID
		//这一句是group_private的逻辑,发频道信息用的是channelid
		//message.Params.GroupID = value
		handleSendGuildChannelMsg(client, api, apiv2, message)
	case "guild_private":
		//用group_id还原出channelid 这是虚拟成群的私聊信息
		message.Params.ChannelID = message.Params.GroupID.(string)
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err := idmap.RetrieveRowByIDv2(message.Params.ChannelID)
		if err != nil {
			mylog.Printf("error retrieving real ChannelID: %v", err)
		}
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(RChannelID, "guild_id")
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return
		}
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, &value, &message.Params.ChannelID)
	case "group_private":
		//用userid还原出openid 这是虚拟成群的群聊私聊信息
		message.Params.UserID = message.Params.GroupID.(string)
		handleSendPrivateMsg(client, api, apiv2, message)
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

// 整理和组合富媒体信息
func generateGroupMessage(id string, foundItems map[string][]string, messageText string, msgseq int) interface{} {
	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// 从本地路径读取图片
		imageData, err := os.ReadFile(imageURLs[0])
		if err != nil {
			// 读入文件失败
			mylog.Printf("Error reading the image from path %s: %v", imageURLs[0], err)
			// 返回文本信息，提示图片文件不存在
			return &dto.MessageToCreate{
				Content: "错误: 图片文件不存在",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}
		}
		// 首先压缩图片 默认不压缩
		compressedData, err := images.CompressSingleImage(imageData)
		if err != nil {
			mylog.Printf("Error compressing image: %v", err)
			return &dto.MessageToCreate{
				Content: "错误: 压缩图片失败",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}
		}

		// base64编码
		base64Encoded := base64.StdEncoding.EncodeToString(compressedData)

		// 上传base64编码的图片并获取其URL
		imageURL, err := images.UploadBase64ImageToServer(base64Encoded)
		if err != nil {
			mylog.Printf("Error uploading base64 encoded image: %v", err)
			// 如果上传失败，也返回文本信息，提示上传失败
			return &dto.MessageToCreate{
				Content: "错误: 上传图片失败",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}
		}

		// 创建RichMediaMessage并返回，当作URL图片处理
		return &dto.RichMediaMessage{
			EventID:    id,
			FileType:   1, // 1代表图片
			URL:        imageURL,
			Content:    "", // 这个字段文档没有了
			SrvSendMsg: true,
		}
	} else if RecordURLs, ok := foundItems["local_record"]; ok && len(RecordURLs) > 0 {
		// 从本地路径读取语音
		RecordData, err := os.ReadFile(RecordURLs[0])
		if err != nil {
			// 读入文件失败
			mylog.Printf("Error reading the record from path %s: %v", RecordURLs[0], err)
			// 返回文本信息，提示语音文件不存在
			return &dto.MessageToCreate{
				Content: "错误: 语音文件不存在",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}
		}
		//判断并转码
		if !silk.IsAMRorSILK(RecordData) {
			mt, ok := silk.CheckAudio(bytes.NewReader(RecordData))
			if !ok {
				mylog.Errorf("voice type error: " + mt)
				return nil
			}
			RecordData = silk.EncoderSilk(RecordData)
			mylog.Errorf("音频转码ing")
			if err != nil {
				return nil
			}
		}
		// 将解码的语音数据转换回base64格式并上传
		imageURL, err := images.UploadBase64RecordToServer(base64.StdEncoding.EncodeToString(RecordData))
		if err != nil {
			mylog.Printf("failed to upload base64 record: %v", err)
			return nil
		}
		// 创建RichMediaMessage并返回
		return &dto.RichMediaMessage{
			EventID:    id,
			FileType:   3, // 3代表语音
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
		// 适配base64 slik
		if base64_record, ok := foundItems["base64_record"]; ok && len(base64_record) > 0 {
			// 解码base64语音数据
			fileRecordData, err := base64.StdEncoding.DecodeString(base64_record[0])
			if err != nil {
				mylog.Printf("failed to decode base64 record: %v", err)
				return nil
			}
			//判断并转码
			if !silk.IsAMRorSILK(fileRecordData) {
				mt, ok := silk.CheckAudio(bytes.NewReader(fileRecordData))
				if !ok {
					mylog.Errorf("voice type error: " + mt)
					return nil
				}
				fileRecordData = silk.EncoderSilk(fileRecordData)
				mylog.Errorf("音频转码ing")
				if err != nil {
					return nil
				}
			}
			// 将解码的语音数据转换回base64格式并上传
			imageURL, err := images.UploadBase64RecordToServer(base64.StdEncoding.EncodeToString(fileRecordData))
			if err != nil {
				mylog.Printf("failed to upload base64 record: %v", err)
				return nil
			}
			// 创建RichMediaMessage并返回
			return &dto.RichMediaMessage{
				EventID:    id,
				FileType:   3, // 3代表语音
				URL:        imageURL,
				Content:    "", // 这个字段文档没有了
				SrvSendMsg: true,
			}
		}
	} else if base64_image, ok := foundItems["base64_image"]; ok && len(base64_image) > 0 {
		// todo 适配base64图片
		//因为QQ群没有 form方式上传,所以在gensokyo内置了图床,需公网,或以lotus方式连接位于公网的gensokyo
		//要正确的开放对应的端口和设置正确的ip地址在config,这对于一般用户是有一些难度的
		if base64Image, ok := foundItems["base64_image"]; ok && len(base64Image) > 0 {
			// 解码base64图片数据
			fileImageData, err := base64.StdEncoding.DecodeString(base64Image[0])
			if err != nil {
				mylog.Printf("failed to decode base64 image: %v", err)
				return nil
			}
			// 首先压缩图片 默认不压缩
			compressedData, err := images.CompressSingleImage(fileImageData)
			if err != nil {
				mylog.Printf("Error compressing image: %v", err)
				return &dto.MessageToCreate{
					Content: "错误: 压缩图片失败",
					MsgID:   id,
					MsgSeq:  msgseq,
					MsgType: 0, // 默认文本类型
				}
			}
			// 将解码的图片数据转换回base64格式并上传
			imageURL, err := images.UploadBase64ImageToServer(base64.StdEncoding.EncodeToString(compressedData))
			if err != nil {
				mylog.Printf("failed to upload base64 image: %v", err)
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
			MsgSeq:  msgseq,
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
