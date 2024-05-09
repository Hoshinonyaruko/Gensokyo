package dto

// Interaction 互动行为对象
type Interaction struct {
	ID                string           `json:"id,omitempty"`                  // 平台方事件 ID
	EventID           string           `json:"event_id,omitempty"`            // 外层event_id
	Type              InteractionType  `json:"type,omitempty"`                // 消息按钮: 11, 单聊快捷菜单: 12
	Scene             string           `json:"scene,omitempty"`               // 事件发生的场景
	ChatType          int              `json:"chat_type,omitempty"`           // 频道场景: 0, 群聊场景: 1, 单聊场景: 2
	Timestamp         string           `json:"timestamp,omitempty"`           // 触发时间 RFC 3339 格式
	GuildID           string           `json:"guild_id,omitempty"`            // 频道的 openid
	ChannelID         string           `json:"channel_id,omitempty"`          // 文字子频道的 openid
	UserOpenID        string           `json:"user_openid,omitempty"`         // 单聊按钮触发的用户 openid
	GroupOpenID       string           `json:"group_openid,omitempty"`        // 群的 openid
	GroupMemberOpenID string           `json:"group_member_openid,omitempty"` // 群成员 openid
	Data              *InteractionData `json:"data,omitempty"`                // 互动数据
	Version           uint32           `json:"version,omitempty"`             // 版本，默认为 1
	ApplicationID     string           `json:"application_id,omitempty"`      // 机器人的 appid
}

// InteractionType 互动类型
type InteractionType uint32

const (
	// InteractionTypePing ping
	InteractionTypePing InteractionType = 1
	// InteractionTypeCommand 命令
	InteractionTypeCommand InteractionType = 2
)

// InteractionData 互动数据
type InteractionData struct {
	Name     string              `json:"name,omitempty"` // 标题
	Type     InteractionDataType `json:"type,omitempty"` // 数据类型
	Resolved struct {
		ButtonData string `json:"button_data,omitempty"` // 操作按钮的 data 字段值
		ButtonID   string `json:"button_id,omitempty"`   // 操作按钮的 id 字段值
		UserID     string `json:"user_id,omitempty"`     // 操作的用户 userid
		FeatureID  string `json:"feature_id,omitempty"`  // 操作按钮的 feature id
		MessageID  string `json:"message_id,omitempty"`  // 操作的消息 id
	} `json:"resolved,omitempty"`
}

// InteractionDataType 互动数据类型
type InteractionDataType uint32

const (
	// InteractionDataTypeChatSearch 聊天框搜索
	InteractionDataTypeChatSearch InteractionDataType = 9
)
