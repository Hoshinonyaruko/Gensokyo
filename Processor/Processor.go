// 处理收到的信息事件
package Processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/wsclient"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

// Processor 结构体用于处理消息
type Processors struct {
	Api             openapi.OpenAPI                   // API 类型
	Apiv2           openapi.OpenAPI                   //群的API
	Settings        *config.Settings                  // 使用指针
	Wsclient        []*wsclient.WebSocketClient       // 指针的切片
	WsServerClients []callapi.WebSocketServerClienter //ws server被连接的客户端
}

type Sender struct {
	Nickname string `json:"nickname"`
	TinyID   string `json:"tiny_id"`
	UserID   int64  `json:"user_id"`
	Role     string `json:"role,omitempty"`
}

// 频道信息事件
type OnebotChannelMessage struct {
	ChannelID   string      `json:"channel_id"`
	GuildID     string      `json:"guild_id"`
	Message     interface{} `json:"message"`
	MessageID   string      `json:"message_id"`
	MessageType string      `json:"message_type"`
	PostType    string      `json:"post_type"`
	SelfID      int64       `json:"self_id"`
	SelfTinyID  string      `json:"self_tiny_id"`
	Sender      Sender      `json:"sender"`
	SubType     string      `json:"sub_type"`
	Time        int64       `json:"time"`
	Avatar      string      `json:"avatar,omitempty"`
	UserID      int64       `json:"user_id"`
	RawMessage  string      `json:"raw_message"`
	Echo        string      `json:"echo,omitempty"`
}

// 群信息事件
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
	Avatar      string      `json:"avatar,omitempty"`
	Echo        string      `json:"echo,omitempty"`
	Message     interface{} `json:"message"` // For array format
	MessageSeq  int         `json:"message_seq"`
	Font        int         `json:"font"`
	UserID      int64       `json:"user_id"`
}

// 私聊信息事件
type OnebotPrivateMessage struct {
	RawMessage  string        `json:"raw_message"`
	MessageID   int           `json:"message_id"` // Can be either string or int depending on logic
	MessageType string        `json:"message_type"`
	PostType    string        `json:"post_type"`
	SelfID      int64         `json:"self_id"` // Can be either string or int depending on logic
	Sender      PrivateSender `json:"sender"`
	SubType     string        `json:"sub_type"`
	Time        int64         `json:"time"`
	Avatar      string        `json:"avatar,omitempty"`
	Echo        string        `json:"echo,omitempty"`
	Message     interface{}   `json:"message"`     // For array format
	MessageSeq  int           `json:"message_seq"` // Optional field
	Font        int           `json:"font"`        // Optional field
	UserID      int64         `json:"user_id"`     // Can be either string or int depending on logic
}

type PrivateSender struct {
	Nickname string `json:"nickname"`
	UserID   int64  `json:"user_id"` // Can be either string or int depending on logic
}

func FoxTimestamp() int64 {
	return time.Now().Unix()
}

// ProcessInlineSearch 处理内联查询
func (p *Processors) ProcessInlineSearch(data *dto.WSInteractionData) error {
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

//return nil

//下面是测试时候固定代码
//发私信给机器人4条机器人不回,就不能继续发了

// timestamp := time.Now().Unix() // 获取当前时间的int64类型的Unix时间戳
// timestampStr := fmt.Sprintf("%d", timestamp)

// dm := &dto.DirectMessage{
// 	GuildID:    GuildID,
// 	ChannelID:  ChannelID,
// 	CreateTime: timestampStr,
// }

// PrintStructWithFieldNames(dm)

// // 发送默认回复
// toCreate := &dto.MessageToCreate{
// 	Content: "默认私信回复",
// 	MsgID:   data.ID,
// }
// _, err = p.Api.PostDirectMessage(
// 	context.Background(), dm, toCreate,
// )
// if err != nil {
// 	mylog.Println("Error sending default reply:", err)
// 	return nil
// }

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
		mylog.Println("Input is not a struct")
		return
	}

	// 迭代所有的字段并打印字段名和值
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)
		mylog.Printf("%s: %v\n", field.Name, value.Interface())
	}
}

// 将结构体转换为 map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	json.Unmarshal(j, &out)
	return out
}

// 修改函数的返回类型为 *Processor
func NewProcessor(api openapi.OpenAPI, apiv2 openapi.OpenAPI, settings *config.Settings, wsclient []*wsclient.WebSocketClient) *Processors {
	return &Processors{
		Api:      api,
		Apiv2:    apiv2,
		Settings: settings,
		Wsclient: wsclient,
	}
}

// 修改函数的返回类型为 *Processor
func NewProcessorV2(api openapi.OpenAPI, apiv2 openapi.OpenAPI, settings *config.Settings) *Processors {
	return &Processors{
		Api:      api,
		Apiv2:    apiv2,
		Settings: settings,
	}
}

// 发信息给所有连接正向ws的客户端
func (p *Processors) SendMessageToAllClients(message map[string]interface{}) error {
	var result *multierror.Error

	for _, client := range p.WsServerClients {
		// 使用接口的方法
		err := client.SendMessage(message)
		if err != nil {
			// Append the error to our result
			result = multierror.Append(result, fmt.Errorf("failed to send to client: %w", err))
		}
	}

	// This will return nil if no errors were added
	return result.ErrorOrNil()
}

