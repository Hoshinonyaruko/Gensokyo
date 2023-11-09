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
	if !p.Settings.GlobalChannelToGroup {
		// 将时间字符串转换为时间戳
		t, err := time.Parse(time.RFC3339, string(data.Timestamp))
		if err != nil {
			return fmt.Errorf("error parsing time: %v", err)
		}
		//获取s
		s := client.GetGlobalS()
		//转换at
		messageText := handlers.RevertTransformedText(data)
		//转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
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
			SelfTinyID:  "",
			Sender: Sender{
				Nickname: data.Member.Nick,
				TinyID:   "",
				UserID:   userid64,
			},
			SubType: "channel",
			Time:    t.Unix(),
			Avatar:  data.Author.Avatar,
		}
		// 根据条件判断是否添加Echo字段
		if config.GetTwoWayEcho() {
			onebotMsg.Echo = echostr
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
		echo.AddMsgType(AppIDString, userid64, "guild")
		//储存当前群或频道号的类型
		idmap.WriteConfigv2(data.ChannelID, "type", "guild")
		//todo 完善频道ob信息
		//懒message_id池
		echo.AddLazyMessageId(data.ChannelID, data.ID, time.Now())

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
		//将channelid写入ini,可取出guild_id
		ChannelID64, err := idmap.StoreIDv2(data.ChannelID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		//转成int再互转
		idmap.WriteConfigv2(fmt.Sprint(ChannelID64), "guild_id", data.GuildID)
		//转换at
		messageText := handlers.RevertTransformedText(data)
		//转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		//构造echo
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)
		//映射str的userid到int
		userid64, err := idmap.StoreIDv2(data.Author.ID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		//userid := int(userid64)
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
			},
			SubType: "normal",
			Time:    time.Now().Unix(),
			Avatar:  data.Author.Avatar,
		}
		// 根据条件判断是否添加Echo字段
		if config.GetTwoWayEcho() {
			groupMsg.Echo = echostr
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
			groupMsg.Sender.Role = "owner"
		} else {
			groupMsg.Sender.Role = "member"
		}
		//将当前s和appid和message进行映射
		echo.AddMsgID(AppIDString, s, data.ID)
		echo.AddMsgType(AppIDString, s, "guild")
		//为不支持双向echo的ob服务端映射
		echo.AddMsgID(AppIDString, ChannelID64, data.ID)
		echo.AddMsgType(AppIDString, ChannelID64, "guild")
		//储存当前群或频道号的类型
		idmap.WriteConfigv2(fmt.Sprint(ChannelID64), "type", "guild")
		echo.AddMsgType(AppIDString, ChannelID64, "guild")
		//懒message_id池
		echo.AddLazyMessageId(strconv.FormatInt(ChannelID64, 10), data.ID, time.Now())

		//调试
		PrintStructWithFieldNames(groupMsg)

		// Convert OnebotGroupMessage to map and send
		groupMsgMap := structToMap(groupMsg)

		//上报信息到onebotv11应用端(正反ws)
		p.BroadcastMessageToAll(groupMsgMap)
	}

	return nil
}
