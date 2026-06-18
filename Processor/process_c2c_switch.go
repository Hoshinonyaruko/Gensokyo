package Processor

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket/client"
)

// ProcessC2CMsgReject 处理用户拒绝机器人主动消息
func (p *Processors) ProcessC2CMsgReject(data *dto.WSC2CMsgRejectData) error {

	if data == nil {
		return fmt.Errorf("ProcessC2CMsgReject data is nil")
	}

	// 转换appid
	userid64 := int64(0)

	// 获取s
	s := client.GetGlobalS()
	// 转换appid
	AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

	// 获取当前时间的13位毫秒级时间戳
	currentTimeMillis := time.Now().UnixNano() / 1e6

	// 构造echostr，包括AppID，原始的s变量和当前时间戳
	echostr := fmt.Sprintf("%s_%d_%d", AppIDString, s, currentTimeMillis)

	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}

	// 配置检查：决定是发送 Notice 还是 Message
	// 注意：需要在 config 包中实现 GetGlobalC2CMsgSwitchEventToMessage (类似 GroupMsgRejectReciveEventToMessage)
	if !config.GetGlobalC2CMsgSwitchEventToMessage() {
		// --- 分支 A: 发送标准 Notice ---
		notice := &FriendNoticeEvent{ // 使用上一轮定义的 FriendNoticeEvent
			NoticeType: "c2c_msg_reject",
			PostType:   "notice",
			SelfID:     selfid64,
			Time:       time.Now().Unix(),
			UserID:     userid64,
			// RealUserID 稍后处理
		}

		// 增强配置
		if !config.GetNativeOb11() {
			notice.RealUserID = data.OpenID
		}

		// 调试
		// PrintStructWithFieldNames(notice)

		// Convert to map and send
		noticeMap := structToMap(notice)

		//上报信息到onebotv11应用端(正反ws)
		go p.BroadcastMessageToAll(noticeMap, p.Apiv2, data)

		// 储存 EventID (C2C 事件通常没有 GroupID，这里用 0 或 UserID 代替 GroupID 位)
		echo.AddEvnetID(AppIDString, 0, data.EventID)

	} else {
		// --- 分支 B: 伪装成 Message ---
		if data.OpenID != "" {
			// 转换数据
			newdata := ConvertC2CRejectToMessage(data)

			// 如果在Array模式下, 则处理Message为Segment格式
			var segmentedMessages interface{} = newdata.Content
			if config.GetArrayValue() {
				segmentedMessages = handlers.ConvertToSegmentedMessage(newdata)
			}

			// 平台事件,不是真实信息,无需messageID
			messageID64 := 123
			messageID := int(messageID64)

			// 构造 Private Message
			privateMsg := OnebotPrivateMessage{
				RawMessage:  newdata.Content,
				Message:     segmentedMessages,
				MessageID:   messageID,
				MessageType: "private", // C2C 对应 private
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

			// 增强配置
			if !config.GetNativeOb11() {
				privateMsg.RealMessageType = "c2c_msg_reject"
				privateMsg.IsBindedUserId = false
				privateMsg.RealUserID = data.OpenID
				privateMsg.Avatar, _ = GenerateAvatarURLV2(data.OpenID)
			}

			// 根据条件判断是否添加Echo字段
			if config.GetTwoWayEcho() {
				privateMsg.Echo = echostr
				//用向应用端(如果支持)发送echo,来确定客户端的send_msg对应的触发词原文
				echo.AddMsgIDv3(AppIDString, echostr, newdata.Content)
			}

			// 映射消息类型 (private)
			echo.AddMsgType(AppIDString, s, "private")

			// 调试
			// PrintStructWithFieldNames(privateMsg)

			// Convert and send
			privateMsgMap := structToMap(privateMsg)
			//上报信息到onebotv11应用端(正反ws)
			go p.BroadcastMessageToAll(privateMsgMap, p.Apiv2, data)

			// 储存 EventID
			fmt.Printf("测试:储存C2C eventid:[%v]\n", data.EventID)
			// 私聊通常不需要 LongGroupID，这里传 0 或者根据你的 echo 包实现传 UserID
			echo.AddEvnetID(AppIDString, 0, data.EventID)
		}
	}
	return nil
}

