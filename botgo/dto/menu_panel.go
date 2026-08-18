package dto

// Menu item type.
const (
	MenuItemTypeSendMessage = "send_message"
	MenuItemTypeLink        = "link"
	MenuItemTypeSwitch      = "switch"
	MenuItemTypeMenu        = "menu"
)

// Panel scope.
const (
	PanelScopeC2C     = "c2c"
	PanelScopeGroup   = "group"
	PanelScopeChannel = "channel"
	PanelScopeDM      = "dm"
)

// Panel target type.
const (
	PanelTargetTypeAll      = "all"
	PanelTargetTypeSpecific = "specific"
)

// Panel item type.
const (
	PanelItemTypeCommand = "command"
	PanelItemTypeLink    = "link"
)

// Panel target operation.
const (
	PanelTargetOpAdd = "add"
	PanelTargetOpDel = "del"
)

// MenuSwitch 自定义菜单开关配置.
type MenuSwitch struct {
	SwitchID string `json:"switch_id"`
	Default  bool   `json:"default"`
}

// MenuItem 自定义菜单项.
type MenuItem struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	SendMessage  string      `json:"send_message,omitempty"`
	Link         string      `json:"link,omitempty"`
	Switch       *MenuSwitch `json:"switch,omitempty"`
	SubMenuItems []*MenuItem `json:"sub_menu_items,omitempty"`
}

// Menu 全局自定义菜单.
type Menu struct {
	Items []*MenuItem `json:"items"`
}

// MenuWrapper 自定义菜单接口包装对象.
// Menu 为 nil 时序列化为 {}，可用于清空全局菜单.
type MenuWrapper struct {
	Menu *Menu `json:"menu,omitempty"`
}

// PanelItem 指令面板中的单个指令项.
type PanelItem struct {
	Name      string `json:"name"`
	Desc      string `json:"desc,omitempty"`
	Type      string `json:"type"`
	OnlyAdmin bool   `json:"only_admin"`
	Link      string `json:"link,omitempty"`
}

// Panel 指令面板主体.
type Panel struct {
	Items  []*PanelItem `json:"items"`
	Remark string       `json:"remark"`
}

// PanelRecord QQ 远端指令面板记录.
// List 接口和 Detail 接口可以共用该结构，Detail 接口会额外返回关联对象.
type PanelRecord struct {
	PanelID      string   `json:"panel_id"`
	Scope        string   `json:"scope"`
	TargetType   string   `json:"target_type"`
	Panel        *Panel   `json:"panel"`
	UserOpenIDs  []string `json:"user_openids,omitempty"`
	GroupOpenIDs []string `json:"group_openids,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	Version      int      `json:"version,omitempty"`
}

// PanelsResponse 分页查询指令面板响应.
type PanelsResponse struct {
	Records    []*PanelRecord `json:"records"`
	NextCursor string         `json:"next_cursor,omitempty"`
	IsEnd      bool           `json:"is_end"`
}

// PanelToCreate 创建指令面板请求.
type PanelToCreate struct {
	Scope        string   `json:"scope"`
	TargetType   string   `json:"target_type,omitempty"`
	Panel        *Panel   `json:"panel"`
	UserOpenIDs  []string `json:"user_openids,omitempty"`
	GroupOpenIDs []string `json:"group_openids,omitempty"`
}

// PanelCreateResponse 创建指令面板响应.
type PanelCreateResponse struct {
	PanelID string `json:"panel_id"`
}

// PanelWrapper 更新指令面板请求.
// 官方更新接口只修改 panel 内容，不修改 target.
type PanelWrapper struct {
	Panel *Panel `json:"panel"`
}

// PanelTargetToUpdate 修改指令面板关联对象请求.
type PanelTargetToUpdate struct {
	Op           string   `json:"op"`
	UserOpenIDs  []string `json:"user_openids,omitempty"`
	GroupOpenIDs []string `json:"group_openids,omitempty"`
}
