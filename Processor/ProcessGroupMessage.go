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

// ProcessGroupMessage 处理群组消息
func (p *Processors) ProcessGroupMessage(data *dto.WSGroupATMessageData) error {
	// 获取s
	s := client.GetGlobalS()

	// 转换at
	messageText := handlers.RevertTransformedText(data)
	if messageText == "" {
		mylog.Printf("信息被自定义黑白名单拦截")
		return nil
	}

	// 转换appid
	AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

	// 构造echo
	echostr := AppIDString + "_" + strconv.FormatInt(s, 10)

	// 映射str的GroupID到int
	GroupID64, err := idmap.StoreIDv2(data.GroupID)
	if err != nil {
		return fmt.Errorf("failed to convert ChannelID to int: %v", err)
	}

	// 映射str的userid到int
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
	// 如果在Array模式下, 则处理Message为Segment格式
	var segmentedMessages interface{} = messageText
	if config.GetArrayValue() {
		segmentedMessages = handlers.ConvertToSegmentedMessage(data)
	}
	groupMsg := OnebotGroupMessage{
		RawMessage:  messageText,
		Message:     segmentedMessages,
		MessageID:   messageID,
		GroupID:     GroupID64,
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
	// 将当前s和appid和message进行映射
	echo.AddMsgID(AppIDString, s, data.ID)
	echo.AddMsgType(AppIDString, s, "group")
	//为不支持双向echo的ob服务端映射
	echo.AddMsgID(AppIDString, GroupID64, data.ID)
	echo.AddMsgType(AppIDString, GroupID64, "group")
	//储存当前群或频道号的类型
	idmap.WriteConfigv2(fmt.Sprint(GroupID64), "type", "group")
	echo.AddMsgType(AppIDString, GroupID64, "group")
	//懒message_id池
	echo.AddLazyMessageId(strconv.FormatInt(GroupID64, 10), data.ID, time.Now())
	// 调试
	PrintStructWithFieldNames(groupMsg)

	// Convert OnebotGroupMessage to map and send
	groupMsgMap := structToMap(groupMsg)
	//上报信息到onebotv11应用端(正反ws)
	p.BroadcastMessageToAll(groupMsgMap)
	return nil
}
