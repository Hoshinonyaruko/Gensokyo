// 处理收到的信息事件
package Processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/wsclient"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/websocket/client"
)

var compatibilityMapping *idmap.IniMapping
var err error

func init() {
	compatibilityMapping, err = idmap.NewIniMapping()
	if err != nil {
		log.Fatalf("Failed to initialize IniMapping: %v", err)
	}
}

// Processor 结构体用于处理消息
type Processor struct {
	Api      openapi.OpenAPI           // API 类型
	Apiv2    openapi.OpenAPI           //群的API
	Settings *config.Settings          // 使用指针
	Wsclient *wsclient.WebSocketClient // 使用指针
}

type Sender struct {
	Nickname string `json:"nickname"`
	TinyID   string `json:"tiny_id"`
	UserID   int64  `json:"user_id"`
}

type OnebotChannelMessage struct {
	ChannelID   string `json:"channel_id"`
	GuildID     string `json:"guild_id"`
	Message     string `json:"message"`
	MessageID   string `json:"message_id"`
	MessageType string `json:"message_type"`
	PostType    string `json:"post_type"`
	SelfID      int64  `json:"self_id"`
	SelfTinyID  string `json:"self_tiny_id"`
	Sender      Sender `json:"sender"`
	SubType     string `json:"sub_type"`
	Time        int64  `json:"time"`
	Avatar      string `json:"avatar"`
	UserID      int64  `json:"user_id"`
	RawMessage  string `json:"raw_message"`
	Echo        string `json:"echo"`
}

// OnebotGroupMessage represents the message structure for group messages.
type OnebotGroupMessage struct {
	RawMessage  string      `json:"raw_message"`
	MessageID   int         `json:"message_id"`
	GroupID     int64       `json:"group_id"` // Can be either string or int depending on p.Settings.CompleteFields
	MessageType string      `json:"message_type"`
	PostType    string      `json:"post_type"`
	SelfID      int64       `json:"self_id"` // Can be either string or int
	Sender      Sender      `json:"sender"`
	SubType     string      `json:"sub_type"`
	Time        int64       `json:"time"`
	Avatar      string      `json:"avatar"`
	Echo        string      `json:"echo"`
	Message     interface{} `json:"message"` // For array format
	MessageSeq  int         `json:"message_seq"`
	Font        int         `json:"font"`
	UserID      int64       `json:"user_id"`
}

func FoxTimestamp() int64 {
	return time.Now().Unix()
}

// ProcessGuildATMessage 处理消息，执行逻辑并可能使用 api 发送响应
func (p *Processor) ProcessGuildATMessage(data *dto.WSATMessageData) error {
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
		//获取s
		s := client.GetGlobalS()
		compatibilityMapping.WriteConfig(data.ChannelID, "guild_id", data.GuildID)
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

// ProcessInlineSearch 处理内联查询
func (p *Processor) ProcessInlineSearch(data *dto.WSInteractionData) error {
	//ctx := context.Background() // 或从更高级别传递一个上下文

	// 在这里处理内联查询
	// 这可能涉及解析查询、调用某些API、获取结果并格式化为响应
	// ...

	// 示例：发送响应
	// response := "Received your interaction!"            // 创建响应消息
	// err := p.api.PostInteractionResponse(ctx, response) // 替换为您的OpenAPI方法
	// if err != nil {
	// 	return err
	// }

	return nil
}

// ProcessGroupMessage 处理群组消息
func (p *Processor) ProcessGroupMessage(data *dto.WSGroupATMessageData) error {
	// 获取s
	s := client.GetGlobalS()

	compatibilityMapping.WriteConfig(data.ChannelID, "guild_id", data.GuildID)

	// 转换at
	messageText := handlers.RevertTransformedText(data.Content)

	// 转换appid
	AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

	// 构造echo
	echostr := AppIDString + "_" + strconv.FormatInt(s, 10)

	// 映射str的GroupID到int
	GroupID64, err := idmap.StoreID(data.GroupID)
	if err != nil {
		return fmt.Errorf("failed to convert ChannelID to int: %v", err)
	}

	// 映射str的userid到int
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
	// todo 判断array模式 然后对Message处理成array格式
	groupMsg := OnebotGroupMessage{
		RawMessage:  messageText,
		Message:     messageText,
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

// ProcessChannelDirectMessage 处理频道私信消息 这里我们是被动收到
func (p *Processor) ProcessChannelDirectMessage(data *dto.WSDirectMessageData) error {
	// 打印data结构体
	//PrintStructWithFieldNames(data)

	// 从私信中提取必要的信息
	//recipientID := data.Author.ID
	ChannelID := data.ChannelID
	//sourece是源头频道
	GuildID := data.GuildID

	// 创建私信通道 发主动私信才需要创建
	// dm, err := p.Api.CreateDirectMessage(
	// 	context.Background(), &dto.DirectMessageToCreate{
	// 		SourceGuildID: sourceGuildID,
	// 		RecipientID:   recipientID,
	// 	},
	// )
	// if err != nil {
	// 	log.Println("Error creating direct message channel:", err)
	// 	return nil
	// }

	timestamp := time.Now().Unix() // 获取当前时间的int64类型的Unix时间戳
	timestampStr := fmt.Sprintf("%d", timestamp)

	dm := &dto.DirectMessage{
		GuildID:    GuildID,
		ChannelID:  ChannelID,
		CreateTime: timestampStr,
	}

	PrintStructWithFieldNames(dm)

	// 发送默认回复
	toCreate := &dto.MessageToCreate{
		Content: "默认私信回复",
		MsgID:   data.ID,
	}
	_, err = p.Api.PostDirectMessage(
		context.Background(), dm, toCreate,
	)
	if err != nil {
		log.Println("Error sending default reply:", err)
		return nil
	}

	return nil
}

// ProcessC2CMessage 处理C2C消息 群私聊
func (p *Processor) ProcessC2CMessage(rawMessage string, data *dto.WSC2CMessageData) error {
	// ctx := context.Background() // 或从更高级别传递一个上下文

	// // 在这里处理C2C消息
	// // ...

	// // 示例：直接回复收到的消息
	// response := fmt.Sprintf("Received your message: %s", rawMessage) // 创建响应消息
	// err := p.api.PostC2CMessage(ctx, response)                       // 替换为您的OpenAPI方法
	// if err != nil {
	// 	return err
	// }

	return nil
}

// 打印结构体的函数
func PrintStructWithFieldNames(v interface{}) {
	val := reflect.ValueOf(v)

	// 如果是指针，获取其指向的元素
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	// 确保我们传入的是一个结构体
	if typ.Kind() != reflect.Struct {
		fmt.Println("Input is not a struct")
		return
	}

	// 迭代所有的字段并打印字段名和值
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)
		fmt.Printf("%s: %v\n", field.Name, value.Interface())
	}
}

// 将结构体转换为 map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	json.Unmarshal(j, &out)
	return out
}