// ProcessC2CMsgReceive 处理用户开启机器人主动消息
func (p *Processors) ProcessC2CMsgReceive(data *dto.WSC2CMsgReceiveData) error {

	// 【修复1】保留了这里的 nil 检查，很好
	if data == nil {
		return fmt.Errorf("ProcessC2CMsgReject data is nil")
	}

	// 转换appid
	userid64 := int64(0)

	// 获取s
	s := client.GetGlobalS()
	// 转换appid
	AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

	// 获取当前时间的13位毫秒级时间戳
	currentTimeMillis := time.Now().UnixNano() / 1e6

	// 构造echostr
	echostr := fmt.Sprintf("%s_%d_%d", AppIDString, s, currentTimeMillis)

	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}

	// 配置检查
	if !config.GetGlobalC2CMsgSwitchEventToMessage() {
		// --- Branch A: Notice ---
		notice := &FriendNoticeEvent{
			NoticeType: "c2c_msg_receive",
			PostType:   "notice",
			SelfID:     selfid64,
			Time:       time.Now().Unix(),
			UserID:     userid64,
		}

		if !config.GetNativeOb11() {
			notice.RealUserID = data.OpenID
		}

		noticeMap := structToMap(notice)
		go p.BroadcastMessageToAll(noticeMap, p.Apiv2, data)
		echo.AddEvnetID(AppIDString, 0, data.EventID)

	} else {
		// --- Branch B: Message ---
		if data.OpenID != "" {
			newdata := ConvertC2CReceiveToMessage(data)

			var segmentedMessages interface{} = newdata.Content
			if config.GetArrayValue() {
				segmentedMessages = handlers.ConvertToSegmentedMessage(newdata)
			}

			messageID64 := 123
			messageID := int(messageID64)

			privateMsg := OnebotPrivateMessage{
				RawMessage:  newdata.Content,
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

			if !config.GetNativeOb11() {
				privateMsg.RealMessageType = "c2c_msg_receive"
				privateMsg.IsBindedUserId = false
				privateMsg.RealUserID = data.OpenID
				privateMsg.Avatar, _ = GenerateAvatarURLV2(data.OpenID)
			}

			if config.GetTwoWayEcho() {
				privateMsg.Echo = echostr
				echo.AddMsgIDv3(AppIDString, echostr, newdata.Content)
			}

			echo.AddMsgType(AppIDString, s, "private")

			privateMsgMap := structToMap(privateMsg)
			go p.BroadcastMessageToAll(privateMsgMap, p.Apiv2, data)

			fmt.Printf("测试:储存C2C eventid:[%v]\n", data.EventID)
			echo.AddEvnetID(AppIDString, 0, data.EventID)
		}
	}
	return nil
}

// ConvertC2CRejectToMessage 转换 C2C Reject 到 Message
func ConvertC2CRejectToMessage(r *dto.WSC2CMsgRejectData) *dto.Message {
	var message dto.Message

	// ✅ 修复：初始化 User 指针
	message.Author = &dto.User{
		ID: r.OpenID,
	}

	// 特殊处理的字段: 这里的文本需要从 config 获取
	// 请在 config 包中定义 GetGlobalC2CMsgRejectMessage()，例如返回 "关闭主动消息"
	message.Content = config.GetGlobalC2CMsgRejectMessage()
	message.DirectMessage = true // 标记为私信

	return &message
}

// ConvertC2CReceiveToMessage 转换 C2C Receive 到 Message
func ConvertC2CReceiveToMessage(r *dto.WSC2CMsgReceiveData) *dto.Message {
	var message dto.Message

	// ✅ 修复：初始化 User 指针
	message.Author = &dto.User{
		ID: r.OpenID,
	}

	// 请在 config 包中定义 GetGlobalC2CMsgReceiveMessage()，例如返回 "开启主动消息"
	message.Content = config.GetGlobalC2CMsgReceiveMessage()
	message.DirectMessage = true

	return &message
}
