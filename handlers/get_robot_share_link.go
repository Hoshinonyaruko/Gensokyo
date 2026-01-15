package handlers

import (
	"context"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

// ShareLinkNotice 伪装成 Notice 的响应结构
// 对应 OneBot 的 notice 事件结构
type ShareLinkNotice struct {
	PostType     string `json:"post_type"`     // 固定为 notice
	NoticeType   string `json:"notice_type"`   // 我们自定义的类型: "share_link_generated"
	URL          string `json:"url"`           // 核心数据：链接
	CallbackData string `json:"callback_data"` // 透传回来的参数，方便你区分是哪个请求
	Time         int64  `json:"time"`
	SelfID       int64  `json:"self_id"`
}

func init() {
	// 注册 Action
	callapi.RegisterHandler("get_robot_share_link", GetRobotShareLink)
}

// GetRobotShareLink 获取机器人资料页分享链接
// 修改策略：不再返回标准Response，而是发送一个 Notice 事件
func GetRobotShareLink(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {

	// 1. 获取参数
	callbackData := message.Params.CallbackData

	// 2. 调用腾讯 v2 API
	req := &dto.GenerateURLLinkToCreate{
		CallbackData: callbackData,
	}

	// 此时 selfID 最好从配置或 message 中获取，这里演示用 0 或 message.SelfID (如果你的ActionMessage里有)
	var selfID int64 = int64(config.GetAppID())

	resultData, err := apiv2.GenerateURLLink(context.Background(), req)
	if err != nil {
		mylog.Printf("Error generating robot share link: %v", err)
		// 如果出错，也可以选择发送一个 notice_type: "share_link_failed"
		return "", nil // 返回空，不做处理
	}

	// 3. 【关键步骤】构建伪造的 Notice 事件
	fakeNotice := ShareLinkNotice{
		PostType:     "notice",
		NoticeType:   "share_link_generated", // 自定义类型，要在客户端 switch 里加这个
		URL:          resultData.URL,
		CallbackData: callbackData, // 把请求时的参数带回来，这就相当于一种手动 echo
		Time:         time.Now().Unix(),
		SelfID:       selfID,
	}

	// 4. 转换为 Map 并发送 WebSocket 消息
	outputMap := structToMap(fakeNotice)

	mylog.Printf("Pushing Fake Notice: %+v\n", outputMap)

	// 通过 client 发送回包 (假装是 Event 推送)
	err = client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
	}

	// 因为是假装 Event 推送，这里函数返回值其实已经不重要了
	// 返回一个空 JSON 对象即可，满足函数签名
	return "{}", nil
}
