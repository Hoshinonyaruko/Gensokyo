package dto

// UpdateGuildMute 更新频道相关禁言的Body参数
type UpdateGuildMute struct {
	// 禁言截止时间戳，单位秒
	MuteEndTimestamp string `json:"mute_end_timestamp,omitempty"`
	// 禁言多少秒（两个字段二选一，默认以mute_end_timstamp为准）
	MuteSeconds string `json:"mute_seconds,omitempty"`
	// 批量禁言的成员列表（全员禁言时不填写该字段）
	UserIDs []string `json:"user_ids,omitempty"`
}

// UpdateGuildMuteResponse 批量禁言的回参
type UpdateGuildMuteResponse struct {
	// 批量禁言成功的成员列表
	UserIDs []string `json:"user_ids,omitempty"`
}

// GroupMuteState QQ群禁言状态。
// 对应 GET /v2/groups/{group_openid}/restrict_chat_setting。
type GroupMuteState struct {
	GlobalRule *GroupMuteGlobalRule   `json:"global_rule,omitempty"`
	Members    []GroupMemberMuteState `json:"members,omitempty"`
}

// GroupMuteGlobalRule QQ群全员禁言规则。
// Gensokyo 当前只适配单成员禁言，这里保留查询接口返回的基本结构。
type GroupMuteGlobalRule struct {
	Mode string `json:"mode,omitempty"`
}

// GroupMemberMuteState QQ群中当前处于禁言状态的成员。
type GroupMemberMuteState struct {
	MemberOpenID string `json:"member_openid"`
	MuteExpireAt string `json:"mute_expire_at"`
	Username     string `json:"username,omitempty"`
	UnionOpenID  string `json:"union_openid,omitempty"`
}

// SetGroupMemberMute QQ群成员禁言请求。
// QQ接口虽然使用数组，但Gensokyo的OneBot set_group_ban只适配单人。
// 因此上层每次只会向Members中放入一个成员。
type SetGroupMemberMute struct {
	Members []SetGroupMemberMuteState `json:"members"`
}

// SetGroupMemberMuteState QQ群单个成员的禁言操作。
//
// Op:
//   add    - 新增禁言
//   update - 修改已有禁言的截止时间
//   del    - 解除禁言
//
// MuteExpireAt 使用RFC3339格式。
// op=del时不需要填写MuteExpireAt。
type SetGroupMemberMuteState struct {
	Op           string `json:"op"`
	MemberOpenID string `json:"member_openid"`
	MuteExpireAt string `json:"mute_expire_at,omitempty"`
}
