package commandpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/tencent-connect/botgo/dto"
)

const (
	maxRemotePanels = 20
	maxPanelTargets = 1000

	panelBusyRetryCount = 3
	panelBusyRetryDelay = 800 * time.Millisecond
)

// ApplyResult 表示一次配置应用结果.
type ApplyResult struct {
	BackupName string `json:"backup_name,omitempty"`

	MenuUpdated bool `json:"menu_updated"`

	PanelsCreated int `json:"panels_created"`
	PanelsUpdated int `json:"panels_updated"`
	PanelsDeleted int `json:"panels_deleted"`
}

// ApplyConfig 应用普通配置.
//
// 普通 Apply 只管理：
//   - 全局自定义 Menu
//   - target_type=all 的 Panel
//
// specific Panel 只读取、不修改、不删除。
// 这样同一份配置可以安全复制到其他 Bot 使用。
func (m *Manager) ApplyConfig(
	ctx context.Context,
	storage *Storage,
	currentAppID string,
	config *ConfigFile,
) (*ApplyResult, error) {
	if m == nil || m.api == nil {
		return nil, fmt.Errorf("commandpanel api is nil")
	}

	if storage == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf(
			"validate config: %w",
			err,
		)
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	// 必须读取完整状态，因为自动备份必须包含 specific targets。
	before, err := m.FetchFullState(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"fetch remote state before apply: %w",
			err,
		)
	}

	if err := validateRemoteGlobalPanels(before); err != nil {
		return nil, err
	}

	if err := checkApplyCapacity(
		before,
		config,
	); err != nil {
		return nil, err
	}

	backup, err := storage.BackupRemoteState(
		currentAppID,
		before,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create automatic backup: %w",
			err,
		)
	}

	result := &ApplyResult{
		BackupName: backup.Name,
	}

	// Menu
	if !sameMenu(before.Menu, config.Menu) {
		if err := m.api.UpdateGlobalMenu(
			ctx,
			menuForWrite(config.Menu),
		); err != nil {
			return result, applyError(
				result,
				"update global menu",
				err,
			)
		}

		result.MenuUpdated = true
	}

	currentGlobals := globalPanelsByScope(
		before.Panels,
	)

	desiredGlobals := globalPanelsByScope(
		config.Panels,
	)

	// 固定顺序，保证行为可预测。
	// 第一阶段：更新仍然存在的 global Panel。
	for _, scope := range panelScopes {
		current := currentGlobals[scope]
		desired := desiredGlobals[scope]

		if current == nil || desired == nil {
			continue
		}

		if samePanel(current.Panel, desired.Panel) {
			continue
		}

		if err := m.updatePanelWithBusyRetry(
			ctx,
			current.PanelID,
			desired.Panel,
		); err != nil {
			return result, applyError(
				result,
				"update global panel scope="+scope,
				err,
			)
		}

		result.PanelsUpdated++
	}

	// 第二阶段：先删除目标配置里已经不存在的 global Panel。
	// 必须先删后建，避免远端当前恰好达到 Panel 上限时，
	// 创建新 Panel 短暂超过上限。
	for _, scope := range panelScopes {
		current := currentGlobals[scope]
		desired := desiredGlobals[scope]

		if current == nil || desired != nil {
			continue
		}

		if err := m.deletePanelWithBusyRetry(
			ctx,
			current.PanelID,
		); err != nil {
			return result, applyError(
				result,
				"delete global panel scope="+scope,
				err,
			)
		}

		result.PanelsDeleted++
	}

	// 第三阶段：最后创建新的 global Panel。
	for _, scope := range panelScopes {
		current := currentGlobals[scope]
		desired := desiredGlobals[scope]

		if current != nil || desired == nil {
			continue
		}

		_, err := m.createPanelWithBusyRetry(
			ctx,
			&dto.PanelToCreate{
				Scope:      scope,
				TargetType: dto.PanelTargetTypeAll,
				Panel:      desired.Panel,
			},
		)
		if err != nil {
			return result, applyError(
				result,
				"create global panel scope="+scope,
				err,
			)
		}

		result.PanelsCreated++
	}

	// 写完以后一定重新 GET。
	after, err := m.FetchOverview(ctx)
	if err != nil {
		return result, applyError(
			result,
			"fetch remote state after apply",
			err,
		)
	}

	if err := verifyNormalApply(
		after,
		config,
	); err != nil {
		return result, applyError(
			result,
			"verify applied configuration",
			err,
		)
	}

	return result, nil
}