// 方便快捷的发信息函数
func (p *Processors) BroadcastMessageToAll(message map[string]interface{}) error {
	var errors []string

	// 发送到我们作为客户端的Wsclient
	for _, client := range p.Wsclient {
		err := client.SendMessage(message)
		if err != nil {
			errors = append(errors, fmt.Sprintf("error sending private message via wsclient: %v", err))
		}
	}

	// 发送到我们作为服务器连接到我们的WsServerClients
	for _, serverClient := range p.WsServerClients {
		err := serverClient.SendMessage(message)
		if err != nil {
			errors = append(errors, fmt.Sprintf("error sending private message via WsServerClient: %v", err))
		}
	}

	// 在循环结束后处理记录的错误
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

func (p *Processors) HandleFrameworkCommand(messageText string, data interface{}, Type string) error {
	// 正则表达式匹配转换后的 CQ 码
	cqRegex := regexp.MustCompile(`\[CQ:at,qq=\d+\]`)

	// 使用正则表达式替换所有的 CQ 码为 ""
	cleanedMessage := cqRegex.ReplaceAllString(messageText, "")

	// 去除字符串前后的空格
	cleanedMessage = strings.TrimSpace(cleanedMessage)
	var err error
	var now, new string
	var realid string
	if specificData, ok := data.(*dto.WSMessageData); ok {
		realid = specificData.Author.ID
	}
	// 获取MasterID数组
	masterIDs := config.GetMasterID()
	// 根据realid获取new
	now, new, err = idmap.RetrieveVirtualValue(realid)
	// 检查真实值或虚拟值是否在数组中
	realValueIncluded := contains(masterIDs, realid)
	virtualValueIncluded := contains(masterIDs, new)

	// me指令处理逻辑
	if strings.HasPrefix(cleanedMessage, config.GetMePrefix()) {
		if err != nil {
			// 发送错误信息
			SendMessage(err.Error(), data, Type, p.Api, p.Apiv2)
			return err
		}

		// 发送成功信息
		SendMessage("目前状态:\n当前真实值 "+now+"\n当前虚拟值 "+new+"\n"+config.GetBindPrefix()+" 当前虚拟值"+" 目标虚拟值", data, Type, p.Api, p.Apiv2)
		return nil
	}

	if realValueIncluded || virtualValueIncluded {
		// bind指令处理逻辑
		if strings.HasPrefix(cleanedMessage, config.GetBindPrefix()) {
			// 分割指令以获取参数
			parts := strings.Fields(cleanedMessage)
			if len(parts) != 3 {
				mylog.Printf("bind指令参数错误\n正确的格式" + config.GetBindPrefix() + " 当前虚拟值 新虚拟值")
				return nil
			}

			// 将字符串转换为 int64
			oldRowValue, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return err
			}

			newRowValue, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return err
			}

			// 调用 UpdateVirtualValue
			err = idmap.UpdateVirtualValue(oldRowValue, newRowValue)
			if err != nil {
				SendMessage(err.Error(), data, Type, p.Api, p.Apiv2)
				return err
			}
			now, new, err := idmap.RetrieveRealValue(newRowValue)
			if err != nil {
				SendMessage(err.Error(), data, Type, p.Api, p.Apiv2)
			} else {
				SendMessage("绑定成功,目前状态:\n当前真实值 "+now+"\n当前虚拟值 "+new, data, Type, p.Api, p.Apiv2)
			}

		}

		return nil
	} else {
		if strings.HasPrefix(cleanedMessage, config.GetBindPrefix()) {
			mylog.Printf("您没有权限,请设置master_id,发送/me 获取虚拟值或真实值填入其中")
			SendMessage("您没有权限,请设置master_id,发送/me 获取虚拟值或真实值填入其中", data, Type, p.Api, p.Apiv2)
		}
		return nil
	}
}

// contains 检查数组中是否包含指定的字符串
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// SendMessage 发送消息根据不同的类型
func SendMessage(messageText string, data interface{}, messageType string, api openapi.OpenAPI, apiv2 openapi.OpenAPI) error {
	// 强制类型转换，获取Message结构
	var msg *dto.Message
	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSDirectMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSC2CMessageData:
		msg = (*dto.Message)(v)
	default:
		return nil
	}
	switch messageType {
	case "guild":
		// 处理公会消息
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText)
		if _, err := api.PostMessage(context.TODO(), msg.ChannelID, textMsg); err != nil {
			mylog.Printf("发送文本信息失败: %v", err)
			return err
		}

	case "group":
		// 处理群组消息
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText)
		_, err := apiv2.PostGroupMessage(context.TODO(), msg.GroupID, textMsg)
		if err != nil {
			mylog.Printf("发送文本群组信息失败: %v", err)
			return err
		}

	case "guild_private":
		// 处理私信
		timestamp := time.Now().Unix()
		timestampStr := fmt.Sprintf("%d", timestamp)
		dm := &dto.DirectMessage{
			GuildID:    msg.GuildID,
			ChannelID:  msg.ChannelID,
			CreateTime: timestampStr,
		}
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText)
		if _, err := apiv2.PostDirectMessage(context.TODO(), dm, textMsg); err != nil {
			mylog.Printf("发送文本信息失败: %v", err)
			return err
		}

	case "group_private":
		// 处理群组私聊消息
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText)
		_, err := apiv2.PostC2CMessage(context.TODO(), msg.Author.ID, textMsg)
		if err != nil {
			mylog.Printf("发送文本私聊信息失败: %v", err)
			return err
		}

	default:
		return errors.New("未知的消息类型")
	}

	return nil
}
