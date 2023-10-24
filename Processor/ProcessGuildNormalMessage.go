// 处理收到的信息事件
package Processor

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket/client"
)

// ProcessGuildNormalMessage 处理频道常规消息
func (p *Processor) ProcessGuildNormalMessage(data *dto.WSMessageData) error {
	if !p.Settings.GlobalChannelToGroup {
		// 将时间字符串转换为时间戳
		t, err := time.Parse(time.RFC3339, string(data.Timestamp))
		if err != nil {
			return fmt.Errorf("error parsing time: %v", err)
		}
		//获取s
		s := client.GetGlobalS()
		//转换at
		messageText := handlers.RevertTransformedText(data.Content)
		//转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		//构造echo
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)
		//映射str的userid到int
		userid64, err := idmap.StoreID(data.Author.ID)
		if err != nil {
			log.Printf("Error storing ID: %v", err)
			return nil
		}
		//映射str的messageID到int
		//可以是string
		// messageID64, err := idmap.StoreID(data.ID)
		// if err != nil {
		// 	log.Printf("Error storing ID: %v", err)
		// 	return nil
		// }
		// messageID := int(messageID64)
		// 处理onebot_channel_message逻辑
		onebotMsg := OnebotChannelMessage{
			ChannelID:   data.ChannelID,
			GuildID:     data.GuildID,
			Message:     messageText,
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
			Echo:    echostr,
		}

		//将当前s和appid和message进行映射
		echo.AddMsgID(AppIDString, s, data.ID)
		echo.AddMsgType(AppIDString, s, "guild")
		//为不支持双向echo的ob11服务端映射
		echo.AddMsgID(AppIDString, userid64, data.ID)
		echo.AddMsgType(AppIDString, userid64, "guild")

		//调试
		PrintStructWithFieldNames(onebotMsg)

		// 将 onebotMsg 结构体转换为 map[string]interface{}
		msgMap := structToMap(onebotMsg)

		// 使用 wsclient 发送消息
		err = p.Wsclient.SendMessage(msgMap)
		if err != nil {
			return fmt.Errorf("error sending message via wsclient: %v", err)
		}

	} else {
		// GlobalChannelToGroup为true时的处理逻辑
		//将频道转化为一个群
		//获取s
		s := client.GetGlobalS()
		//将channelid写入ini,可取出guild_id todo 比ini更好的储存方式
		idmap.WriteConfig(data.ChannelID, "guild_id", data.GuildID)
		//转换at
		messageText := handlers.RevertTransformedText(data.Content)
		//转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		//构造echo
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)
		//把频道号作为群号
		channelIDInt, err := strconv.Atoi(data.ChannelID)
		if err != nil {
			// handle error, perhaps return it
			return fmt.Errorf("failed to convert ChannelID to int: %v", err)
		}
		//映射str的userid到int
		userid64, err := idmap.StoreID(data.Author.ID)
		if err != nil {
			log.Printf("Error storing ID: %v", err)
			return nil
		}
		//userid := int(userid64)
		//映射str的messageID到int
		messageID64, err := idmap.StoreID(data.ID)
		if err != nil {
			log.Printf("Error storing ID: %v", err)
			return nil
		}
		messageID := int(messageID64)
		//todo 判断array模式 然后对Message处理成array格式
		groupMsg := OnebotGroupMessage{
			RawMessage:  messageText,
			Message:     messageText,
			MessageID:   messageID,
			GroupID:     int64(channelIDInt),
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
			Echo:    echostr,
		}
		//将当前s和appid和message进行映射
		echo.AddMsgID(AppIDString, s, data.ID)
		echo.AddMsgType(AppIDString, s, "guild")
		//为不支持双向echo的ob服务端映射
		echo.AddMsgID(AppIDString, int64(channelIDInt), data.ID)
		echo.AddMsgType(AppIDString, int64(channelIDInt), "guild")

		//调试
		PrintStructWithFieldNames(groupMsg)

		// Convert OnebotGroupMessage to map and send
		groupMsgMap := structToMap(groupMsg)
		err = p.Wsclient.SendMessage(groupMsgMap)
		if err != nil {
			return fmt.Errorf("error sending group message via wsclient: %v", err)
		}

	}

	return nil
}
