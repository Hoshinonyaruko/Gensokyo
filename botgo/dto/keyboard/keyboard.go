package keyboard

// ActionType 按钮操作类型
type ActionType uint32

// PermissionType 按钮的权限类型
type PermissionType uint32

const (
	// ActionTypeURL http 或 小程序 客户端识别 schema, data字段为链接
	ActionTypeURL ActionType = 0
	// ActionTypeCallback 回调互动回调地址, data 传给互动回调地址
	ActionTypeCallback ActionType = 1
	// ActionTypeAtBot at机器人, 根据 at_bot_show_channel_list 决定在当前频道或用户选择频道,自动在输入框 @bot data
	ActionTypeAtBot ActionType = 2
	// PermissionTypeSpecifyUserIDs 仅指定这条消息的人可操作
	PermissionTypeSpecifyUserIDs PermissionType = 0
	// PermissionTypManager  仅频道管理者可操作
	PermissionTypManager PermissionType = 1
	// PermissionTypAll  所有人可操作
	PermissionTypAll PermissionType = 2
	// PermissionTypSpecifyRoleIDs 指定身份组可操作
	PermissionTypSpecifyRoleIDs PermissionType = 3
)

// MessageKeyboard 消息按钮组件
type MessageKeyboard struct {
	ID      string          `json:"id,omitempty"`      // 消息按钮组件模板 ID
	Content *CustomKeyboard `json:"content,omitempty"` // 消息按钮组件自定义内容
}

// CustomKeyboard 自定义 Keyboard
type CustomKeyboard struct {
	Rows []*Row `json:"rows,omitempty"` // 行数组
}

// Row 每行结构
type Row struct {
	Buttons []*Button `json:"buttons,omitempty"` // 每行按钮
}

// Button 单个按纽
type Button struct {
	ID         string      `json:"id,omitempty"`          // 按钮 ID
	RenderData *RenderData `json:"render_data,omitempty"` // 渲染展示字段
	Action     *Action     `json:"action,omitempty"`      // 该按纽操作相关字段
}

// RenderData  按纽渲染展示
type RenderData struct {
	Label        string `json:"label,omitempty"`         // 按纽上的文字
	VisitedLabel string `json:"visited_label,omitempty"` // 点击后按纽上文字
	Style        int    `json:"style,omitempty"`         // 按钮样式，0：灰色线框，1：蓝色线框
}

// Action 按纽点击操作
type Action struct {
	Type                 ActionType  `json:"type,omitempty"`                     // 操作类型 设置 0 跳转按钮：http 或 小程序 客户端识别 scheme，设置 1 回调按钮：回调后台接口, data 传给后台，设置 2 指令按钮：自动在输入框插入 @bot data
	Permission           *Permission `json:"permission,omitempty"`               // 可操作
	ClickLimit           uint32      `json:"click_limit,omitempty"`              // 可点击的次数, 默认不限
	Data                 string      `json:"data,omitempty"`                     // 操作相关数据
	AtBotShowChannelList bool        `json:"at_bot_show_channel_list,omitempty"` // false:当前 true:弹出展示子频道选择器
	UnsupportTips        string      `json:"unsupport_tips"`                     //2024-1-12 新增字段
	AnChor               int         `json:"anchor"`                             //本字段仅在指令按钮下有效，设置后后会忽略 action.enter 配置。设置为 1 时 ，点击按钮自动唤起启手Q选图器，其他值暂无效果。（仅支持手机端版本 8983+ 的单聊场景，桌面端不支持）
	Enter                bool        `json:"enter"`                              //指令按钮可用，点击按钮后直接自动发送 data，默认 false。支持版本 8983
	Reply                bool        `json:"reply"`                              //指令按钮可用，指令是否带引用回复本消息，默认 false。支持版本 8983
}

// Permission 按纽操作权限
type Permission struct {
	// Type 操作权限类型 0 指定用户可操作，1 仅管理者可操作，2 所有人可操作，3 指定身份组可操作（仅频道可用）
	Type PermissionType `json:"type,omitempty"`
	// SpecifyRoleIDs 身份组（仅频道可用）
	SpecifyRoleIDs []string `json:"specify_role_ids,omitempty"`
	// SpecifyUserIDs 指定 UserID 有权限的用户 id 的列表
	SpecifyUserIDs []string `json:"specify_user_ids,omitempty"`
}
