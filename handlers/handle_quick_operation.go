package handlers

import (
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler(".handle_quick_operation", handle_quick_operation)
}

func handle_quick_operation(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 使用 CreateSendGroupMsgAction 函数来确定如何处理消息
	newMsg := CreateSendGroupMsgAction(message)

	// 根据返回的 ActionMessage 类型调用相应的处理函数
	if newMsg != nil {
		switch newMsg.Action {
		case "send_group_msg":
			handleSendGroupMsg(client, api, apiv2, *newMsg)
		case "send_private_msg":
			handleSendPrivateMsg(client, api, apiv2, *newMsg)
		}
	}
}

func CreateSendGroupMsgAction(originalMsg callapi.ActionMessage) *callapi.ActionMessage {
	switch originalMsg.Params.Context.MessageType {
	case "group":
		return &callapi.ActionMessage{
			Action: "send_group_msg",
			Params: callapi.ParamsContent{
				GroupID: originalMsg.Params.Context.GroupID,
				Message: originalMsg.Params.Operation.Reply,
			},
		}

	case "private":
		return &callapi.ActionMessage{
			Action: "send_private_msg",
			Params: callapi.ParamsContent{
				UserID:  originalMsg.Params.Context.UserID,
				Message: originalMsg.Params.Operation.Reply,
			},
		}

	default:
		return nil // 或处理其他类型消息的逻辑
	}
}