// ValidateConfig 对统一 JSON 配置执行 QQ 业务级校验.
//
// Storage 只负责 JSON/文件系统安全。
// QQ 的字段和业务规则统一放在这里。
func ValidateConfig(config *ConfigFile) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if config.Format != ConfigFormat {
		return fmt.Errorf(
			"unsupported config format %q",
			config.Format,
		)
	}

	if config.SchemaVersion != ConfigSchemaVersion {
		return fmt.Errorf(
			"unsupported schema_version %d",
			config.SchemaVersion,
		)
	}

	if err := validateMenu(config.Menu); err != nil {
		return err
	}

	if len(config.Panels) > maxRemotePanels {
		return fmt.Errorf(
			"panels exceeds platform limit: %d > %d",
			len(config.Panels),
			maxRemotePanels,
		)
	}

	globalScopes := make(
		map[string]struct{},
	)

	for i, record := range config.Panels {
		if err := validatePanelRecord(
			record,
		); err != nil {
			return fmt.Errorf(
				"panels[%d]: %w",
				i,
				err,
			)
		}

		if record.TargetType ==
			dto.PanelTargetTypeAll {

			if _, exists := globalScopes[record.Scope]; exists {
				return fmt.Errorf(
					"scope %s contains more than one global panel",
					record.Scope,
				)
			}

			globalScopes[record.Scope] =
				struct{}{}
		}
	}

	return nil
}

func validateMenu(menu *dto.Menu) error {
	// nil 表示没有全局菜单 / 清空菜单。
	if menu == nil {
		return nil
	}

	if len(menu.Items) > 10 {
		return fmt.Errorf(
			"menu.items exceeds limit: %d > 10",
			len(menu.Items),
		)
	}

	for i, item := range menu.Items {
		if err := validateMenuItem(
			item,
			false,
		); err != nil {
			return fmt.Errorf(
				"menu.items[%d]: %w",
				i,
				err,
			)
		}
	}

	return nil
}

func validateMenuItem(
	item *dto.MenuItem,
	child bool,
) error {
	if item == nil {
		return fmt.Errorf("menu item is nil")
	}

	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf(
			"name is empty",
		)
	}

	if weightedLength(item.Name) > 10 {
		return fmt.Errorf(
			"name exceeds weighted length 10",
		)
	}

	switch item.Type {
	case dto.MenuItemTypeSendMessage:
		if strings.TrimSpace(
			item.SendMessage,
		) == "" {
			return fmt.Errorf(
				"send_message is empty",
			)
		}

	case dto.MenuItemTypeLink:
		if !validHTTPSURL(item.Link) {
			return fmt.Errorf(
				"link must be a valid https URL",
			)
		}

	case dto.MenuItemTypeSwitch:
		if child {
			return fmt.Errorf(
				"submenu does not support switch",
			)
		}

		if item.Switch == nil ||
			strings.TrimSpace(
				item.Switch.SwitchID,
			) == "" {
			return fmt.Errorf(
				"switch.switch_id is empty",
			)
		}

	case dto.MenuItemTypeMenu:
		if child {
			return fmt.Errorf(
				"submenu cannot contain nested menu",
			)
		}

		if len(item.SubMenuItems) > 5 {
			return fmt.Errorf(
				"sub_menu_items exceeds limit: %d > 5",
				len(item.SubMenuItems),
			)
		}

		for i, sub := range item.SubMenuItems {
			if err := validateMenuItem(
				sub,
				true,
			); err != nil {
				return fmt.Errorf(
					"sub_menu_items[%d]: %w",
					i,
					err,
				)
			}
		}

	default:
		return fmt.Errorf(
			"unsupported menu type %q",
			item.Type,
		)
	}

	return nil
}

func validatePanelRecord(
	record *dto.PanelRecord,
) error {
	if record == nil {
		return fmt.Errorf(
			"panel record is nil",
		)
	}

	if !validScope(record.Scope) {
		return fmt.Errorf(
			"invalid scope %q",
			record.Scope,
		)
	}

	switch record.TargetType {
	case dto.PanelTargetTypeAll:
		if len(record.UserOpenIDs) != 0 ||
			len(record.GroupOpenIDs) != 0 {
			return fmt.Errorf(
				"global panel must not contain targets",
			)
		}

	case dto.PanelTargetTypeSpecific:
		switch record.Scope {
		case dto.PanelScopeC2C:
			if len(record.GroupOpenIDs) != 0 {
				return fmt.Errorf(
					"c2c specific panel cannot contain group_openids",
				)
			}

			userTargets := uniqueStrings(record.UserOpenIDs)

			if len(userTargets) > maxPanelTargets {
				return fmt.Errorf(
					"user_openids exceeds limit %d",
					maxPanelTargets,
				)
			}

		case dto.PanelScopeGroup:
			if len(record.UserOpenIDs) != 0 {
				return fmt.Errorf(
					"group specific panel cannot contain user_openids",
				)
			}

			groupTargets := uniqueStrings(record.GroupOpenIDs)

			if len(groupTargets) > maxPanelTargets {
				return fmt.Errorf(
					"group_openids exceeds limit %d",
					maxPanelTargets,
				)
			}

		default:
			return fmt.Errorf(
				"scope %s does not support specific target_type",
				record.Scope,
			)
		}

	default:
		return fmt.Errorf(
			"invalid target_type %q",
			record.TargetType,
		)
	}

	return validatePanel(record.Panel)
}

