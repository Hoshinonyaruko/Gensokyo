// 处理收到的信息事件
package Processor

import (
	"log"
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

// ProcessC2CMessage 处理C2C消息 群私聊
func (p *Processors) ProcessC2CMessage(data *dto.WSC2CMessageData) error {
	// 打印data结构体
	PrintStructWithFieldNames(data)

	// 从私信中提取必要的信息 这是测试回复需要用到
	//recipientID := data.Author.ID
	//ChannelID := data.ChannelID
	//sourece是源头频道
	//GuildID := data.GuildID

	//获取当前的s值 当前ws连接所收到的信息条数
	s := client.GetGlobalS()
	if !p.Settings.GlobalPrivateToChannel {
		// 直接转换成ob11私信

		//转换appidstring
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)

		//将真实id转为int userid64
		userid64, err := idmap.StoreIDv2(data.Author.ID)
		if err != nil {
			log.Fatalf("Error storing ID: %v", err)
		}

		//收到私聊信息调用的具体还原步骤
		//1,idmap还原真实userid,
		//发信息使用的是userid

		messageID64, err := idmap.StoreIDv2(data.ID)
		if err != nil {
			log.Fatalf("Error storing ID: %v", err)
		}
		messageID := int(messageID64)
		messageText := data.Content
		// 如果在Array模式下, 则处理Message为Segment格式
		var segmentedMessages interface{} = messageText
		if config.GetArrayValue() {
			segmentedMessages = handlers.ConvertToSegmentedMessage(data)
		}
		privateMsg := OnebotPrivateMessage{
			RawMessage:  messageText,
			Message:     segmentedMessages,
			MessageID:   messageID,
			MessageType: "private",
			PostType:    "message",
			SelfID:      int64(p.Settings.AppID),
			UserID:      userid64,
			Sender: PrivateSender{
				Nickname: "", //这个不支持,但加机器人好友,会收到一个事件,可以对应储存获取,用idmaps可以做到.
				UserID:   userid64,
			},
			SubType: "friend",
			Time:    time.Now().Unix(),
			Avatar:  "", //todo 同上
		}
		// 根据条件判断是否添加Echo字段
		if config.GetTwoWayEcho() {
			privateMsg.Echo = echostr
		}
		// 将当前s和appid和message进行映射
		echo.AddMsgID(AppIDString, s, data.ID)
		echo.AddMsgType(AppIDString, s, "group_private")
		//其实不需要用AppIDString,因为gensokyo是单机器人框架
		//可以试着开发一个,会很棒的
		echo.AddMsgID(AppIDString, userid64, data.ID)
		//懒message_id池
		echo.AddLazyMessageId(strconv.FormatInt(userid64, 10), data.ID, time.Now())
		//储存类型
		echo.AddMsgType(AppIDString, userid64, "group_private")
		//储存当前群或频道号的类型 私信不需要
		//idmap.WriteConfigv2(data.ChannelID, "type", "group_private")

		// 调试
		PrintStructWithFieldNames(privateMsg)

		// Convert OnebotGroupMessage to map and send
		privateMsgMap := structToMap(privateMsg)
		//上报信息到onebotv11应用端(正反ws)
		p.BroadcastMessageToAll(privateMsgMap)
	} else {
		//将私聊信息转化为群信息(特殊需求情况下)

		//转换at
		messageText := handlers.RevertTransformedText(data)
		if messageText == "" {
			mylog.Printf("信息被自定义黑白名单拦截")
			return nil
		}
		//转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		//构造echo
		echostr := AppIDString + "_" + strconv.FormatInt(s, 10)
		//把userid作为群号
		//映射str的userid到int
		userid64, err := idmap.StoreIDv2(data.Author.ID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		//映射str的messageID到int
		messageID64, err := idmap.StoreIDv2(data.ID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		messageID := int(messageID64)
		//todo 判断array模式 然后对Message处理成array格式
		groupMsg := OnebotGroupMessage{
			RawMessage:  messageText,
			Message:     messageText,
			MessageID:   messageID,
			GroupID:     userid64,
			MessageType: "group",
			PostType:    "message",
			SelfID:      int64(p.Settings.AppID),
			UserID:      userid64,
			Sender: Sender{
				Nickname: "",
				UserID:   userid64,
			},
			SubType: "normal",
			Time:    time.Now().Unix(),
			Avatar:  "",
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
		echo.AddMsgType(AppIDString, s, "group_private")
		//为不支持双向echo的ob服务端映射
		echo.AddMsgID(AppIDString, userid64, data.ID)
		echo.AddMsgType(AppIDString, userid64, "group_private")
		//懒message_id池
		echo.AddLazyMessageId(strconv.FormatInt(userid64, 10), data.ID, time.Now())

		//调试
		PrintStructWithFieldNames(groupMsg)

		// Convert OnebotGroupMessage to map and send
		groupMsgMap := structToMap(groupMsg)
		//上报信息到onebotv11应用端(正反ws)
		p.BroadcastMessageToAll(groupMsgMap)
	}
	return nil
}
