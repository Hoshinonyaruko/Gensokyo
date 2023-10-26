// 处理收到的信息事件
package Processor

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket/client"
)

// ProcessGroupMessage 处理群组消息
func (p *Processor) ProcessGroupMessage(data *dto.WSGroupATMessageData) error {
	// 获取s
	s := client.GetGlobalS()

	idmap.WriteConfigv2(data.ChannelID, "guild_id", data.GuildID)

	// 转换at
	messageText := handlers.RevertTransformedText(data.Content)

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
		log.Printf("Error storing ID: %v", err)
		return nil
	}

	//映射str的messageID到int
	messageID64, err := idmap.StoreIDv2(data.ID)
	if err != nil {
		log.Printf("Error storing ID: %v", err)
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
		Echo:    echostr,
	}

	// 将当前s和appid和message进行映射
	echo.AddMsgID(AppIDString, s, data.ID)
	echo.AddMsgType(AppIDString, s, "group")
	//为不支持双向echo的ob服务端映射
	echo.AddMsgID(AppIDString, GroupID64, data.ID)
	echo.AddMsgType(AppIDString, GroupID64, "group")
	//储存当前群或频道号的类型
	idmap.WriteConfigv2(data.GroupID, "type", "group")

	// 调试
	PrintStructWithFieldNames(groupMsg)

	// Convert OnebotGroupMessage to map and send
	groupMsgMap := structToMap(groupMsg)
	err = p.Wsclient.SendMessage(groupMsgMap)
	if err != nil {
		return fmt.Errorf("error sending group message via wsclient: %v", err)
	}

	return nil
}
