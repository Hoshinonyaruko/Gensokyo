package commandpanel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tencent-connect/botgo/dto"
)

// API 是 commandpanel 当前需要的最小 botgo 能力集合.
//
// 故意不直接依赖庞大的 openapi.OpenAPI 接口。
// apiV2 天然实现这些方法，同时这样也便于后续做单元测试。
type API interface {
	GetGlobalMenu(ctx context.Context) (*dto.Menu, error)
	UpdateGlobalMenu(ctx context.Context, menu *dto.Menu) error

	ListPanels(
		ctx context.Context,
		scope string,
		cursor string,
		limit int,
	) (*dto.PanelsResponse, error)

	CreatePanel(
		ctx context.Context,
		value *dto.PanelToCreate,
	) (*dto.PanelCreateResponse, error)

	GetPanel(
		ctx context.Context,
		panelID string,
	) (*dto.PanelRecord, error)

	UpdatePanel(
		ctx context.Context,
		panelID string,
		panel *dto.Panel,
	) error

	DeletePanel(
		ctx context.Context,
		panelID string,
	) error

	UpdatePanelTargets(
		ctx context.Context,
		panelID string,
		value *dto.PanelTargetToUpdate,
	) error
}

// Manager QQ自定义菜单与指令面板管理器.
//
// Manager 负责组合 botgo 的 Menu / Panel API，
// 对外提供远端读取、配置应用、备份与恢复所需的统一操作。
//
// 保持单层管理器设计，不再额外堆
// service/repository/controller 等无必要层次。
type Manager struct {
	api API

	// QQ 指令面板写操作必须串行。
	//
	// 不仅是为了避免 WebUI 连点导致并发写，
	// 腾讯平台还存在 40030009「指令面板操作进行中」。
	writeMu sync.Mutex
}

// RemoteState 表示 QQ 当前实际生效的远端状态.
//
// Panels 使用一个扁平数组保存。
// 每个 Panel 自身已经带有 scope，因此没有必要再构造四层 map。
// 后续 WebUI 按 scope 分组即可。
type RemoteState struct {
	Menu        *dto.Menu          `json:"menu"`
	Panels      []*dto.PanelRecord `json:"panels"`
	TotalPanels int                `json:"total_panels"`
}

const (
	panelPageLimit      = 50
	maxPanelPageFetches = 100
)

var panelScopes = [...]string{
	dto.PanelScopeC2C,
	dto.PanelScopeGroup,
	dto.PanelScopeChannel,
	dto.PanelScopeDM,
}

// NewManager 创建 commandpanel Manager.
func NewManager(api API) *Manager {
	return &Manager{
		api: api,
	}
}

// FetchOverview 获取适合 WebUI 实时刷新的远端状态.
//
// 只执行：
//  1. GET /v2/menu
//  2. 四个 scope 的 GET /v2/panels
//
// 不主动查询每个 Panel Detail。
// 因此非常适合浏览器以后每 30 秒刷新一次。
func (m *Manager) FetchOverview(ctx context.Context) (*RemoteState, error) {
	if m == nil || m.api == nil {
		return nil, fmt.Errorf("commandpanel api is nil")
	}

	menu, err := m.api.GetGlobalMenu(ctx)
	if err != nil {
		return nil, fmt.Errorf("get global menu: %w", err)
	}

	state := &RemoteState{
		Menu:   menu,
		Panels: make([]*dto.PanelRecord, 0),
	}

	for _, scope := range panelScopes {
		records, err := m.listAllPanels(ctx, scope)
		if err != nil {
			return nil, err
		}

		state.Panels = append(state.Panels, records...)
	}

	state.TotalPanels = len(state.Panels)

	return state, nil
}

