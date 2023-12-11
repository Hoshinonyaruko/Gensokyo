package handlers

import (
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

// 初始化handler，在程序启动时会被调用
func init() {
	callapi.RegisterHandler("get_group_member_info", GetGroupMemberInfo)
}

// 成员信息的结构定义
type MemberInfo struct {
	UserID          int64  `json:"user_id"`
	GroupID         int64  `json:"group_id"`
	Nickname        string `json:"nickname"`
	Card            string `json:"card"`
	Sex             string `json:"sex"`
	Age             int32  `json:"age"`
	Area            string `json:"area"`
	JoinTime        int32  `json:"join_time"`
	LastSentTime    int32  `json:"last_sent_time"`
	Level           string `json:"level"`
	Role            string `json:"role"`
	Unfriendly      bool   `json:"unfriendly"`
	Title           string `json:"title"`
	TitleExpireTime int64  `json:"title_expire_time"`
	CardChangeable  bool   `json:"card_changeable"`
	ShutUpTimestamp int64  `json:"shut_up_timestamp"`
}

// 构建单个成员的响应数据
func buildResponseForSingleMember(memberInfo *MemberInfo, echoValue interface{}) map[string]interface{} {
	// 构建成员数据的映射
	memberMap := map[string]interface{}{
		"group_id":          memberInfo.GroupID,
		"user_id":           memberInfo.UserID,
		"nickname":          memberInfo.Nickname,
		"card":              memberInfo.Card,
		"sex":               memberInfo.Sex,
		"age":               memberInfo.Age,
		"area":              memberInfo.Area,
		"join_time":         memberInfo.JoinTime,
		"last_sent_time":    memberInfo.LastSentTime,
		"level":             memberInfo.Level,
		"role":              memberInfo.Role,
		"unfriendly":        memberInfo.Unfriendly,
		"title":             memberInfo.Title,
		"title_expire_time": memberInfo.TitleExpireTime,
		"card_changeable":   memberInfo.CardChangeable,
		"shut_up_timestamp": memberInfo.ShutUpTimestamp,
	}

	// 构建完整的响应映射
	response := map[string]interface{}{
		"retcode": 0,
		"status":  "ok",
		"data":    memberMap,
		"echo":    echoValue,
	}

	return response
}

// getGroupMemberInfo是处理获取群成员信息的函数
func GetGroupMemberInfo(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 使用虚拟数据构造 MemberInfo
	memberInfo := &MemberInfo{
		UserID:          123456789, // 虚拟的 QQ 号
		GroupID:         987654321, // 虚拟的群号
		Nickname:        "主人",      // 虚拟昵称
		Card:            "主人",
		Sex:             "unknown", // 性别未知
		Age:             20,        // 虚拟年龄
		Area:            "虚拟地区",
		JoinTime:        1630416000, // 虚拟加群时间戳
		LastSentTime:    1630502400, // 虚拟最后发言时间戳
		Level:           "1",        // 虚拟成员等级
		Role:            "member",   // 角色为普通成员
		Unfriendly:      false,      // 没有不良记录
		Title:           "虚拟头衔",
		TitleExpireTime: 1630598800, // 虚拟头衔过期时间
		CardChangeable:  true,       // 允许修改群名片
		ShutUpTimestamp: 0,          // 不在禁言中
	}

	// 构建响应JSON
	responseJSON := buildResponseForSingleMember(memberInfo, message.Echo)
	mylog.Printf("get_group_member_info: %s\n", responseJSON)

	// 发送响应回去
	err := client.SendMessage(responseJSON)
	if err != nil {
		mylog.Printf("发送消息时出错: %v", err)
	}
	result, err := ConvertMapToJSONString(responseJSON)
	if err != nil {
		mylog.Printf("Error marshaling data: %v", err)
		//todo 符合onebotv11 ws返回的错误码
		return "", nil
	}
	return string(result), nil
}
