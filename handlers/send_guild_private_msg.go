package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"

	"github.com/hoshinonyaruko/gensokyo/echo"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

// 调用send_private_msg即可 一样的

// func init() {
// 	callapi.RegisterHandler("send_guild_private_msg", HandleSendGuildChannelPrivateMsg)
// }

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
	if messageID == "2000" {
		messageID = ""
		mylog.Println("通过lazymsgid发送频道私聊主动信息,若非主动信息请提交issue")
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
				// 压缩图片
				compressedData, err := images.CompressSingleImage(fileImageData)
				if err != nil {
					mylog.Printf("Error compressing image: %v", err)
				}
				if _, err = api.PostDirectMessageMultipart(context.TODO(), dm, reply, compressedData); err != nil {
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