func validatePanel(panel *dto.Panel) error {
	if panel == nil {
		return fmt.Errorf(
			"panel is nil",
		)
	}

	if len(panel.Items) > 20 {
		return fmt.Errorf(
			"panel.items exceeds limit: %d > 20",
			len(panel.Items),
		)
	}

	if len([]rune(panel.Remark)) > 255 {
		return fmt.Errorf(
			"panel.remark exceeds length 255",
		)
	}

	for i, item := range panel.Items {
		if item == nil {
			return fmt.Errorf(
				"panel.items[%d] is nil",
				i,
			)
		}

		if strings.TrimSpace(
			item.Name,
		) == "" {
			return fmt.Errorf(
				"panel.items[%d].name is empty",
				i,
			)
		}

		if weightedLength(
			item.Name,
		) > 14 {
			return fmt.Errorf(
				"panel.items[%d].name exceeds weighted length 14",
				i,
			)
		}

		if weightedLength(
			item.Desc,
		) > 30 {
			return fmt.Errorf(
				"panel.items[%d].desc exceeds weighted length 30",
				i,
			)
		}

		switch item.Type {
		case dto.PanelItemTypeCommand:
			// command 不要求 link.

		case dto.PanelItemTypeLink:
			if !validHTTPSURL(
				item.Link,
			) {
				return fmt.Errorf(
					"panel.items[%d].link must be a valid https URL",
					i,
				)
			}

		default:
			return fmt.Errorf(
				"panel.items[%d] has invalid type %q",
				i,
				item.Type,
			)
		}
	}

	return nil
}

func validScope(scope string) bool {
	switch scope {
	case dto.PanelScopeC2C,
		dto.PanelScopeGroup,
		dto.PanelScopeChannel,
		dto.PanelScopeDM:
		return true
	}

	return false
}

// weightedLength 腾讯文档中的字符权重:
//
// ASCII = 1
// 非 ASCII = 2
func weightedLength(value string) int {
	length := 0

	for _, r := range value {
		if r <= 0x7F {
			length++
		} else {
			length += 2
		}
	}

	return length
}

func validHTTPSURL(value string) bool {
	value = strings.TrimSpace(value)

	if value == "" {
		return false
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}

	return strings.EqualFold(
		parsed.Scheme,
		"https",
	) && parsed.Host != ""
}

// validateRemoteGlobalPanels 防止远端出现异常状态时
// 我们随便挑一个 Panel 修改。
func validateRemoteGlobalPanels(
	state *RemoteState,
) error {
	if state == nil {
		return fmt.Errorf("remote state is nil")
	}

	counts := make(map[string]int)

	for i, record := range state.Panels {
		if record == nil {
			continue
		}

		if record.TargetType != dto.PanelTargetTypeAll {
			continue
		}

		if !validScope(record.Scope) {
			return fmt.Errorf(
				"remote panels[%d] has invalid scope %q",
				i,
				record.Scope,
			)
		}

		if strings.TrimSpace(record.PanelID) == "" {
			return fmt.Errorf(
				"remote global panel scope=%s has empty panel_id",
				record.Scope,
			)
		}

		if record.Panel == nil {
			return fmt.Errorf(
				"remote global panel scope=%s has empty panel content",
				record.Scope,
			)
		}

		counts[record.Scope]++

		if counts[record.Scope] > 1 {
			return fmt.Errorf(
				"remote scope %s contains multiple global panels; refusing automatic apply",
				record.Scope,
			)
		}
	}

	return nil
}

