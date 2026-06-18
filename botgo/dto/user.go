package dto

// User 用户
type User struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	Avatar           string `json:"avatar"`
	Bot              bool   `json:"bot"`
	UnionOpenID      string `json:"union_openid"`       // 特殊关联应用的 openid
	UnionUserAccount string `json:"union_user_account"` // 机器人关联的用户信息，与union_openid关联的应用是同一个

	// --- [新增] 手术刀级补充：根据群聊 Payload 拓展的缺失字段 ---
	MemberOpenID string `json:"member_openid,omitempty"` // 成员的 OpenID
	MemberRole   string `json:"member_role,omitempty"`   // 成员角色（如 owner, admin, member 等）
	IsYou        bool   `json:"is_you,omitempty"`        // Mentions 数组专属：被 @ 的人是否为当前机器人自己
	Scope        string `json:"scope,omitempty"`         // Mentions 数组专属：@ 的范围（如 single）
}
