package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_msg", HandleSendMsg)
}

func HandleSendMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	var retmsg string
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
	var idInt64 int64
	var err error

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
			// 递归调用handleSendMsg，使用设置的消息类型
			echo.AddMsgType(config.GetAppIDStr(), idInt64, "group_private")
			retmsg, _ = HandleSendMsg(client, api, apiv2, messageCopy)
		}
	}

	switch msgType {
	case "group":
		//复用处理逻辑
		retmsg, _ = HandleSendGroupMsg(client, api, apiv2, message)
	case "guild":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		message.Params.ChannelID = message.Params.GroupID.(string)
		var RChannelID string
		if message.Params.UserID != nil && config.GetIdmapPro() {
			RChannelID, _, err = idmap.RetrieveRowByIDv2Pro(message.Params.ChannelID, message.Params.UserID.(string))
			mylog.Printf("测试,通过Proid获取的RChannelID:%v", RChannelID)
		}
		if RChannelID == "" {
			// 使用RetrieveRowByIDv2还原真实的ChannelID
			RChannelID, err = idmap.RetrieveRowByIDv2(message.Params.ChannelID)
		}
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
		message.Params.ChannelID = RChannelID
		retmsg, _ = HandleSendGuildChannelMsg(client, api, apiv2, message)
	case "guild_private":
		//send_msg比具体的send_xxx少一层,其包含的字段类型在虚拟化场景已经失去作用
		//根据userid绑定得到的具体真实事件类型,这里也有多种可能性
		//1,私聊(但虚拟成了群),这里用群号取得需要的id
		//2,频道私聊(但虚拟成了私聊)这里传递2个nil,用user_id去推测channel_id和guild_id
		retmsg, _ = HandleSendGuildChannelPrivateMsg(client, api, apiv2, message, nil, nil)
	case "group_private":
		//私聊信息
		retmsg, _ = HandleSendPrivateMsg(client, api, apiv2, message)
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
		delay := config.GetSendDelay()
		time.Sleep(time.Duration(delay) * time.Millisecond)
		retmsg, _ = HandleSendMsg(client, api, apiv2, messageCopy)
	}
	return retmsg, nil
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
		mylog.Printf("Error storing ID 241: %v", err)
		return ""
	}
	key := appID + "_" + fmt.Sprint(userid64)
	mylog.Printf("GetMessageIDByUseridOrGroupid_key:%v", key)
	messageid := echo.GetMsgIDByKey(key)
	if messageid == "" {
		key := appID + "_" + userIDStr
		mylog.Printf("GetMessageIDByUseridOrGroupid_key_2:%v", key)
		messageid = echo.GetMsgIDByKey(key)
	}
	return messageid
}

// 通过user_id获取messageID
func GetMessageIDByUseridAndGroupid(appID string, userID interface{}, groupID interface{}) string {
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
	// 从appID和userID生成key
	var GroupIDStr string
	switch u := groupID.(type) {
	case int:
		GroupIDStr = strconv.Itoa(u)
	case int64:
		GroupIDStr = strconv.FormatInt(u, 10)
	case float64:
		GroupIDStr = strconv.FormatFloat(u, 'f', 0, 64)
	case string:
		GroupIDStr = u
	default:
		// 可能需要处理其他类型或报错
		return ""
	}
	var userid64, groupid64 int64
	var err error
	if config.GetIdmapPro() {
		//将真实id转为int userid64
		groupid64, userid64, err = idmap.StoreIDv2Pro(GroupIDStr, userIDStr)
		if err != nil {
			mylog.Fatalf("Error storing ID 210: %v", err)
		}
	} else {
		//将真实id转为int
		userid64, err = idmap.StoreIDv2(userIDStr)
		if err != nil {
			mylog.Fatalf("Error storing ID 241: %v", err)
			return ""
		}
		//将真实id转为int
		groupid64, err = idmap.StoreIDv2(GroupIDStr)
		if err != nil {
			mylog.Fatalf("Error storing ID 256: %v", err)
			return ""
		}
	}
	key := appID + "_" + fmt.Sprint(groupid64) + "_" + fmt.Sprint(userid64)
	mylog.Printf("GetMessageIDByUseridAndGroupid_key:%v", key)
	return echo.GetMsgIDByKey(key)
}
