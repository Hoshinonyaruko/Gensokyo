// 处理收到的回调事件
package Processor

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

var (
	// 用户或群组ID到下一次允许执行操作的时间戳的映射
	nextAllowedTime = make(map[string]time.Time)
	mu              sync.Mutex
)

// ProcessInlineSearch 处理内联查询
func (p *Processors) ProcessInlineSearch(data *dto.WSInteractionData) error {
	// 转换appid
	var userid64 int64
	var GroupID64 int64
	var err error
	var fromgid, fromuid string
	if data.GroupOpenID != "" {
		fromgid = data.GroupOpenID
		fromuid = data.GroupMemberOpenID
	} else {
		fromgid = data.ChannelID
		fromuid = "0"
	}
	//这里处理自动handle回调回应
	if config.GetAutoPutInteraction() {
		DelayedPutInteraction(p.Api, data.ID, fromuid, fromgid)
	}
	if config.GetIdmapPro() {
		//将真实id转为int userid64
		GroupID64, userid64, err = idmap.StoreIDv2Pro(fromgid, fromuid)
		if err != nil {
			mylog.Fatalf("Error storing ID: %v", err)
		}
		//当参数不全
		_, _ = idmap.StoreIDv2(fromgid)
		_, _ = idmap.StoreIDv2(fromuid)
		if !config.GetHashIDValue() {
			mylog.Fatalf("避坑日志:你开启了高级id转换,请设置hash_id为true,并且删除idmaps并重启")
		}
	} else {
		// 映射str的GroupID到int
		GroupID64, err = idmap.StoreIDv2(fromgid)
		if err != nil {
			mylog.Errorf("failed to convert ChannelID to int: %v", err)
			return nil
		}
		// 映射str的userid到int
		userid64, err = idmap.StoreIDv2(fromuid)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
	}
	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}

	if !config.GetGlobalInteractionToMessage() {
		notice := &OnebotInteractionNotice{
			GroupID:    GroupID64,
			NoticeType: "interaction",
			PostType:   "notice",
			SelfID:     selfid64,
			SubType:    "create",
			Time:       time.Now().Unix(),
			UserID:     userid64,
			Data:       data,
		}
		//调试
		PrintStructWithFieldNames(notice)

		// Convert OnebotGroupMessage to map and send
		noticeMap := structToMap(notice)

		//上报信息到onebotv11应用端(正反ws)
		p.BroadcastMessageToAll(noticeMap)

		// 转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

		// 储存和群号相关的eventid
		echo.AddEvnetID(AppIDString, GroupID64, data.ID)
	} else {
		if data.GroupOpenID != "" {
			//群回调
			newdata := ConvertInteractionToMessage(data)
			mylog.Printf("回调测试111-newdata:%v\n", newdata)
			segmentedMessages := handlers.ConvertToSegmentedMessage(newdata)
			//映射str的messageID到int
			messageID64, err := idmap.StoreIDv2(data.ID)
			if err != nil {
				mylog.Printf("Error storing ID: %v", err)
				return nil
			}
			messageID := int(messageID64)
			var selfid64 int64
			if config.GetUseUin() {
				selfid64 = config.GetUinint64()
			} else {
				selfid64 = int64(p.Settings.AppID)
			}
			mylog.Printf("回调测试-interaction:%v\n", segmentedMessages)
			groupMsg := OnebotGroupMessage{
				RawMessage:  data.Data.Resolved.ButtonData,
				Message:     segmentedMessages,
				MessageID:   messageID,
				GroupID:     GroupID64,
				MessageType: "group",
				PostType:    "message",
				SelfID:      selfid64,
				UserID:      userid64,
				Sender: Sender{
					UserID: userid64,
					Sex:    "0",
					Age:    0,
					Area:   "0",
					Level:  "0",
				},
				SubType: "normal",
				Time:    time.Now().Unix(),
			}
			//增强配置
			if !config.GetNativeOb11() {
				groupMsg.RealMessageType = "interaction"
			}
			// 调试
			PrintStructWithFieldNames(groupMsg)

			// Convert OnebotGroupMessage to map and send
			groupMsgMap := structToMap(groupMsg)
			//上报信息到onebotv11应用端(正反ws)
			p.BroadcastMessageToAll(groupMsgMap)

			// 转换appid
			AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

			// 储存和群号相关的eventid
			echo.AddEvnetID(AppIDString, GroupID64, data.ID)
		} else if data.UserOpenID != "" {
			//私聊回调
			newdata := ConvertInteractionToMessage(data)
			segmentedMessages := handlers.ConvertToSegmentedMessage(newdata)
			//映射str的messageID到int
			messageID64, err := idmap.StoreIDv2(data.ID)
			if err != nil {
				mylog.Printf("Error storing ID: %v", err)
				return nil
			}
			messageID := int(messageID64)
			var selfid64 int64
			if config.GetUseUin() {
				selfid64 = config.GetUinint64()
			} else {
				selfid64 = int64(p.Settings.AppID)
			}
			privateMsg := OnebotPrivateMessage{
				RawMessage:  data.Data.Resolved.ButtonData,
				Message:     segmentedMessages,
				MessageID:   messageID,
				MessageType: "private",
				PostType:    "message",
				SelfID:      selfid64,
				UserID:      userid64,
				Sender: PrivateSender{
					Nickname: "", //这个不支持,但加机器人好友,会收到一个事件,可以对应储存获取,用idmaps可以做到.
					UserID:   userid64,
				},
				SubType: "friend",
				Time:    time.Now().Unix(),
			}
			//增强配置
			if !config.GetNativeOb11() {
				privateMsg.RealMessageType = "interaction"
			}
			// 调试
			PrintStructWithFieldNames(privateMsg)

			// Convert OnebotGroupMessage to map and send
			privateMsgMap := structToMap(privateMsg)
			//上报信息到onebotv11应用端(正反ws)
			p.BroadcastMessageToAll(privateMsgMap)

			// 转换appid
			AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

			// 储存和用户ID相关的eventid
			echo.AddEvnetID(AppIDString, userid64, data.ID)
		} else {
			//频道回调
			// 处理onebot_channel_message逻辑
			newdata := ConvertInteractionToMessage(data)
			segmentedMessages := handlers.ConvertToSegmentedMessage(newdata)
			var selfid64 int64
			if config.GetUseUin() {
				selfid64 = config.GetUinint64()
			} else {
				selfid64 = int64(p.Settings.AppID)
			}
			onebotMsg := OnebotChannelMessage{
				ChannelID:   data.ChannelID,
				GuildID:     data.GuildID,
				Message:     segmentedMessages,
				RawMessage:  data.Data.Resolved.ButtonData,
				MessageID:   data.ID,
				MessageType: "guild",
				PostType:    "message",
				SelfID:      selfid64,
				UserID:      userid64,
				SelfTinyID:  "0",
				Sender: Sender{
					Nickname: "频道按钮回调",
					TinyID:   "0",
					UserID:   userid64,
					Card:     "频道按钮回调",
					Sex:      "0",
					Age:      0,
					Area:     "0",
					Level:    "0",
				},
				SubType: "channel",
				Time:    time.Now().Unix(),
				Avatar:  "",
			}
			//增强配置
			if !config.GetNativeOb11() {
				onebotMsg.RealMessageType = "interaction"
			}
			//调试
			PrintStructWithFieldNames(onebotMsg)

			// 将 onebotMsg 结构体转换为 map[string]interface{}
			msgMap := structToMap(onebotMsg)

			//上报信息到onebotv11应用端(正反ws)
			p.BroadcastMessageToAll(msgMap)

			// TODO: 实现eventid
		}
	}

	return nil
}