// checkApplyCapacity 普通 Apply 绝不删除 specific Panel。
//
// 如果当前已经达到 20 个 Panel，配置又要求在一个此前没有
// all Panel 的 scope 新建 global panel，则提前失败，绝不先写 QQ。
func checkApplyCapacity(
	current *RemoteState,
	config *ConfigFile,
) error {
	currentGlobals := globalPanelsByScope(
		current.Panels,
	)

	desiredGlobals := globalPanelsByScope(
		config.Panels,
	)

	creates := 0
	deletes := 0

	for _, scope := range panelScopes {
		currentPanel := currentGlobals[scope]
		desiredPanel := desiredGlobals[scope]

		if currentPanel == nil &&
			desiredPanel != nil {
			creates++
		}

		if currentPanel != nil &&
			desiredPanel == nil {
			deletes++
		}
	}

	// 普通 Apply 可以先删除“确定要删除的 global”，
	// 再创建新的 global。
	projected := len(current.Panels) -
		deletes +
		creates

	if projected > maxRemotePanels {
		return fmt.Errorf(
			"apply would require %d panels, platform limit is %d; specific panels will not be deleted automatically",
			projected,
			maxRemotePanels,
		)
	}

	return nil
}

func globalPanelsByScope(
	records []*dto.PanelRecord,
) map[string]*dto.PanelRecord {
	result := make(
		map[string]*dto.PanelRecord,
	)

	for _, record := range records {
		if record == nil ||
			record.TargetType !=
				dto.PanelTargetTypeAll {
			continue
		}

		result[record.Scope] = record
	}

	return result
}

// menuForWrite 把“没有菜单”和“空菜单”统一转换成平台的清空语义.
//
// 比较层已经把 nil 与 items=[] 视为等价，
// 写入层也应保持一致。
func menuForWrite(menu *dto.Menu) *dto.Menu {
	if menu == nil || len(menu.Items) == 0 {
		return nil
	}

	return menu
}

func sameMenu(
	a *dto.Menu,
	b *dto.Menu,
) bool {
	return sameCanonicalJSON(
		normalizeMenu(a),
		normalizeMenu(b),
	)
}

func samePanel(
	a *dto.Panel,
	b *dto.Panel,
) bool {
	return sameCanonicalJSON(
		normalizePanel(a),
		normalizePanel(b),
	)
}

func normalizeMenu(menu *dto.Menu) *dto.Menu {
	if menu == nil {
		return &dto.Menu{
			Items: []*dto.MenuItem{},
		}
	}

	copyMenu := *menu

	if copyMenu.Items == nil {
		copyMenu.Items =
			[]*dto.MenuItem{}
	}

	return &copyMenu
}

func normalizePanel(
	panel *dto.Panel,
) *dto.Panel {
	if panel == nil {
		return nil
	}

	copyPanel := *panel

	if copyPanel.Items == nil {
		copyPanel.Items =
			[]*dto.PanelItem{}
	}

	return &copyPanel
}

func sameCanonicalJSON(
	a interface{},
	b interface{},
) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}

	right, err := json.Marshal(b)
	if err != nil {
		return false
	}

	return string(left) == string(right)
}

// verifyNormalApply 校验普通 Apply:
//
// 1. Menu 必须等于目标
// 2. target_type=all 的 Panels 必须等于目标
//
// specific Panel 不属于普通 Apply 的控制范围。
// 它可能由运行中的业务程序动态变化，因此这里不比较 specific。
//
// Panel ID/version/timestamp 不参与业务配置比较。
func verifyNormalApply(
	after *RemoteState,
	config *ConfigFile,
) error {
	if after == nil {
		return fmt.Errorf("remote state after apply is nil")
	}

	if !sameMenu(
		after.Menu,
		config.Menu,
	) {
		return fmt.Errorf(
			"remote menu does not match desired configuration",
		)
	}

	if !samePanelRecordSet(
		filterGlobalPanels(after.Panels),
		filterGlobalPanels(config.Panels),
		false,
	) {
		return fmt.Errorf(
			"remote global panels do not match desired configuration",
		)
	}

	return nil
}

func filterGlobalPanels(
	records []*dto.PanelRecord,
) []*dto.PanelRecord {
	result := make(
		[]*dto.PanelRecord,
		0,
	)

	for _, record := range records {
		if record != nil &&
			record.TargetType ==
				dto.PanelTargetTypeAll {
			result = append(
				result,
				record,
			)
		}
	}

	return result
}

func filterSpecificPanels(
	records []*dto.PanelRecord,
) []*dto.PanelRecord {
	result := make(
		[]*dto.PanelRecord,
		0,
	)

	for _, record := range records {
		if record != nil &&
			record.TargetType ==
				dto.PanelTargetTypeSpecific {
			result = append(
				result,
				record,
			)
		}
	}

	return result
}

