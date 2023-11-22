package handlers

import (
	"context"
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
	callapi.RegisterHandler("send_msg", handleSendMsg)
}

func handleSendMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
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
			// 递归调用handleSendMsg，使用设置的消息类型
			echo.AddMsgType(config.GetAppIDStr(), idInt64, "group_private")
			handleSendMsg(client, api, apiv2, messageCopy)
		}
	}

	switch msgType {
	case "group":
		//复用处理逻辑
		handleSendGroupMsg(client, api, apiv2, message)
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err := idmap.RetrieveRowByIDv2(message.Params.ChannelID)
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
		message.Params.ChannelID = RChannelID
		handleSendGuildChannelMsg(client, api, apiv2, message)
	case "guild_private":
		//send_msg比具体的send_xxx少一层,其包含的字段类型在虚拟化场景已经失去作用
		//根据userid绑定得到的具体真实事件类型,这里也有多种可能性
		//1,私聊(但虚拟成了群),这里用群号取得需要的id
		//2,频道私聊(但虚拟成了私聊)这里传递2个nil,用user_id去推测channel_id和guild_id
		handleSendGuildChannelPrivateMsg(client, api, apiv2, message, nil, nil)
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
			_, err = apiv2.PostC2CMessage(context.TODO(), UserID, groupMessage)
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
				singleItem[key] = []string{url} // 创建一个只包含一个 URL 的 singleItem
				msgseq := echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				groupReply := generateGroupMessage(messageID, singleItem, "", msgseq+1)

				// 进行类型断言
				richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
				if !ok {
					mylog.Printf("Error: Expected RichMediaMessage type for key %s.", key)
					continue // 跳过这个项，继续下一个
				}
				_, err = apiv2.PostC2CMessage(context.TODO(), UserID, richMediaMessage)
				if err != nil {
					mylog.Printf("发送 %s 私聊信息失败: %v", key, err)
				}
				//发送成功回执
				SendResponse(client, err, &message)
			}
		}
	default:
		mylog.Printf("1Unknown message type: %s", msgType)
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
		handleSendMsg(client, api, apiv2, messageCopy)
	}
}

// 通过user_id获取messageID
func GetMessageIDByUseridOrGroupid(appID string, userID interface{}) string {
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
	//将真实id转为int
	userid64, err := idmap.StoreIDv2(userIDStr)
	if err != nil {
		mylog.Fatalf("Error storing ID 241: %v", err)
		return ""
	}
	key := appID + "_" + fmt.Sprint(userid64)
	return echo.GetMsgIDByKey(key)
}
