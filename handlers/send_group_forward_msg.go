package handlers

import (
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_group_forward_msg", HandleSendGroupForwardMsg)
}

func HandleSendGroupForwardMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	nodes, ok := message.Params.Messages.([]interface{})
	if !ok {
		mylog.Printf("send_group_forward_msg: Messages 不是 []interface{} 类型")
		return "", nil
	}
	var retmsg string
	forwardMsgLimit := config.GetForwardMsgLimit() // 获取消息发送条数上限
	count := 0

	for _, nodeInterface := range nodes {
		if count > forwardMsgLimit {
			break // 如果消息数量已达上限，则停止发送
		}

		nodeMap, ok := nodeInterface.(map[string]interface{})
		if !ok {
			continue
		}

		nodeData, ok := nodeMap["data"].(map[string]interface{})
		if !ok {
			continue
		}

		var messageText string

		// 检查 content 的类型
		content, ok := nodeData["content"].([]interface{})
		if ok {
			// 处理 segment 类型的 content
			messageText, _ = parseMessageContent(callapi.ParamsContent{Message: content}, message, client, api, apiv2)
		} else {
			// 处理直接包含的文本内容
			contentString, ok := nodeData["content"].(string)
			if ok {
				messageText = contentString
			}
		}

		// 创建新的 ActionMessage 实例并发送
		newMessage := callapi.ActionMessage{
			Action: "send_group_msg",
			Params: callapi.ParamsContent{
				GroupID: message.Params.GroupID,
				Message: messageText,
			},
		}

		HandleSendGroupMsg(client, api, apiv2, newMessage)
		count++
		time.Sleep(500 * time.Millisecond) // 每条消息之间的延时
	}
	retmsg, _ = SendResponse(client, nil, &message, nil, api, apiv2)
	return retmsg, nil
}
