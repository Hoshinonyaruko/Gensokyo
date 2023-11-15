package handlers

import (
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler(".handle_quick_operation", handle_quick_operation)
}

func handle_quick_operation(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 根据上下文和操作类型处理请求
	switch message.PostType {
	case "message":
		switch message.MessageType {
		case "group":
			handleSendGroupMsg(client, api, apiv2, message)
		case "private":
			handleSendPrivateMsg(client, api, apiv2, message)
		}
	}
}
