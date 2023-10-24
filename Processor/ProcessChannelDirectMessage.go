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

// ProcessChannelDirectMessage 处理频道私信消息 这里我们是被动收到
func (p *Processor) ProcessChannelDirectMessage(data *dto.WSDirectMessageData) error {
	// 打印data结构体
	//PrintStructWithFieldNames(data)

	// 从私信中提取必要的信息 这是测试回复需要用到
	//recipientID := data.Author.ID
	//ChannelID := data.ChannelID
	//sourece是源头频道
	//GuildID := data.GuildID

	//获取当前的s值 当前ws连接所收到的信息条数
	s := client.GetGlobalS()
	if !p.Settings.GlobalPrivateToChannel {
		// 把频道类型的私信转换成普通ob11的私信

		//转换appidstring
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)

		//将真实id转为int userid64
		userid64, err := idmap.StoreID(data.Author.ID)
		if err != nil {
			log.Fatalf("Error storing ID: %v", err)
		}
		//将真实id写入数据库,可取出ChannelID
		idmap.WriteConfig(data.Author.ID, "channel_id", data.ChannelID)
		//将channelid写入数据库,可取出guild_id
		idmap.WriteConfig(data.ChannelID, "guild_id", data.GuildID)

		//收到私聊信息调用的具体还原步骤
		//1,idmap还原真实userid,
		//2,通过idmap获取channelid,
		//3,通过idmap用channelid获取guildid,
		//发信息使用的是guildid
		//todo 优化数据库读写次数
		messageID64, err := idmap.StoreID(data.ID)
		if err != nil {
			log.Fatalf("Error storing ID: %v", err)
		}
		messageID := int(messageID64)

		privateMsg := OnebotPrivateMessage{
			RawMessage:  data.Content,
			Message:     data.Content,
			MessageID:   messageID,
			MessageType: "private",
			PostType:    "message",
			SelfID:      int64(p.Settings.AppID),
			UserID:      userid64,
			Sender: PrivateSender{
				Nickname: data.Member.Nick,
				UserID:   userid64,
			},
			SubType: "friend",
			Time:    time.Now().Unix(),
			Avatar:  data.Author.Avatar,
			Echo:    echostr,
		}

		// 将当前s和appid和message进行映射
		echo.AddMsgID(AppIDString, s, data.ID)
		echo.AddMsgType(AppIDString, s, "guild_private")
		//其实不需要用AppIDString,因为gensokyo是单机器人框架
		echo.AddMsgID(AppIDString, userid64, data.ID)
		echo.AddMsgType(AppIDString, userid64, "guild_private")

		// 调试
		PrintStructWithFieldNames(privateMsg)

		// Convert OnebotGroupMessage to map and send
		privateMsgMap := structToMap(privateMsg)
		err = p.Wsclient.SendMessage(privateMsgMap)
		if err != nil {
			return fmt.Errorf("error sending group message via wsclient: %v", err)
		}
	} else {
		if !p.Settings.GlobalChannelToGroup {
			//将频道私信作为普通频道信息

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
			//OnebotChannelMessage
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
			//通过echo始终得知真实的事件类型,来对应调用正确的api
			echo.AddMsgType(AppIDString, s, "guild_private")
			//为不支持双向echo的ob服务端映射
			echo.AddMsgID(AppIDString, userid64, data.ID)
			echo.AddMsgType(AppIDString, userid64, "guild_private")

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
			//将频道信息转化为群信息(特殊需求情况下)
			//将channelid写入ini,可取出guild_id
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
			echo.AddMsgType(AppIDString, s, "guild_private")
			//为不支持双向echo的ob服务端映射
			echo.AddMsgID(AppIDString, userid64, data.ID)
			echo.AddMsgType(AppIDString, userid64, "guild_private")

			//调试
			PrintStructWithFieldNames(groupMsg)

			// Convert OnebotGroupMessage to map and send
			groupMsgMap := structToMap(groupMsg)
			err = p.Wsclient.SendMessage(groupMsgMap)
			if err != nil {
				return fmt.Errorf("error sending group message via wsclient: %v", err)
			}
		}

	}
	return nil
}