// FetchFullState 获取完整远端状态.
//
// 先通过 List 获取 Menu + 所有 Panel。
// 对 specific Panel 再请求 Detail，以获得完整的
// user_openids / group_openids。
//
// 这个方法以后用于：
//   - 完整备份
//   - Apply 前读取真实状态
//   - Restore
//   - 最终 Verify
//
// 不用于前端 30 秒轮询。
func (m *Manager) FetchFullState(ctx context.Context) (*RemoteState, error) {
	state, err := m.FetchOverview(ctx)
	if err != nil {
		return nil, err
	}

	for _, record := range state.Panels {
		if record == nil {
			continue
		}

		if record.TargetType != dto.PanelTargetTypeSpecific {
			continue
		}

		if record.PanelID == "" {
			return nil, fmt.Errorf(
				"specific panel in scope %s has empty panel_id",
				record.Scope,
			)
		}

		detail, err := m.GetPanelDetail(ctx, record.PanelID)
		if err != nil {
			return nil, err
		}

		if err := validateSpecificPanelDetail(
			record,
			detail,
		); err != nil {
			return nil, err
		}

		mergePanelDetail(record, detail)
	}

	return state, nil
}

// GetPanelDetail 获取一个 Panel 的完整详情.
//
// WebUI 后续点击 specific Panel 的“查看”按钮时也直接调用它。
func (m *Manager) GetPanelDetail(
	ctx context.Context,
	panelID string,
) (*dto.PanelRecord, error) {
	if m == nil || m.api == nil {
		return nil, fmt.Errorf("commandpanel api is nil")
	}

	panelID = strings.TrimSpace(panelID)

	if panelID == "" {
		return nil, fmt.Errorf("panel_id is empty")
	}

	panel, err := m.api.GetPanel(ctx, panelID)
	if err != nil {
		return nil, fmt.Errorf(
			"get panel detail %s: %w",
			panelID,
			err,
		)
	}

	if panel == nil {
		return nil, fmt.Errorf(
			"get panel detail %s: empty response",
			panelID,
		)
	}

	switch {
	case panel.PanelID == "":
		// Detail 如果省略 panel_id，则使用请求目标补齐。
		panel.PanelID = panelID

	case panel.PanelID != panelID:
		return nil, fmt.Errorf(
			"get panel detail %s: response panel_id mismatch: %s",
			panelID,
			panel.PanelID,
		)
	}

	return panel, nil
}

// listAllPanels 拉取指定 scope 的所有 Panel.
//
// 腾讯当前一个机器人最多只有 20 个 Panel，正常情况下 limit=50
// 一次即可完成。但这里仍然正确处理 cursor，避免把当前平台限制
// 错误地写死成客户端假设。
func (m *Manager) listAllPanels(
	ctx context.Context,
	scope string,
) ([]*dto.PanelRecord, error) {
	records := make([]*dto.PanelRecord, 0)

	cursor := ""
	seenCursors := make(map[string]struct{})
	seenPanelIDs := make(map[string]struct{})

	for page := 0; page < maxPanelPageFetches; page++ {
		result, err := m.api.ListPanels(
			ctx,
			scope,
			cursor,
			panelPageLimit,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"list panels scope=%s: %w",
				scope,
				err,
			)
		}

		if result == nil {
			return nil, fmt.Errorf(
				"list panels scope=%s: empty response",
				scope,
			)
		}

		for _, record := range result.Records {
			if record == nil {
				continue
			}

			panelID := strings.TrimSpace(record.PanelID)
			if panelID == "" {
				return nil, fmt.Errorf(
					"list panels scope=%s: panel has empty panel_id",
					scope,
				)
			}

			record.PanelID = panelID

			if _, exists := seenPanelIDs[panelID]; exists {
				return nil, fmt.Errorf(
					"list panels scope=%s: duplicate panel_id=%s",
					scope,
					panelID,
				)
			}

			seenPanelIDs[panelID] = struct{}{}

			// 平台正常应该返回 scope。
			// 如果列表记录缺失，则使用本次查询 scope 补齐。
			switch {
			case record.Scope == "":
				record.Scope = scope

			case record.Scope != scope:
				return nil, fmt.Errorf(
					"list panels scope=%s: panel_id=%s returned mismatched scope=%s",
					scope,
					record.PanelID,
					record.Scope,
				)
			}

			switch record.TargetType {
			case dto.PanelTargetTypeAll:
				// 合法.

			case dto.PanelTargetTypeSpecific:
				if scope != dto.PanelScopeC2C &&
					scope != dto.PanelScopeGroup {
					return nil, fmt.Errorf(
						"list panels scope=%s: panel_id=%s returned unsupported specific target_type",
						scope,
						record.PanelID,
					)
				}

			default:
				return nil, fmt.Errorf(
					"list panels scope=%s: panel_id=%s returned invalid target_type=%q",
					scope,
					record.PanelID,
					record.TargetType,
				)
			}

			if record.Panel == nil {
				return nil, fmt.Errorf(
					"list panels scope=%s: panel_id=%s has empty panel content",
					scope,
					record.PanelID,
				)
			}

			records = append(records, record)
		}

		if result.IsEnd {
			return records, nil
		}

		nextCursor := strings.TrimSpace(result.NextCursor)

		if nextCursor == "" {
			return nil, fmt.Errorf(
				"list panels scope=%s: response is not end but next_cursor is empty",
				scope,
			)
		}

		if nextCursor == cursor {
			return nil, fmt.Errorf(
				"list panels scope=%s: cursor did not advance: %s",
				scope,
				cursor,
			)
		}

		if _, exists := seenCursors[nextCursor]; exists {
			return nil, fmt.Errorf(
				"list panels scope=%s: cursor loop detected: %s",
				scope,
				nextCursor,
			)
		}

		seenCursors[nextCursor] = struct{}{}
		cursor = nextCursor
	}

	return nil, fmt.Errorf(
		"list panels scope=%s: pagination exceeded safety limit %d",
		scope,
		maxPanelPageFetches,
	)
}

