package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/structs"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

var msgIDToIndex sync.Map
var msgIDToRelatedID sync.Map

func init() {
	callapi.RegisterHandler("send_private_msg_sse", HandleSendPrivateMsgSSE)
}

func incrementIndex(msgID string) int {
	actual, loaded := msgIDToIndex.LoadOrStore(msgID, 0)
	if !loaded {
		return 0
	}
	newVal := actual.(int) + 1
	msgIDToIndex.Store(msgID, newVal)
	return newVal
}

func UpdateRelatedID(MessageID, ID string) {
	msgIDToRelatedID.Store(MessageID, ID)
}

func GetRelatedID(MessageID string) string {
	if relatedID, ok := msgIDToRelatedID.Load(MessageID); ok {
		return relatedID.(string)
	}
	return ""
}

func HandleSendPrivateMsgSSE(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 使用 message.Echo 作为key来获取消息类型
	var retmsg string

	// 检查UserID是否为0
	checkZeroUserID := func(id interface{}) bool {
		switch v := id.(type) {
		case int:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != "0" // 同样检查字符串形式的0
		default:
			return true // 如果不是int、int64或string，假定它不为0
		}
	}

	// New checks for UserID and GroupID being nil or 0
	if message.Params.UserID == nil || !checkZeroUserID(message.Params.UserID) {
		mylog.Printf("send_group_msg_sse接收到错误action: %v", message)
		return "", nil
	}

	var err error

	var resp *dto.C2CMessageResponse

	var UserID string

	if message.Params.UserID != nil && len(message.Params.UserID.(string)) != 32 {
		//私聊信息
		if config.GetIdmapPro() {
			//还原真实的userid
			//mylog.Printf("group_private:%v", message.Params.UserID.(string))
			_, UserID, err = idmap.RetrieveRowByIDv2Pro("690426430", message.Params.UserID.(string))
			if err != nil {
				mylog.Printf("Error reading config: %v", err)
				return "", nil
			}
			mylog.Printf("测试,通过Proid获取的UserID:%v", UserID)
		} else {
			//还原真实的userid
			UserID, err = idmap.RetrieveRowByIDv2(message.Params.UserID.(string))
			if err != nil {
				mylog.Printf("Error reading config: %v", err)
				return "", nil
			}
		}
	} else {
		UserID = message.Params.UserID.(string)
	}

	// 首先，将message.Params.Message序列化成JSON字符串
	messageJSON, err := json.Marshal(message.Params.Message)
	if err != nil {
		fmt.Printf("Error marshalling message: %v\n", err)
		return "", nil
	}

	// 然后，将这个JSON字符串反序列化到InterfaceBody类型的对象中
	var messageBody structs.InterfaceBody
	err = json.Unmarshal(messageJSON, &messageBody)
	if err != nil {
		fmt.Printf("Error unmarshalling to InterfaceBody: %v\n", err)
		return "", nil
	}

	// 输出反序列化后的对象，确认是否成功转换
	fmt.Printf("Recovered InterfaceBody: %+v\n", messageBody)

	// 使用 echo 获取消息ID
	var messageID string

	// 如果messageID仍然为空，尝试使用config.GetAppID和UserID的组合来获取messageID
	if messageID == "" {
		if config.GetStringOb11() {
			messageID = GetMessageIDByUseridOrGroupidSP(config.GetAppIDStr(), UserID)
			mylog.Errorf("通过GetMessageIDByUserid函数获取的message_id:" + messageID)
		} else {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), UserID)
			mylog.Errorf("通过GetMessageIDByUserid函数获取的message_id:" + messageID)
		}

	}

	// 获取并打印相关ID
	relatedID := GetRelatedID(messageID)

	dtoSSE := generateMessageSSE(messageBody, messageID, relatedID)

	resp, err = apiv2.PostC2CMessageSSE(context.TODO(), UserID, dtoSSE)
	if err != nil {
		mylog.Errorf("发送文本私聊信息失败: %v", err)
		//如果失败 防止进入递归
		return "", nil
	}

	// 更新或刷新映射关系
	UpdateRelatedID(messageID, resp.Message.ID)

	//发送成功回执
	retmsg, _ = SendC2CResponse(client, err, &message, resp)

	return retmsg, nil
}

func generateMessageSSE(body structs.InterfaceBody, msgID, ID string) *dto.MessageSSE {
	index := incrementIndex(msgID) // 获取并递增Index

	// 将InterfaceBody的PromptKeyboard转换为MessageSSE的结构
	var rows []dto.RowSSE
	for _, label := range body.PromptKeyboard {
		row := dto.RowSSE{
			Buttons: []dto.ButtonSSE{
				{
					RenderData: dto.RenderDataSSE{Label: label, Style: 2},
					Action:     dto.ActionSSE{Type: 2},
				},
			},
		}
		rows = append(rows, row)
	}

	var msgsse dto.MessageSSE

	if body.Content != "" {
		// 确保Markdown已经初始化
		msgsse.Markdown = &dto.MarkdownSSE{}
		msgsse.Markdown.Content = body.Content
	}

	if len(rows) > 0 {
		// 确保PromptKeyboard及其嵌套结构已经初始化
		msgsse.PromptKeyboard = &dto.KeyboardSSE{
			KeyboardContentSSE: dto.KeyboardContentSSE{
				Content: dto.ContentSSE{
					Rows: []dto.RowSSE{}, // 初始化空切片，避免nil切片赋值
				},
			},
		}
		msgsse.PromptKeyboard.KeyboardContentSSE.Content.Rows = rows
	}

	// 剩余字段赋值
	msgsse.MsgType = 2
	msgsse.MsgSeq = index + 10 // 防止seq重复被去重 预留10条信息供上下文发送
	msgsse.Stream = &dto.StreamSSE{
		State: body.State,
		Index: index,
	}

	if ID != "" {
		msgsse.Stream.ID = ID
	}
	if msgID != "" {
		msgsse.MsgID = msgID
	}

	// 初始化ActionButtonSSE，如果CallbackData有值
	if body.CallbackData != "" {
		msgsse.ActionButton = &dto.ActionButtonSSE{
			TemplateID:   body.ActionButton,
			CallbackData: body.CallbackData,
		}
	}

	return &msgsse

}