func samePanelRecordSet(
	a []*dto.PanelRecord,
	b []*dto.PanelRecord,
	includePanelID bool,
) bool {
	left := panelSignatures(
		a,
		includePanelID,
	)

	right := panelSignatures(
		b,
		includePanelID,
	)

	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

func panelSignatures(
	records []*dto.PanelRecord,
	includePanelID bool,
) []string {
	signatures := make(
		[]string,
		0,
		len(records),
	)

	for _, record := range records {
		if record == nil {
			continue
		}

		value := struct {
			PanelID      string     `json:"panel_id,omitempty"`
			Scope        string     `json:"scope"`
			TargetType   string     `json:"target_type"`
			Panel        *dto.Panel `json:"panel"`
			UserOpenIDs  []string   `json:"user_openids,omitempty"`
			GroupOpenIDs []string   `json:"group_openids,omitempty"`
		}{
			Scope:      record.Scope,
			TargetType: record.TargetType,
			Panel:      normalizePanel(record.Panel),
		}

		if includePanelID {
			value.PanelID =
				record.PanelID
		}

		if record.TargetType ==
			dto.PanelTargetTypeSpecific {

			value.UserOpenIDs =
				uniqueStrings(
					record.UserOpenIDs,
				)

			value.GroupOpenIDs =
				uniqueStrings(
					record.GroupOpenIDs,
				)
		}

		data, _ := json.Marshal(value)

		signatures = append(
			signatures,
			string(data),
		)
	}

	sort.Strings(signatures)

	return signatures
}

func uniqueStrings(
	values []string,
) []string {
	set := make(map[string]struct{})

	for _, value := range values {
		value = strings.TrimSpace(value)

		if value == "" {
			continue
		}

		set[value] = struct{}{}
	}

	result := make(
		[]string,
		0,
		len(set),
	)

	for value := range set {
		result = append(
			result,
			value,
		)
	}

	sort.Strings(result)

	return result
}

func applyError(
	result *ApplyResult,
	action string,
	err error,
) error {
	if result != nil &&
		result.BackupName != "" {
		return fmt.Errorf(
			"%s: %w; remote state was backed up to %s",
			action,
			err,
			result.BackupName,
		)
	}

	return fmt.Errorf(
		"%s: %w",
		action,
		err,
	)
}

// 腾讯 botgo 当前把非 2xx 响应包装成 errs.Err，
// 其中 Error.Text() 保存的是平台响应 body。
// 所以 40030009 会出现在 err.Error() 文本中。
//
// 现有 botgo 的错误结构就是这种设计。
func isPanelBusyError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(
		err.Error(),
		"40030009",
	)
}

func waitContext(
	ctx context.Context,
	duration time.Duration,
) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		return nil
	}
}

func (m *Manager) updatePanelWithBusyRetry(
	ctx context.Context,
	panelID string,
	panel *dto.Panel,
) error {
	var lastErr error

	for attempt := 0; attempt < panelBusyRetryCount; attempt++ {

		lastErr = m.api.UpdatePanel(
			ctx,
			panelID,
			panel,
		)

		if lastErr == nil {
			return nil
		}

		if !isPanelBusyError(lastErr) {
			return lastErr
		}

		if attempt+1 >=
			panelBusyRetryCount {
			break
		}

		if err := waitContext(
			ctx,
			panelBusyRetryDelay,
		); err != nil {
			return err
		}
	}

	return lastErr
}

func (m *Manager) deletePanelWithBusyRetry(
	ctx context.Context,
	panelID string,
) error {
	var lastErr error

	for attempt := 0; attempt < panelBusyRetryCount; attempt++ {

		lastErr = m.api.DeletePanel(
			ctx,
			panelID,
		)

		if lastErr == nil {
			return nil
		}

		if !isPanelBusyError(lastErr) {
			return lastErr
		}

		if attempt+1 >=
			panelBusyRetryCount {
			break
		}

		if err := waitContext(
			ctx,
			panelBusyRetryDelay,
		); err != nil {
			return err
		}
	}

	return lastErr
}

func (m *Manager) createPanelWithBusyRetry(
	ctx context.Context,
	value *dto.PanelToCreate,
) (*dto.PanelCreateResponse, error) {
	var (
		result  *dto.PanelCreateResponse
		lastErr error
	)

	for attempt := 0; attempt < panelBusyRetryCount; attempt++ {

		result, lastErr =
			m.api.CreatePanel(
				ctx,
				value,
			)

		if lastErr == nil {
			if result == nil ||
				strings.TrimSpace(result.PanelID) == "" {
				return nil, fmt.Errorf(
					"create panel returned empty panel_id",
				)
			}

			return result, nil
		}

		if !isPanelBusyError(lastErr) {
			return nil, lastErr
		}

		if attempt+1 >=
			panelBusyRetryCount {
			break
		}

		if err := waitContext(
			ctx,
			panelBusyRetryDelay,
		); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}
