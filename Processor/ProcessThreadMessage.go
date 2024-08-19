// 处理收到的帖子信息事件
package Processor

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket/client"
)

// ProcessInlineSearch 处理帖子事件
func (p *Processors) ProcessThreadMessage(data *dto.WSThreadData) error {
	// 过滤，仅当ID以"FORUM_THREAD_CREATE"开头时继续执行 后期再改
	if !strings.HasPrefix(data.ID, "FORUM_THREAD_CREATE") {
		return nil
	}
	if !p.Settings.GlobalForumToChannel {
		//原始帖子类型
		// 将时间字符串转换为时间戳
		t, err := time.Parse(time.RFC3339, string(data.ThreadInfo.DateTime))
		if err != nil {
			return fmt.Errorf("error parsing time: %v", err)
		}
		//获取s
		s := client.GetGlobalS()
		//转换at
		//帖子没有at
		//框架内指令
		//帖子不需要
		//转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
		// 获取当前时间的13位毫秒级时间戳
		currentTimeMillis := time.Now().UnixNano() / 1e6
		// 构造echostr，包括AppID，原始的s变量和当前时间戳
		echostr := fmt.Sprintf("%s_%d_%d", AppIDString, s, currentTimeMillis)
		//映射str的userid到int
		userid64, err := idmap.StoreIDv2(data.AuthorID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		// 如果在Array模式下, 则处理Message为Segment格式
		var segmentedMessages interface{} = data.ThreadInfo.Content
		if config.GetArrayValue() {
			segmentedMessages = handlers.ConvertToSegmentedMessage(data)
		}
		messageText, err := parseContent(data.ThreadInfo.Content)
		if err != nil {
			mylog.Printf("Error parseContent Forum: %v", err)
		}

		var selfid64 int64
		if config.GetUseUin() {
			selfid64 = config.GetUinint64()
		} else {
			selfid64 = int64(p.Settings.AppID)
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
			SelfID:      selfid64,
			UserID:      userid64,
			SelfTinyID:  "0",
			Sender: Sender{
				Nickname: "发帖人",
				TinyID:   "0",
				UserID:   userid64,
				Card:     "发帖人昵称",
				Sex:      "0",
				Age:      0,
				Area:     "0",
				Level:    "0",
			},
			SubType: "forum",
			Time:    t.Unix(),
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
		echo.AddMsgType(AppIDString, s, "forum")
		//为不支持双向echo的ob11服务端映射
		echo.AddMsgID(AppIDString, userid64, data.ID)
		//映射类型
		echo.AddMsgType(AppIDString, userid64, "forum")
		//储存当前群或频道号的类型
		idmap.WriteConfigv2(data.ChannelID, "type", "forum")
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
		go p.BroadcastMessageToAll(msgMap, p.Apiv2, data)

		return nil
	} else {
		//转换为频道或者群
		if !p.Settings.GlobalChannelToGroup {
			//转化为频道信息
			// 将时间字符串转换为时间戳
			t, err := time.Parse(time.RFC3339, string(data.ThreadInfo.DateTime))
			if err != nil {
				return fmt.Errorf("error parsing time: %v", err)
			}
			//获取s
			s := client.GetGlobalS()
			//转换at
			//帖子没有at
			//框架内指令
			//帖子不需要
			//转换appid
			AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
			// 获取当前时间的13位毫秒级时间戳
			currentTimeMillis := time.Now().UnixNano() / 1e6
			// 构造echostr，包括AppID，原始的s变量和当前时间戳
			echostr := fmt.Sprintf("%s_%d_%d", AppIDString, s, currentTimeMillis)
			//映射str的userid到int
			userid64, err := idmap.StoreIDv2(data.AuthorID)
			if err != nil {
				mylog.Printf("Error storing ID: %v", err)
				return nil
			}
			// 如果在Array模式下, 则处理Message为Segment格式
			var segmentedMessages interface{} = data.ThreadInfo.Content
			if config.GetArrayValue() {
				segmentedMessages = handlers.ConvertToSegmentedMessage(data)
			}
			messageText, err := parseContent(data.ThreadInfo.Content)
			if err != nil {
				mylog.Printf("Error parseContent Forum: %v", err)
			}

			var selfid64 int64
			if config.GetUseUin() {
				selfid64 = config.GetUinint64()
			} else {
				selfid64 = int64(p.Settings.AppID)
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
				SelfID:      selfid64,
				UserID:      userid64,
				SelfTinyID:  "0",
				Sender: Sender{
					Nickname: "发帖人",
					TinyID:   "0",
					UserID:   userid64,
					Card:     "发帖人昵称",
					Sex:      "0",
					Age:      0,
					Area:     "0",
					Level:    "0",
				},
				SubType: "channel",
				Time:    t.Unix(),
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
			echo.AddMsgType(AppIDString, s, "forum")
			//为不支持双向echo的ob11服务端映射
			echo.AddMsgID(AppIDString, userid64, data.ID)
			//映射类型
			echo.AddMsgType(AppIDString, userid64, "forum")
			//储存当前群或频道号的类型
			idmap.WriteConfigv2(data.ChannelID, "type", "forum")
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
			go p.BroadcastMessageToAll(msgMap, p.Apiv2, data)
		} else {
			//转化为群信息
			//将频道转化为一个群
			//获取s
			AppIDString := strconv.FormatUint(p.Settings.AppID, 10)
			s := client.GetGlobalS()
			var userid64 int64
			var ChannelID64 int64
			var err error
			if config.GetIdmapPro() {
				//将真实id转为int userid64
				ChannelID64, userid64, err = idmap.StoreIDv2Pro(data.ChannelID, data.AuthorID)
				if err != nil {
					mylog.Errorf("Error storing ID: %v", err)
				}
				//当参数不全时
				_, _ = idmap.StoreIDv2(data.ChannelID)
				_, _ = idmap.StoreIDv2(data.AuthorID)
				if !config.GetHashIDValue() {
					mylog.Fatalf("避坑日志:你开启了高级id转换,请设置hash_id为true,并且删除idmaps并重启")
				}
				//补救措施
				idmap.SimplifiedStoreID(data.AuthorID)
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
				userid64, err = idmap.StoreIDv2(data.AuthorID)
				if err != nil {
					mylog.Printf("Error storing ID: %v", err)
					return nil
				}
			}
			//转成int再互转
			idmap.WriteConfigv2(fmt.Sprint(ChannelID64), "guild_id", data.GuildID)
			//储存原来的(获取群列表需要)
			idmap.WriteConfigv2(data.ChannelID, "guild_id", data.GuildID)

			messageText, err := parseContent(data.ThreadInfo.Content)
			if err != nil {
				mylog.Printf("Error parseContent Forum: %v", err)
			}
			// 获取当前时间的13位毫秒级时间戳
			currentTimeMillis := time.Now().UnixNano() / 1e6
			// 构造echostr，包括AppID，原始的s变量和当前时间戳
			echostr := fmt.Sprintf("%s_%d_%d", AppIDString, s, currentTimeMillis)
			//映射str的messageID到int
			var messageID64 int64
			if config.GetMemoryMsgid() {
				messageID64, err = echo.StoreCacheInMemory(data.ID)
				if err != nil {
					log.Fatalf("Error storing ID: %v", err)
				}
			} else {
				messageID64, err = idmap.StoreCachev2(data.ID)
				if err != nil {
					log.Fatalf("Error storing ID: %v", err)
				}
			}
			messageID := int(messageID64)
			// 如果在Array模式下, 则处理Message为Segment格式
			var segmentedMessages interface{} = messageText
			if config.GetArrayValue() {
				segmentedMessages = handlers.ConvertToSegmentedMessage(data)
			}
			var IsBindedUserId, IsBindedGroupId bool
			if config.GetHashIDValue() {
				IsBindedUserId = idmap.CheckValue(data.AuthorID, userid64)
				IsBindedGroupId = idmap.CheckValue(data.ChannelID, ChannelID64)
			} else {
				IsBindedUserId = idmap.CheckValuev2(userid64)
				IsBindedGroupId = idmap.CheckValuev2(ChannelID64)
			}
			var selfid64 int64
			if config.GetUseUin() {
				selfid64 = config.GetUinint64()
			} else {
				selfid64 = int64(p.Settings.AppID)
			}
			groupMsg := OnebotGroupMessage{
				RawMessage:  messageText,
				Message:     segmentedMessages,
				MessageID:   messageID,
				GroupID:     ChannelID64,
				MessageType: "group",
				PostType:    "message",
				SelfID:      selfid64,
				UserID:      userid64,
				Sender: Sender{
					Nickname: "发帖人昵称",
					UserID:   userid64,
					Card:     "发帖人昵称",
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
				groupMsg.RealMessageType = "forum"
				groupMsg.IsBindedUserId = IsBindedUserId
				groupMsg.IsBindedGroupId = IsBindedGroupId
			}
			// 根据条件判断是否添加Echo字段
			if config.GetTwoWayEcho() {
				groupMsg.Echo = echostr
				//用向应用端(如果支持)发送echo,来确定客户端的send_msg对应的触发词原文
				echo.AddMsgIDv3(AppIDString, echostr, messageText)
			}

			//将当前s和appid和message进行映射
			echo.AddMsgID(AppIDString, s, data.ID)
			echo.AddMsgType(AppIDString, s, "forum")
			//为不支持双向echo的ob服务端映射
			echo.AddMsgID(AppIDString, ChannelID64, data.ID)
			//将当前的userid和groupid和msgid进行一个更稳妥的映射
			echo.AddMsgIDv2(AppIDString, ChannelID64, userid64, data.ID)
			//储存当前群或频道号的类型
			idmap.WriteConfigv2(fmt.Sprint(ChannelID64), "type", "forum")
			echo.AddMsgType(AppIDString, ChannelID64, "forum")
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
			go p.BroadcastMessageToAll(groupMsgMap, p.Apiv2, data)

		}
	}
	return nil
}

// UnmarshalForumContentElements 动态解析元素类型
func UnmarshalForumContentElements(content string) ([]dto.ForumContentElement, error) {
	var contentStructure dto.ForumContentStructure
	err := json.Unmarshal([]byte(content), &contentStructure)
	if err != nil {
		return nil, err
	}

	var elements []dto.ForumContentElement
	for _, paragraph := range contentStructure.Paragraphs {
		for _, rawElem := range paragraph.Elems {
			var base struct {
				Type int `json:"type"`
			}
			err := json.Unmarshal(rawElem, &base)
			if err != nil {
				fmt.Println("Error determining element type:", err)
				continue
			}

			switch base.Type {
			case 1: // 文本元素
				var textElem dto.ForumTextElement
				err := json.Unmarshal(rawElem, &textElem)
				if err != nil {
					fmt.Println("Error unmarshalling text element:", err)
					continue
				}
				elements = append(elements, textElem)
			case 3: // URL元素
				var urlElem dto.ForumURLElement
				err := json.Unmarshal(rawElem, &urlElem)
				if err != nil {
					fmt.Println("Error unmarshalling URL element:", err)
					continue
				}
				elements = append(elements, urlElem)
			case 5: // 频道元素
				var channelElem dto.ForumChannelElement
				err := json.Unmarshal(rawElem, &channelElem)
				if err != nil {
					fmt.Println("Error unmarshalling channel element:", err)
					continue
				}
				elements = append(elements, channelElem)
			default:
				fmt.Printf("Unknown type: %d\n", base.Type)
			}
		}
	}

	return elements, nil
}

// parseContent 解析字符串content，并生成消息文本
func parseContent(content string) (string, error) {
	elements, err := UnmarshalForumContentElements(content)
	if err != nil {
		return "", err
	}
	mylog.Printf("测试:%v", elements)

	var messageTextBuilder strings.Builder

	// 计算是不是只有一个ForumTextElement而且没有其他元素
	onlyOneTextElement := len(elements) == 1
	if onlyOneTextElement {
		_, onlyOneTextElement = elements[0].(dto.ForumTextElement)
	}

	for _, element := range elements {
		switch e := element.(type) {
		case dto.ForumTextElement:
			messageTextBuilder.WriteString(e.TextInfo.Text)
			if !onlyOneTextElement {
				messageTextBuilder.WriteString("\n") // 如果不是只有一个ForumTextElement，加换行符
			}
		case dto.ForumURLElement:
			if e.URLInfo.DisplayText != "" {
				messageTextBuilder.WriteString(e.URLInfo.DisplayText + ": ")
			}
			messageTextBuilder.WriteString(e.URLInfo.URL)
			messageTextBuilder.WriteString("\n")
		case *dto.ForumChannelElement:
			// 频道元素被忽略
		}
	}

	return messageTextBuilder.String(), nil
}
