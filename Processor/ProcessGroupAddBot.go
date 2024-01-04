// 处理收到的信息事件
package Processor

import (
	"math/rand"
	"strconv"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
)

// GroupRequestEvent 表示群组请求事件的数据结构
type GroupRequestEvent struct {
	Comment     string `json:"comment"`
	Flag        string `json:"flag"`
	GroupID     int64  `json:"group_id"`
	PostType    string `json:"post_type"`
	RequestType string `json:"request_type"`
	SelfID      int64  `json:"self_id"`
	SubType     string `json:"sub_type"`
	Time        int64  `json:"time"`
	UserID      int64  `json:"user_id"`
}

// GroupNoticeEvent 表示群通知事件的数据结构
type GroupNoticeEvent struct {
	GroupID    int64  `json:"group_id"`
	NoticeType string `json:"notice_type"`
	OperatorID int64  `json:"operator_id"`
	PostType   string `json:"post_type"`
	SelfID     int64  `json:"self_id"`
	SubType    string `json:"sub_type"`
	Time       int64  `json:"time"`
	UserID     int64  `json:"user_id"`
}

// 定义了一个符合 Client 接口的 SelfIntroduceClient 结构体
type SelfIntroduceClient struct {
	// 可添加所需字段
}

// 实现 Client 接口的 SendMessage 方法
// 假client中不执行任何操作，只是返回 nil 来符合接口要求
func (c *SelfIntroduceClient) SendMessage(message map[string]interface{}) error {
	// 不实际发送消息
	// log.Printf("SendMessage called with: %v", message)

	// 返回nil占位符
	return nil
}

// ProcessGroupAddBot 处理机器人增加
func (p *Processors) ProcessGroupAddBot(data *dto.GroupAddBotEvent) error {
	var userid64 int64
	var GroupID64 int64
	var err error
	var Request GroupRequestEvent
	var Notice GroupNoticeEvent
	if config.GetIdmapPro() {
		GroupID64, userid64, err = idmap.StoreIDv2Pro(data.GroupOpenID, data.OpMemberOpenID)
		if err != nil {
			mylog.Fatalf("Error storing ID: %v", err)
		}
	} else {
		GroupID64, err = idmap.StoreIDv2(data.GroupOpenID)
		if err != nil {
			mylog.Errorf("failed to convert ChannelID to int: %v", err)
			return nil
		}
		userid64, err = idmap.StoreIDv2(data.OpMemberOpenID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
	}
	var timestampInt64 int64
	switch v := data.Timestamp.(type) {
	case string:
		timestampInt64, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			mylog.Printf("Error converting timestamp string to int64: %v", err)
			return nil
		}
	case int64:
		timestampInt64 = v
	case float64:
		timestampInt64 = int64(v)
	default:
		mylog.Printf("Invalid type for timestamp: %T", v)
		return nil
	}
  
	mylog.Printf("Bot被[%v]邀请进入群[%v]", userid64, GroupID64)
	Request = GroupRequestEvent{
		Comment:     "",
		Flag:        "",
		GroupID:     GroupID64,
		PostType:    "request",
		RequestType: "group",
		SelfID:      int64(config.GetAppID()),
		SubType:     "invite",
		Time:        timestampInt64,
		UserID:      userid64,
	}
	Notice = GroupNoticeEvent{
		GroupID:    GroupID64,
		NoticeType: "group_increase",
		OperatorID: 0,
		PostType:   "notice",
		SelfID:     int64(config.GetAppID()),
		SubType:    "invite",
		Time:       timestampInt64,
		UserID:     userid64,
	}
	groupMsgMap := structToMap(Request)
	//上报信息到onebotv11应用端(正反ws)
	p.BroadcastMessageToAll(groupMsgMap)

	groupMsgMap = structToMap(Notice)
	//上报信息到onebotv11应用端(正反ws)
	p.BroadcastMessageToAll(groupMsgMap)

	// 调用GetSelfIntroduce函数
	intros := config.GetSelfIntroduce()

	// 检查intros是否为空或只包含空字符串
	var validIntros []string
	for _, intro := range intros {
		if intro != "" {
			validIntros = append(validIntros, intro)
		}
	}

	if len(validIntros) == 0 {
		return nil
	}

	// 从validIntros中随机选择一个
	selectedIntro := validIntros[rand.Intn(len(validIntros))]

	// 创建 ActionMessage 实例
	message := callapi.ActionMessage{
		Action: "send_group_msg",
		Params: callapi.ParamsContent{
			GroupID: strconv.FormatInt(GroupID64, 10), // 转换 GroupID 类型
			UserID:  strconv.FormatInt(userid64, 10),
			Message: selectedIntro,
		},
	}
	// clinet是发回值用的 这里相当于和http一样 不发回值所以建立一个假的client
	client := &SelfIntroduceClient{}
	// 调用处理函数
	_, err = handlers.HandleSendGroupMsg(client, p.Api, p.Apiv2, message)
	if err != nil {
		mylog.Printf("自我介绍发送失败%v", err)
		return nil
	}
	return nil
}