// ConvertInteractionToMessage 转换 Interaction 到 Message
func ConvertInteractionToMessage(interaction *dto.WSInteractionData) *dto.Message {
	var message dto.Message

	// 直接映射的字段
	message.ID = interaction.ID
	message.ChannelID = interaction.ChannelID
	message.GuildID = interaction.GuildID
	message.GroupID = interaction.GroupOpenID

	// 特殊处理的字段
	message.Content = interaction.Data.Resolved.ButtonData
	message.DirectMessage = interaction.ChatType == 2

	return &message
}

// 延迟执行PutInteraction
func DelayedPutInteraction(o openapi.OpenAPI, interactionID, fromuid, fromgid string) {
	key := fromuid
	if fromuid == "0" {
		key = fromgid
	}

	mu.Lock()
	delay := config.GetPutInteractionDelay()
	nextTime, exists := nextAllowedTime[key]
	if !exists || time.Now().After(nextTime) {
		nextAllowedTime[key] = time.Now().Add(time.Millisecond * time.Duration(delay))
		mu.Unlock()
		o.PutInteraction(context.TODO(), interactionID, `{"code": 0}`)
	} else {
		mu.Unlock()
		time.Sleep(time.Until(nextTime))
		DelayedPutInteraction(o, interactionID, fromuid, fromgid) // 重新尝试
	}
}