// validateSpecificPanelDetail 确保 Detail 确实属于 List 中的同一个
// specific Panel，并且关联对象字段完整返回。
//
// FetchFullState 会用于自动备份和完整恢复，因此这里宁可失败，
// 也不能生成一个看起来完整、实际丢失 targets 的快照。
func validateSpecificPanelDetail(
	base *dto.PanelRecord,
	detail *dto.PanelRecord,
) error {
	if base == nil || detail == nil {
		return fmt.Errorf(
			"specific panel detail is nil",
		)
	}

	if detail.Scope != "" &&
		detail.Scope != base.Scope {
		return fmt.Errorf(
			"panel_id=%s detail scope mismatch: list=%s detail=%s",
			base.PanelID,
			base.Scope,
			detail.Scope,
		)
	}

	if detail.TargetType != "" &&
		detail.TargetType != base.TargetType {
		return fmt.Errorf(
			"panel_id=%s detail target_type mismatch: list=%s detail=%s",
			base.PanelID,
			base.TargetType,
			detail.TargetType,
		)
	}

	switch base.Scope {
	case dto.PanelScopeC2C:
		if detail.UserOpenIDs == nil {
			return fmt.Errorf(
				"panel_id=%s c2c specific detail missing user_openids",
				base.PanelID,
			)
		}

	case dto.PanelScopeGroup:
		if detail.GroupOpenIDs == nil {
			return fmt.Errorf(
				"panel_id=%s group specific detail missing group_openids",
				base.PanelID,
			)
		}

	default:
		return fmt.Errorf(
			"panel_id=%s specific panel has unsupported scope=%s",
			base.PanelID,
			base.Scope,
		)
	}

	return nil
}

// mergePanelDetail 把 Detail 接口返回的信息合并回 List 记录.
//
// 不直接用 detail 替换 base，是为了防止平台 Detail 将来省略
// 某些 List 中已经存在的字段。
// slices 使用 nil 判断，可以区分“字段未返回”和“明确返回空数组”。
func mergePanelDetail(base, detail *dto.PanelRecord) {
	if base == nil || detail == nil {
		return
	}

	if detail.PanelID != "" {
		base.PanelID = detail.PanelID
	}

	if detail.Scope != "" {
		base.Scope = detail.Scope
	}

	if detail.TargetType != "" {
		base.TargetType = detail.TargetType
	}

	if detail.Panel != nil {
		base.Panel = detail.Panel
	}

	if detail.UserOpenIDs != nil {
		base.UserOpenIDs = detail.UserOpenIDs
	}

	if detail.GroupOpenIDs != nil {
		base.GroupOpenIDs = detail.GroupOpenIDs
	}

	if detail.CreatedAt != "" {
		base.CreatedAt = detail.CreatedAt
	}

	if detail.UpdatedAt != "" {
		base.UpdatedAt = detail.UpdatedAt
	}

	if detail.Version != 0 {
		base.Version = detail.Version
	}
}
