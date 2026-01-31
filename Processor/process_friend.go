package Processor

import (
	"fmt"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
)

// FriendRequestEvent 表示好友请求事件的数据结构 (对应 OneBot V11 request.friend)
type FriendRequestEvent struct {
	PostType    string `json:"post_type"`
	RequestType string `json:"request_type"`
	UserID      int64  `json:"user_id"`
	Comment     string `json:"comment"` // 用于存放 scene_param (callbackData)
	Flag        string `json:"flag"`    // 这里存放 event_id 作为 flag
	Time        int64  `json:"time"`
	SelfID      int64  `json:"self_id"`
	RealUserID  string `json:"real_user_id,omitempty"` //当前真实uid
}

// FriendNoticeEvent 表示好友变动通知事件的数据结构 (对应 OneBot V11 notice.friend_add 等)
type FriendNoticeEvent struct {
	PostType   string `json:"post_type"`
	NoticeType string `json:"notice_type"`
	UserID     int64  `json:"user_id"`
	Time       int64  `json:"time"`
	SelfID     int64  `json:"self_id"`
	RealUserID string `json:"real_user_id,omitempty"` // 当前真实uid

	// [新增] 场景值与参数，用于归因
	Scene      int    `json:"scene,omitempty"`       // 添加好友的场景值
	SceneParam string `json:"scene_param,omitempty"` // 场景参数 (CallbackData)
}

// ProcessFriendAdd 处理好友添加 (用户添加机器人)
func (p *Processors) ProcessFriendAdd(data *dto.WSFriendAddData) error {
	var userid64 int64
	var err error
	var Request FriendRequestEvent
	var Notice FriendNoticeEvent

	// 1. ID 转换
	userid64, err = idmap.StoreIDv2(data.OpenID)
	if err != nil {
		mylog.Printf("Error storing ID: %v", err)
		return nil
	}

	// 2. 时间戳转换
	timestampInt64 := int64(data.Timestamp)

	// 3. 获取自身 ID
	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}

	// 4. 构造 Request 事件 (模拟 request.friend)
	// 将 SceneParam (CallbackData) 放入 Comment，方便上层业务做归因
	commentMsg := data.SceneParam
	if commentMsg == "" {
		commentMsg = fmt.Sprintf("Scene: %d", data.Scene)
	}

	Request = FriendRequestEvent{
		PostType:    "request",
		RequestType: "friend",
		UserID:      userid64,
		Comment:     commentMsg,
		Flag:        data.EventID,
		Time:        timestampInt64,
		SelfID:      selfid64,
	}

	if !config.GetNativeOb11() {
		Request.RealUserID = data.OpenID
	}

	// 5. 构造 Notice 事件 (标准 notice.friend_add)
	Notice = FriendNoticeEvent{
		PostType:   "notice",
		NoticeType: "friend_add", // 标准 OB11 类型
		UserID:     userid64,
		Time:       timestampInt64,
		SelfID:     selfid64,

		// [新增] 关键修正：传递场景值和参数
		Scene:      data.Scene,
		SceneParam: data.SceneParam,
	}

	// 增强配置
	if !config.GetNativeOb11() {
		Notice.RealUserID = data.OpenID
	}

	mylog.Printf("Bot被用户[%v]添加为好友, 场景[%d], 参数[%s]", userid64, data.Scene, data.SceneParam)

	// 6. 广播 Request 事件
	reqMsgMap := structToMap(Request)
	go p.BroadcastMessageToAll(reqMsgMap, p.Apiv2, data)

	// 7. 广播 Notice 事件
	noticeMsgMap := structToMap(Notice)
	go p.BroadcastMessageToAll(noticeMsgMap, p.Apiv2, data)

	return nil
}

// ProcessFriendDel 处理好友删除 (用户删除机器人)
func (p *Processors) ProcessFriendDel(data *dto.WSFriendDelData) error {
	var userid64 int64
	var err error
	var Notice FriendNoticeEvent

	// 1. ID 转换
	userid64, err = idmap.StoreIDv2(data.OpenID)
	if err != nil {
		mylog.Printf("Error storing ID: %v", err)
		return nil
	}

	// 2. 时间戳转换
	timestampInt64 := int64(data.Timestamp)

	// 3. 获取自身 ID
	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}

	// 4. 清理缓存 (可选，参考 GroupDelBot 逻辑)
	// idmap.DeleteConfigv2(fmt.Sprint(userid64), "user_type") // 如果有类似缓存可清理

	mylog.Printf("Bot被用户[%v]解除好友关系", userid64)

	// 5. 构造 Notice 事件
	// 注意: OneBot V11 标准没有 friend_decrease，这里定义为自定义类型
	Notice = FriendNoticeEvent{
		PostType:   "notice",
		NoticeType: "friend_decrease", // 自定义类型
		UserID:     userid64,
		Time:       timestampInt64,
		SelfID:     selfid64,
	}

	// 增强配置
	if !config.GetNativeOb11() {
		Notice.RealUserID = data.OpenID
	}

	// 6. 广播事件
	noticeMsgMap := structToMap(Notice)
	go p.BroadcastMessageToAll(noticeMsgMap, p.Apiv2, data)

	return nil
}
