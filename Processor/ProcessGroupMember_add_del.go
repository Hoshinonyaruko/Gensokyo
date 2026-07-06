// 处理收到的信息事件
package Processor

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket/client"
)

// ProcessGroupMemberAdd 将用户进群事件伪装成 String 类型的普通群消息
func (p *Processors) ProcessGroupMemberAdd(data *dto.WSGroupMemberAddData) error {
	// [核心] 格式化一个特殊的指令型文本，你的后端可以直接拦截这个文案
	messageText := "群成员增加"

	return p.dispatchSpoofedEventMessage(data.EventID, data.GroupOpenID, data.MemberOpenID, data.Timestamp, messageText, data)
}

// ProcessGroupMemberRemove 将用户退群事件伪装成 String 类型的普通群消息
func (p *Processors) ProcessGroupMemberRemove(data *dto.WSGroupMemberRemoveData) error {
	// [核心] 格式化退群文本
	messageText := "群成员减少"

	return p.dispatchSpoofedEventMessage(data.EventID, data.GroupOpenID, data.MemberOpenID, data.Timestamp, messageText, data)
}

// dispatchSpoofedEventMessage 内部通用方法：组装 OnebotGroupMessageS 并上报 (无整数转换开销)
func (p *Processors) dispatchSpoofedEventMessage(eventID, groupID, userID string, timestamp uint64, messageText string, rawData interface{}) error {
	s := client.GetGlobalS()
	AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
	currentTimeMillis := time.Now().UnixNano() / 1e6
	echostr := fmt.Sprintf("%s_%d_%d", AppIDString, s, currentTimeMillis)

	// 获取自身机器人 ID
	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}

	platform := "qq"
	if config.GetUnionID() {
		platform = "unionqq"
	}

	// 直接且仅构造 OnebotGroupMessageS
	groupMsgS := OnebotGroupMessageS{
		RawMessage:  messageText,
		Message:     messageText, // 纯文本模式
		MessageID:   eventID,     // 直接复用底层的 EventID
		GroupID:     groupID,
		MessageType: "group",
		PostType:    "message", // 伪装完成！后端会认为这是一条正常的消息
		SelfID:      selfid64,
		UserID:      userID,
		Sender: Sender{
			UserID: 0, // String 模式下无视 Int ID
			Sex:    "0",
			Role:   "member", // 进退群都默认算 member 即可
		},
		SubType:  "normal",
		Time:     int64(timestamp),
		Platform: platform,
		Font:     0,
	}

	// 增强配置注入
	if !config.GetNativeOb11() {
		groupMsgS.RealMessageType = "group" // 依然伪装成 group 消息
		groupMsgS.RealGroupID = groupID
		groupMsgS.RealUserID = userID
		groupMsgS.Avatar, _ = GenerateAvatarURLV2(userID)
	}

	// Echo 逻辑保留，方便后端对账
	if config.GetTwoWayEcho() {
		groupMsgS.Echo = echostr
		echo.AddMsgIDv3(AppIDString, echostr, messageText)
	}

	echo.AddMsgID(AppIDString, s, eventID)
	echo.AddMsgType(AppIDString, s, "group")

	// 成员事件的 ID 不是普通消息 msg_id，不能塞进 lazy message_id。
	// 它应该作为 event_id 存起来，供 send_group_msg 的 messageID=="2000" 分支取用。
	echo.AddEvnetIDv2(AppIDString, groupID, eventID)

	// 打印与上报到所有后端实例
	PrintStructWithFieldNames(groupMsgS)
	groupMsgMap := structToMap(groupMsgS)

	// 使用 FAF 式的高速广播，丢给 MUD 引擎处理
	go p.BroadcastMessageToAll(groupMsgMap, p.Apiv2, rawData)

	return nil
}
