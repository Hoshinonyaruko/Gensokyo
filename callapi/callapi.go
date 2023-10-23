package callapi

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/tencent-connect/botgo/openapi"
)

// onebot发来的action调用信息
type ActionMessage struct {
	Action string        `json:"action"`
	Params ParamsContent `json:"params"`
	Echo   string        `json:"echo,omitempty"`
}

// params类型
type ParamsContent struct {
	BotQQ     string      `json:"botqq"`
	ChannelID string      `json:"channel_id"`
	GuildID   string      `json:"guild_id"`
	GroupID   interface{} `json:"group_id"` // 每一种onebotv11实现的字段类型都可能不同
	Message   interface{} `json:"message"`  // 这里使用interface{}因为它可能是多种类型
	UserID    interface{} `json:"user_id"`  // 这里使用interface{}因为它可能是多种类型
}

// 自定义一个ParamsContent的UnmarshalJSON 让GroupID同时兼容str和int
func (p *ParamsContent) UnmarshalJSON(data []byte) error {
	type Alias ParamsContent
	aux := &struct {
		GroupID interface{} `json:"group_id"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.GroupID.(type) {
	case nil: // 当GroupID不存在时
		p.GroupID = ""
	case float64: // JSON的数字默认被解码为float64
		p.GroupID = fmt.Sprintf("%.0f", v) // 将其转换为字符串，忽略小数点后的部分
	case string:
		p.GroupID = v
	default:
		return fmt.Errorf("GroupID has unsupported type")
	}
	return nil
}

// Message represents a standardized structure for the incoming messages.
type Message struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
	Echo   interface{}            `json:"echo,omitempty"`
}

// 这是一个接口,在wsclient传入client但不需要引用wsclient包,避免循环引用
type Client interface {
	SendMessage(message map[string]interface{}) error
}

// 根据action订阅handler处理api
type HandlerFunc func(client Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, messgae ActionMessage)

var handlers = make(map[string]HandlerFunc)

// RegisterHandler registers a new handler for a specific action.
func RegisterHandler(action string, handler HandlerFunc) {
	handlers[action] = handler
}

// CallAPIFromDict 处理信息 by calling the 对应的 handler.
func CallAPIFromDict(client Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message ActionMessage) {
	handler, ok := handlers[message.Action]
	if !ok {
		log.Println("Unsupported action:", message.Action)
		return
	}
	handler(client, api, apiv2, message)
}
