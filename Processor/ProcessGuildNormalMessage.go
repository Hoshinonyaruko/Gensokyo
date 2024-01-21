// 处理收到的信息事件
package Processor

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket/client"
)

// ProcessGuildNormalMessage 处理频道常规消息
func (p *Processors) ProcessGuildNormalMessage(data *dto.WSMessageData) error {
	var AppIDString string
	if !p.Settings.GlobalChannelToGroup {
		// 将时间字符串转换为时间戳
		t, err := time.Parse(time.RFC3339, string(data.Timestamp))
		if err != nil {
			return fmt.Errorf("error parsing time: %v", err)
		}
		//获取s
		s := client.GetGlobalS()
		//转换at
		messageText := handlers.RevertTransformedText(data, "guild", p.Api, p.Apiv2, 10000, 10000, config.GetWhiteEnable(2)) //这里未转换
		if messageText == "" {
			mylog.Printf("信息被自定义黑白名单拦截")
			return nil
		}
		//框架内指令
		p.HandleFrameworkCommand(messageText, data, "guild")
		//转换appid
		AppIDString = strconv.FormatUint(p.Settings.AppID, 10)
		//构造echo
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)
		//映射str的userid到int
		userid64, err := idmap.StoreIDv2(data.Author.ID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		// 如果在Array模式下, 则处理Message为Segment格式
		var segmentedMessages interface{} = messageText
		if config.GetArrayValue() {
			segmentedMessages = handlers.ConvertToSegmentedMessage(data)
		}
		// 处理onebot_channel_message逻辑
		onebotMsg := OnebotChannelMessage{
			ChannelID:   data.ChannelID,
			GuildID:     data.GuildID,
			Message:     segmentedMessages,
			RawMessage:  messageText,
			MessageID:   data.ID,
			MessageType: "guild",
			PostType:    "message",
			SelfID:      int64(p.Settings.AppID),
			UserID:      userid64,
			SelfTinyID:  "0",
			Sender: Sender{
				Nickname: data.Member.Nick,
				TinyID:   "0",
				UserID:   userid64,
				Card:     data.Member.Nick,
				Sex:      "0",
				Age:      0,
				Area:     "0",
				Level:    "0",
			},
			SubType: "channel",
			Time:    t.Unix(),
			Avatar:  data.Author.Avatar,
		}
		// 根据条件判断是否添加Echo字段
		if config.GetTwoWayEcho() {
			onebotMsg.Echo = echostr
			//用向应用端(如果支持)发送echo,来确定客户端的send_msg对应的触发词原文
			echo.AddMsgIDv3(AppIDString, echostr, messageText)
		}
		// 获取MasterID数组
		masterIDs := config.GetMasterID()

		// 判断userid64是否在masterIDs数组里
		isMaster := false
		for _, id := range masterIDs {
			if strconv.FormatInt(userid64, 10) == id {
				isMaster = true
				break
			}
		}

		// 根据isMaster的值为groupMsg的Sender赋值role字段
		if isMaster {
			onebotMsg.Sender.Role = "owner"
		} else {
			onebotMsg.Sender.Role = "member"
		}
		//将当前s和appid和message进行映射
		echo.AddMsgID(AppIDString, s, data.ID)
		echo.AddMsgType(AppIDString, s, "guild")
		//为不支持双向echo的ob11服务端映射
		echo.AddMsgID(AppIDString, userid64, data.ID)
		//映射类型
		echo.AddMsgType(AppIDString, userid64, "guild")
		//储存当前群或频道号的类型
		idmap.WriteConfigv2(data.ChannelID, "type", "guild")
		//todo 完善频道ob信息
		//懒message_id池
		echo.AddLazyMessageId(data.ChannelID, data.ID, time.Now())
		//懒message_id池
		//echo.AddLazyMessageId(strconv.FormatInt(userid64, 10), data.ID, time.Now())
		//echo.AddLazyMessageIdv2(data.ChannelID, strconv.FormatInt(userid64, 10), data.ID, time.Now())

		//调试
		PrintStructWithFieldNames(onebotMsg)

		// 将 onebotMsg 结构体转换为 map[string]interface{}
		msgMap := structToMap(onebotMsg)

		//上报信息到onebotv11应用端(正反ws)
		p.BroadcastMessageToAll(msgMap)
	} else {
		// GlobalChannelToGroup为true时的处理逻辑
		//将频道转化为一个群
		//获取s
		s := client.GetGlobalS()
		var userid64 int64
		var ChannelID64 int64
		var err error
		if config.GetIdmapPro() {
			//将真实id转为int userid64
			ChannelID64, userid64, err = idmap.StoreIDv2Pro(data.ChannelID, data.Author.ID)
			if err != nil {
				mylog.Fatalf("Error storing ID: %v", err)
			}
			//当参数不全时
			_, _ = idmap.StoreIDv2(data.ChannelID)
			_, _ = idmap.StoreIDv2(data.Author.ID)
			if !config.GetHashIDValue() {
				mylog.Fatalf("避坑日志:你开启了高级id转换,请设置hash_id为true,并且删除idmaps并重启")
			}
			//补救措施
			idmap.SimplifiedStoreID(data.Author.ID)
			//补救措施
			idmap.SimplifiedStoreID(data.ChannelID)
			//补救措施
			echo.AddMsgIDv3(AppIDString, data.ChannelID, data.ID)
		} else {
			//将channelid写入ini,可取出guild_id
			ChannelID64, err = idmap.StoreIDv2(data.ChannelID)
			if err != nil {
				mylog.Printf("Error storing ID: %v", err)
				return nil
			}
			//映射str的userid到int
			userid64, err = idmap.StoreIDv2(data.Author.ID)
			if err != nil {
				mylog.Printf("Error storing ID: %v", err)
				return nil
			}
		}
		//转成int再互转
		idmap.WriteConfigv2(fmt.Sprint(ChannelID64), "guild_id", data.GuildID)
		//储存原来的(获取群列表需要)
		idmap.WriteConfigv2(data.ChannelID, "guild_id", data.GuildID)
		//转换at
		messageText := handlers.RevertTransformedText(data, "guild", p.Api, p.Apiv2, ChannelID64, userid64, config.GetWhiteEnable(2))
		if messageText == "" {
			mylog.Printf("信息被自定义黑白名单拦截")
			return nil
		}
		//框架内指令
		p.HandleFrameworkCommand(messageText, data, "guild")
		//转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		//构造echo
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)
		//映射str的messageID到int
		messageID64, err := idmap.StoreIDv2(data.ID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		messageID := int(messageID64)
		// 如果在Array模式下, 则处理Message为Segment格式
		var segmentedMessages interface{} = messageText
		if config.GetArrayValue() {
			segmentedMessages = handlers.ConvertToSegmentedMessage(data)
		}
		var IsBindedUserId, IsBindedGroupId bool
		if config.GetHashIDValue() {
			IsBindedUserId = idmap.CheckValue(data.Author.ID, userid64)
			IsBindedGroupId = idmap.CheckValue(data.ChannelID, ChannelID64)
		} else {
			IsBindedUserId = idmap.CheckValuev2(userid64)
			IsBindedGroupId = idmap.CheckValuev2(ChannelID64)
		}
		groupMsg := OnebotGroupMessage{
			RawMessage:  messageText,
			Message:     segmentedMessages,
			MessageID:   messageID,
			GroupID:     ChannelID64,
			MessageType: "group",
			PostType:    "message",
			SelfID:      int64(p.Settings.AppID),
			UserID:      userid64,
			Sender: Sender{
				Nickname: data.Member.Nick,
				UserID:   userid64,
				Card:     data.Member.Nick,
				Sex:      "0",
				Age:      0,
				Area:     "",
				Level:    "0",
			},
			SubType: "normal",
			Time:    time.Now().Unix(),
		}
		//增强配置
		if !config.GetNativeOb11() {
			groupMsg.RealMessageType = "guild"
			groupMsg.IsBindedUserId = IsBindedUserId
			groupMsg.IsBindedGroupId = IsBindedGroupId
			groupMsg.Avatar = data.Author.Avatar
		}
		// 根据条件判断是否添加Echo字段
		if config.GetTwoWayEcho() {
			groupMsg.Echo = echostr
			//用向应用端(如果支持)发送echo,来确定客户端的send_msg对应的触发词原文
			echo.AddMsgIDv3(AppIDString, echostr, messageText)
		}
		// 获取MasterID数组
		masterIDs := config.GetMasterID()

		// 判断userid64是否在masterIDs数组里
		isMaster := false
		for _, id := range masterIDs {
			if strconv.FormatInt(userid64, 10) == id {
				isMaster = true
				break
			}
		}

		// 频道转群时获取频道身份组
		// 频道身份组文档https://bot.q.qq.com/wiki/develop/api-v2/server-inter/channel/role/member/role_model.html#role
		channelRoleName := "member"
		for _, role := range data.Member.Roles {
			switch role {
			case "4":
				channelRoleName = "owner" //群主/创建者为4
			case "2":
				channelRoleName = "admin" //管理员（超级管理员）为2
			}
		}

		// 根据isMaster的值为groupMsg的Sender赋值role字段
		if isMaster {
			groupMsg.Sender.Role = "owner"
		} else {
			groupMsg.Sender.Role = channelRoleName
		}
		//将当前s和appid和message进行映射
		echo.AddMsgID(AppIDString, s, data.ID)
		echo.AddMsgType(AppIDString, s, "guild")
		//为不支持双向echo的ob服务端映射
		echo.AddMsgID(AppIDString, ChannelID64, data.ID)
		//将当前的userid和groupid和msgid进行一个更稳妥的映射
		echo.AddMsgIDv2(AppIDString, ChannelID64, userid64, data.ID)
		//储存当前群或频道号的类型
		idmap.WriteConfigv2(fmt.Sprint(ChannelID64), "type", "guild")
		echo.AddMsgType(AppIDString, ChannelID64, "guild")
		//懒message_id池
		echo.AddLazyMessageId(strconv.FormatInt(ChannelID64, 10), data.ID, time.Now())
		//懒message_id池
		//echo.AddLazyMessageId(strconv.FormatInt(userid64, 10), data.ID, time.Now())
		//echo.AddLazyMessageIdv2(strconv.FormatInt(ChannelID64, 10), strconv.FormatInt(userid64, 10), data.ID, time.Now())

		//调试
		PrintStructWithFieldNames(groupMsg)

		// Convert OnebotGroupMessage to map and send
		groupMsgMap := structToMap(groupMsg)

		//上报信息到onebotv11应用端(正反ws)
		p.BroadcastMessageToAll(groupMsgMap)
	}

	return nil
}
