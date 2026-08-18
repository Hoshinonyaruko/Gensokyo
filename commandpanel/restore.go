package commandpanel

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tencent-connect/botgo/dto"
)

const (
	panelTargetBatchSize = 20
)

// Full Restore 可能连续操作大量 Panel。
// 腾讯 create/update/delete 为 10 QPM，target 为 60 QPM。
//
// 使用变量而不是 const，方便单元测试把间隔临时设为 0。
var (
	restorePanelWriteInterval  = 6100 * time.Millisecond
	restoreTargetWriteInterval = 1050 * time.Millisecond
)

// RestoreResult 表示一次完整恢复的执行结果.
type RestoreResult struct {
	BackupName string `json:"backup_name,omitempty"`

	MenuUpdated bool `json:"menu_updated"`

	PanelsReused  int `json:"panels_reused"`
	PanelsCreated int `json:"panels_created"`
	PanelsUpdated int `json:"panels_updated"`
	PanelsDeleted int `json:"panels_deleted"`

	TargetsAdded   int `json:"targets_added"`
	TargetsDeleted int `json:"targets_deleted"`
}

// restoreBinding 表示目标配置中的一个 Panel
// 最终对应哪个当前远端 Panel.
//
// current == nil 表示需要创建。
type restoreBinding struct {
	desired *dto.PanelRecord
	current *dto.PanelRecord
	created bool
}

// restorePlan 是任何 QQ 写操作发生之前计算出的完整计划.
type restorePlan struct {
	bindings []*restoreBinding
	extras   []*dto.PanelRecord
}

func (p *restorePlan) createCount() int {
	count := 0

	for _, binding := range p.bindings {
		if binding.current == nil {
			count++
		}
	}

	return count
}

// RestoreConfig 把同一个 Bot 的历史配置完整恢复.
//
// 与普通 ApplyConfig 不同：
//
//	ApplyConfig
//	    只管理 Menu + all Panels
//
//	RestoreConfig
//	    管理 Menu + all + specific + targets
//
// Full Restore 是显式的高影响操作，因此只接受
// source_appid 与当前 Bot AppID 完全一致的配置。
func (m *Manager) RestoreConfig(
	ctx context.Context,
	storage *Storage,
	currentAppID string,
	config *ConfigFile,
) (*RestoreResult, error) {
	if m == nil || m.api == nil {
		return nil, fmt.Errorf("commandpanel api is nil")
	}

	if storage == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	currentAppID = strings.TrimSpace(currentAppID)
	sourceAppID := strings.TrimSpace(config.SourceAppID)

	if currentAppID == "" {
		return nil, fmt.Errorf("current appid is empty")
	}

	if sourceAppID == "" {
		return nil, fmt.Errorf(
			"full restore requires source_appid",
		)
	}

	if sourceAppID != currentAppID {
		return nil, fmt.Errorf(
			"full restore source_appid mismatch: config=%s current=%s",
			sourceAppID,
			currentAppID,
		)
	}

	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf(
			"validate restore config: %w",
			err,
		)
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	// Full Restore 必须先拿到完整 specific targets。
	before, err := m.FetchFullState(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"fetch remote state before restore: %w",
			err,
		)
	}

	// 所有匹配、容量判断必须在第一个 QQ 写操作之前完成。
	plan, err := buildRestorePlan(
		before.Panels,
		config.Panels,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"build restore plan: %w",
			err,
		)
	}

	createCount := plan.createCount()

	// 如果远端当前已经接近 20 个 Panel，
	// 创建历史配置缺失 Panel 之前可能必须释放槽位。
	//
	// 只允许预删除已经被计划明确判定为 extra 的资源，
	// 不会为了腾位置删除任何目标 Panel。
	requiredPreDeletes :=
		len(before.Panels) +
			createCount -
			maxRemotePanels

	if requiredPreDeletes < 0 {
		requiredPreDeletes = 0
	}

	if requiredPreDeletes > len(plan.extras) {
		return nil, fmt.Errorf(
			"restore requires %d free panel slots but only %d removable extra panels exist",
			requiredPreDeletes,
			len(plan.extras),
		)
	}

	// 任何 QQ mutation 之前，自动备份必须成功。
	backup, err := storage.BackupRemoteState(
		currentAppID,
		before,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create automatic backup before restore: %w",
			err,
		)
	}

	result := &RestoreResult{
		BackupName: backup.Name,
	}

	for _, binding := range plan.bindings {
		if binding.current != nil {
			result.PanelsReused++
		}
	}

	pacer := &restoreWritePacer{}

	sort.Slice(
		plan.extras,
		func(i, j int) bool {
			return plan.extras[i].PanelID <
				plan.extras[j].PanelID
		},
	)

	// ------------------------------------------------
	// 第一阶段：
	// 仅当 20 Panel 上限确实挡住创建时，
	// 最少量删除已经确定属于 extra 的 Panel。
	// ------------------------------------------------

	preDeleted := make(
		map[string]struct{},
		requiredPreDeletes,
	)

	for i := 0; i < requiredPreDeletes; i++ {
		record := plan.extras[i]

		if err := m.restoreDeletePanel(
			ctx,
			pacer,
			record.PanelID,
		); err != nil {
			return result, restoreError(
				result,
				"pre-delete extra panel "+record.PanelID,
				err,
			)
		}

		preDeleted[record.PanelID] = struct{}{}
		result.PanelsDeleted++
	}

	// ------------------------------------------------
	// 第二阶段：
	// 先创建所有缺失的目标 Panel。
	//
	// specific 创建时最多带前 20 个 target，
	// 剩余 target 后面再批量追加。
	// ------------------------------------------------

	for _, binding := range plan.bindings {
		if binding.current != nil {
			continue
		}

		current, err := m.restoreCreateDesiredPanel(
			ctx,
			pacer,
			binding.desired,
		)
		if err != nil {
			return result, restoreError(
				result,
				"create desired panel scope="+binding.desired.Scope,
				err,
			)
		}

		binding.current = current
		binding.created = true
		result.PanelsCreated++
	}

	// ------------------------------------------------
	// 第三阶段：
	// 修复/更新已经存在 Panel 的内容。
	// ------------------------------------------------

	for _, binding := range plan.bindings {
		if binding.created {
			continue
		}

		if samePanel(
			binding.current.Panel,
			binding.desired.Panel,
		) {
			continue
		}

		if err := m.restoreUpdatePanel(
			ctx,
			pacer,
			binding.current.PanelID,
			binding.desired.Panel,
		); err != nil {
			return result, restoreError(
				result,
				"update panel "+binding.current.PanelID,
				err,
			)
		}

		binding.current.Panel =
			binding.desired.Panel

		result.PanelsUpdated++
	}

	// ------------------------------------------------
	// 第四阶段：
	// 恢复 specific targets。
	//
	// 使用集合差：
	//
	//	current - desired = 删除
	//	desired - current = 添加
	//
	// 每次最多 20 个。
	// 接近 1000 容量时，只预删“必须先删”的最少数量。
	// ------------------------------------------------

	for _, binding := range plan.bindings {
		if binding.desired.TargetType !=
			dto.PanelTargetTypeSpecific {
			continue
		}

		added, deleted, err :=
			m.restoreSpecificTargets(
				ctx,
				pacer,
				binding.current,
				binding.desired,
			)
		if err != nil {
			return result, restoreError(
				result,
				"restore targets panel="+binding.current.PanelID,
				err,
			)
		}

		result.TargetsAdded += added
		result.TargetsDeleted += deleted
	}

	// ------------------------------------------------
	// 第五阶段：
	// 所有目标 Panel 都已经存在后，
	// 删除剩余历史配置不需要的旧 Panel。
	// ------------------------------------------------

	for _, record := range plan.extras {
		if _, alreadyDeleted :=
			preDeleted[record.PanelID]; alreadyDeleted {
			continue
		}

		if err := m.restoreDeletePanel(
			ctx,
			pacer,
			record.PanelID,
		); err != nil {
			return result, restoreError(
				result,
				"delete extra panel "+record.PanelID,
				err,
			)
		}

		result.PanelsDeleted++
	}

	// ------------------------------------------------
	// 第六阶段：
	// 最后恢复 Menu。
	//
	// 再 GET 一次，而不是只相信 restore 开始前的 Menu，
	// 防止恢复过程中其他业务恰好修改 Menu。
	// ------------------------------------------------

	remoteMenu, err := m.api.GetGlobalMenu(ctx)
	if err != nil {
		return result, restoreError(
			result,
			"get global menu before final restore",
			err,
		)
	}

	if !sameMenu(
		remoteMenu,
		config.Menu,
	) {
		if err := m.api.UpdateGlobalMenu(
			ctx,
			menuForWrite(config.Menu),
		); err != nil {
			return result, restoreError(
				result,
				"restore global menu",
				err,
			)
		}

		result.MenuUpdated = true
	}

	// ------------------------------------------------
	// 第七阶段：
	// 最终一定重新读取平台完整真实状态。
	// ------------------------------------------------

	after, err := m.FetchFullState(ctx)
	if err != nil {
		return result, restoreError(
			result,
			"fetch remote state after restore",
			err,
		)
	}

	if err := verifyFullRestore(
		after,
		config,
	); err != nil {
		return result, restoreError(
			result,
			"verify full restore",
			err,
		)
	}

	return result, nil
}

// buildRestorePlan 在完全不写 QQ 的前提下，
// 将历史配置中的每个 desired Panel 与当前 Panel 对应起来.
//
// 匹配优先级:
//
//  1. 历史 panel_id 当前仍存在 -> 直接复用
//  2. global/all -> 同 scope 的当前 global
//  3. specific -> 完整业务语义完全相同的当前 specific
//  4. 仍无法匹配 -> 创建新 Panel
//
// 不会仅仅因为两个 specific Panel 都属于 group
// 就随便把它们视为同一个资源。
func buildRestorePlan(
	current []*dto.PanelRecord,
	desired []*dto.PanelRecord,
) (*restorePlan, error) {
	currentByID := make(
		map[string]*dto.PanelRecord,
		len(current),
	)

	for _, record := range current {
		if record == nil {
			continue
		}

		panelID := strings.TrimSpace(
			record.PanelID,
		)

		if panelID == "" {
			return nil, fmt.Errorf(
				"current panel has empty panel_id",
			)
		}

		if _, exists := currentByID[
			panelID]; exists {
			return nil, fmt.Errorf(
				"duplicate current panel_id %s",
				panelID,
			)
		}

		currentByID[panelID] = record
	}

	plan := &restorePlan{
		bindings: make(
			[]*restoreBinding,
			0,
			len(desired),
		),
		extras: make(
			[]*dto.PanelRecord,
			0,
		),
	}

	for _, record := range desired {
		plan.bindings = append(
			plan.bindings,
			&restoreBinding{
				desired: record,
			},
		)
	}

	usedCurrent := make(
		map[string]struct{},
	)

	desiredIDs := make(
		map[string]struct{},
	)

	// Pass 1:
	// 优先使用历史保存下来的 panel_id。
	for _, binding := range plan.bindings {
		desiredID := strings.TrimSpace(
			binding.desired.PanelID,
		)

		if desiredID == "" {
			continue
		}

		if _, exists := desiredIDs[desiredID]; exists {
			return nil, fmt.Errorf(
				"duplicate desired panel_id %s",
				desiredID,
			)
		}

		desiredIDs[desiredID] =
			struct{}{}

		currentRecord, exists :=
			currentByID[desiredID]

		if !exists {
			continue
		}

		if currentRecord.Scope !=
			binding.desired.Scope ||
			currentRecord.TargetType !=
				binding.desired.TargetType {

			return nil, fmt.Errorf(
				"panel_id %s exists remotely but scope/target_type changed: current=%s/%s desired=%s/%s",
				desiredID,
				currentRecord.Scope,
				currentRecord.TargetType,
				binding.desired.Scope,
				binding.desired.TargetType,
			)
		}

		binding.current =
			currentRecord

		usedCurrent[currentRecord.PanelID] =
			struct{}{}
	}

	// Pass 2:
	// global Panel 的业务身份是 scope。
	//
	// 即使历史 panel_id 已经不存在，
	// 当前同 scope 的 global 仍可以安全复用。
	for _, binding := range plan.bindings {
		if binding.current != nil ||
			binding.desired.TargetType !=
				dto.PanelTargetTypeAll {
			continue
		}

		candidates :=
			unusedPanelCandidates(
				current,
				usedCurrent,
				binding.desired.Scope,
				dto.PanelTargetTypeAll,
			)

		if len(candidates) == 0 {
			continue
		}

		// 如果其中有内容已经完全相同的，
		// 优先复用，减少无意义 UPDATE。
		chosen := candidates[0]

		for _, candidate :=
			range candidates {

			if samePanel(
				candidate.Panel,
				binding.desired.Panel,
			) {
				chosen = candidate
				break
			}
		}

		binding.current = chosen
		usedCurrent[chosen.PanelID] =
			struct{}{}
	}

	// Pass 3:
	// specific 没有“同 scope 就是同一个”的概念。
	//
	// panel_id 不存在时，只复用业务语义已经完全相同的 Panel。
	// 否则创建新的，再把旧的作为 extra 最后删除。
	for _, binding := range plan.bindings {
		if binding.current != nil ||
			binding.desired.TargetType !=
				dto.PanelTargetTypeSpecific {
			continue
		}

		candidates :=
			unusedPanelCandidates(
				current,
				usedCurrent,
				binding.desired.Scope,
				dto.PanelTargetTypeSpecific,
			)

		for _, candidate :=
			range candidates {

			if !sameSpecificPanel(
				candidate,
				binding.desired,
			) {
				continue
			}

			binding.current =
				candidate

			usedCurrent[candidate.PanelID] =
				struct{}{}

			break
		}
	}

	for _, record := range current {
		if record == nil {
			continue
		}

		if _, used :=
			usedCurrent[record.PanelID]; used {
			continue
		}

		plan.extras = append(
			plan.extras,
			record,
		)
	}

	return plan, nil
}

func unusedPanelCandidates(
	current []*dto.PanelRecord,
	used map[string]struct{},
	scope string,
	targetType string,
) []*dto.PanelRecord {
	result := make(
		[]*dto.PanelRecord,
		0,
	)

	for _, record := range current {
		if record == nil {
			continue
		}

		if _, exists :=
			used[record.PanelID]; exists {
			continue
		}

		if record.Scope != scope ||
			record.TargetType != targetType {
			continue
		}

		result = append(
			result,
			record,
		)
	}

	sort.Slice(
		result,
		func(i, j int) bool {
			return result[i].PanelID <
				result[j].PanelID
		},
	)

	return result
}

func sameSpecificPanel(
	a *dto.PanelRecord,
	b *dto.PanelRecord,
) bool {
	if a == nil || b == nil {
		return false
	}

	if a.Scope != b.Scope ||
		a.TargetType !=
			dto.PanelTargetTypeSpecific ||
		b.TargetType !=
			dto.PanelTargetTypeSpecific {
		return false
	}

	if !samePanel(
		a.Panel,
		b.Panel,
	) {
		return false
	}

	return sameStringSet(
		panelTargetValues(a),
		panelTargetValues(b),
	)
}

func sameStringSet(
	a []string,
	b []string,
) bool {
	left := uniqueStrings(a)
	right := uniqueStrings(b)

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

// restoreCreateDesiredPanel 创建一个历史配置中缺失的 Panel.
//
// specific Panel 创建接口单次最多携带 20 个 target。
// 超出部分由 restoreSpecificTargets 后续添加。
func (m *Manager) restoreCreateDesiredPanel(
	ctx context.Context,
	pacer *restoreWritePacer,
	desired *dto.PanelRecord,
) (*dto.PanelRecord, error) {
	if desired == nil {
		return nil, fmt.Errorf(
			"desired panel is nil",
		)
	}

	request := &dto.PanelToCreate{
		Scope:      desired.Scope,
		TargetType: desired.TargetType,
		Panel:      desired.Panel,
	}

	initialTargets := []string{}

	if desired.TargetType ==
		dto.PanelTargetTypeSpecific {

		targets := panelTargetValues(
			desired,
		)

		if len(targets) >
			panelTargetBatchSize {
			initialTargets =
				append(
					[]string(nil),
					targets[:panelTargetBatchSize]...,
				)
		} else {
			initialTargets =
				append(
					[]string(nil),
					targets...,
				)
		}

		switch desired.Scope {
		case dto.PanelScopeC2C:
			request.UserOpenIDs =
				initialTargets

		case dto.PanelScopeGroup:
			request.GroupOpenIDs =
				initialTargets

		default:
			return nil, fmt.Errorf(
				"specific panel scope %s is unsupported",
				desired.Scope,
			)
		}
	}

	response, err := m.restoreCreatePanel(
		ctx,
		pacer,
		request,
	)
	if err != nil {
		return nil, err
	}

	record := &dto.PanelRecord{
		PanelID:    response.PanelID,
		Scope:      desired.Scope,
		TargetType: desired.TargetType,
		Panel:      desired.Panel,
	}

	if desired.TargetType ==
		dto.PanelTargetTypeSpecific {

		switch desired.Scope {
		case dto.PanelScopeC2C:
			record.UserOpenIDs =
				append(
					[]string(nil),
					initialTargets...,
				)

		case dto.PanelScopeGroup:
			record.GroupOpenIDs =
				append(
					[]string(nil),
					initialTargets...,
				)
		}
	}

	return record, nil
}

// restoreSpecificTargets 把当前 specific Panel 的 target
// 恢复成 desired target 集合.
//
// 容量感知算法:
//
// 如果 current + adds <= 1000:
//     ADD 全部缺失 target
//     DEL 全部多余 target
//
// 如果会临时超过 1000:
//     先 DEL 恰好足够腾空间的最少 target
//     ADD
//     再 DEL 剩余多余 target
func (m *Manager) restoreSpecificTargets(
	ctx context.Context,
	pacer *restoreWritePacer,
	current *dto.PanelRecord,
	desired *dto.PanelRecord,
) (added int, deleted int, err error) {
	if current == nil || desired == nil {
		return 0, 0, fmt.Errorf(
			"specific panel is nil",
		)
	}

	currentTargets :=
		panelTargetValues(current)

	desiredTargets :=
		panelTargetValues(desired)

	if len(currentTargets) >
		maxPanelTargets {
		return 0, 0, fmt.Errorf(
			"current targets exceed limit: %d > %d",
			len(currentTargets),
			maxPanelTargets,
		)
	}

	if len(desiredTargets) >
		maxPanelTargets {
		return 0, 0, fmt.Errorf(
			"desired targets exceed limit: %d > %d",
			len(desiredTargets),
			maxPanelTargets,
		)
	}

	toAdd := stringDifference(
		desiredTargets,
		currentTargets,
	)

	toDelete := stringDifference(
		currentTargets,
		desiredTargets,
	)

	// 如果先把所有新增对象加进去会超过 1000，
	// 只提前删除恰好足够腾空间的数量。
	needFree :=
		len(currentTargets) +
			len(toAdd) -
			maxPanelTargets

	if needFree < 0 {
		needFree = 0
	}

	if needFree > len(toDelete) {
		return 0, 0, fmt.Errorf(
			"cannot free enough target capacity: need=%d removable=%d",
			needFree,
			len(toDelete),
		)
	}

	if needFree > 0 {
		preDelete :=
			toDelete[:needFree]

		if err := m.restoreTargetBatches(
			ctx,
			pacer,
			current.PanelID,
			current.Scope,
			dto.PanelTargetOpDel,
			preDelete,
		); err != nil {
			return added, deleted, err
		}

		deleted += len(preDelete)
		toDelete =
			toDelete[needFree:]
	}

	if len(toAdd) > 0 {
		if err := m.restoreTargetBatches(
			ctx,
			pacer,
			current.PanelID,
			current.Scope,
			dto.PanelTargetOpAdd,
			toAdd,
		); err != nil {
			return added, deleted, err
		}

		added += len(toAdd)
	}

	if len(toDelete) > 0 {
		if err := m.restoreTargetBatches(
			ctx,
			pacer,
			current.PanelID,
			current.Scope,
			dto.PanelTargetOpDel,
			toDelete,
		); err != nil {
			return added, deleted, err
		}

		deleted += len(toDelete)
	}

	// 本地同步 current，
	// 仅用于本次 restore 后续逻辑。
	switch current.Scope {
	case dto.PanelScopeC2C:
		current.UserOpenIDs =
			append(
				[]string(nil),
				desiredTargets...,
			)

	case dto.PanelScopeGroup:
		current.GroupOpenIDs =
			append(
				[]string(nil),
				desiredTargets...,
			)
	}

	return added, deleted, nil
}

func (m *Manager) restoreTargetBatches(
	ctx context.Context,
	pacer *restoreWritePacer,
	panelID string,
	scope string,
	op string,
	targets []string,
) error {
	for start := 0;
		start < len(targets);
		start += panelTargetBatchSize {

		end :=
			start + panelTargetBatchSize

		if end > len(targets) {
			end = len(targets)
		}

		batch := append(
			[]string(nil),
			targets[start:end]...,
		)

		request :=
			&dto.PanelTargetToUpdate{
				Op: op,
			}

		switch scope {
		case dto.PanelScopeC2C:
			request.UserOpenIDs =
				batch

		case dto.PanelScopeGroup:
			request.GroupOpenIDs =
				batch

		default:
			return fmt.Errorf(
				"scope %s does not support specific targets",
				scope,
			)
		}

		if err := m.restoreUpdateTargets(
			ctx,
			pacer,
			panelID,
			request,
		); err != nil {
			return fmt.Errorf(
				"target op=%s batch=%d-%d: %w",
				op,
				start,
				end,
				err,
			)
		}
	}

	return nil
}

func panelTargetValues(
	record *dto.PanelRecord,
) []string {
	if record == nil {
		return []string{}
	}

	switch record.Scope {
	case dto.PanelScopeC2C:
		return uniqueStrings(
			record.UserOpenIDs,
		)

	case dto.PanelScopeGroup:
		return uniqueStrings(
			record.GroupOpenIDs,
		)
	}

	return []string{}
}

func stringDifference(
	left []string,
	right []string,
) []string {
	rightSet := make(
		map[string]struct{},
	)

	for _, value :=
		range uniqueStrings(right) {
		rightSet[value] = struct{}{}
	}

	result := make(
		[]string,
		0,
	)

	for _, value :=
		range uniqueStrings(left) {

		if _, exists :=
			rightSet[value]; exists {
			continue
		}

		result = append(
			result,
			value,
		)
	}

	sort.Strings(result)

	return result
}

// verifyFullRestore 做完整业务语义比较.
//
// 明确忽略:
//
//	panel_id
//	version
//	created_at
//	updated_at
//
// 因为“恢复成功”的定义是业务状态相同，
// 而不是要求腾讯重新生成相同资源 ID。
func verifyFullRestore(
	after *RemoteState,
	config *ConfigFile,
) error {
	if after == nil {
		return fmt.Errorf(
			"remote state after restore is nil",
		)
	}

	if !sameMenu(
		after.Menu,
		config.Menu,
	) {
		return fmt.Errorf(
			"remote menu does not match restore configuration",
		)
	}

	if !samePanelRecordSet(
		after.Panels,
		config.Panels,
		false,
	) {
		return fmt.Errorf(
			"remote panels do not match restore configuration",
		)
	}

	return nil
}

func restoreError(
	result *RestoreResult,
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

// ----------------------------------------------------
// Full Restore QPM pacing
// ----------------------------------------------------

type restoreWritePacer struct {
	lastCreate time.Time
	lastUpdate time.Time
	lastDelete time.Time
	lastTarget time.Time
}

func (p *restoreWritePacer) wait(
	ctx context.Context,
	last *time.Time,
	interval time.Duration,
) error {
	if p == nil ||
		interval <= 0 {
		return nil
	}

	if !last.IsZero() {
		wait :=
			time.Until(
				last.Add(interval),
			)

		if wait > 0 {
			if err := waitContext(
				ctx,
				wait,
			); err != nil {
				return err
			}
		}
	}

	*last = time.Now()

	return nil
}

func (p *restoreWritePacer) waitCreate(
	ctx context.Context,
) error {
	return p.wait(
		ctx,
		&p.lastCreate,
		restorePanelWriteInterval,
	)
}

func (p *restoreWritePacer) waitUpdate(
	ctx context.Context,
) error {
	return p.wait(
		ctx,
		&p.lastUpdate,
		restorePanelWriteInterval,
	)
}

func (p *restoreWritePacer) waitDelete(
	ctx context.Context,
) error {
	return p.wait(
		ctx,
		&p.lastDelete,
		restorePanelWriteInterval,
	)
}

func (p *restoreWritePacer) waitTarget(
	ctx context.Context,
) error {
	return p.wait(
		ctx,
		&p.lastTarget,
		restoreTargetWriteInterval,
	)
}

// Full Restore 自己实现 retry，
// 这样 40030009 重试也会继续经过 QPM pacing。
// 普通 Apply 的轻量 retry 保持不变。

func (m *Manager) restoreCreatePanel(
	ctx context.Context,
	pacer *restoreWritePacer,
	value *dto.PanelToCreate,
) (*dto.PanelCreateResponse, error) {
	var lastErr error

	for attempt := 0;
		attempt < panelBusyRetryCount;
		attempt++ {

		if err := pacer.waitCreate(
			ctx,
		); err != nil {
			return nil, err
		}

		result, err :=
			m.api.CreatePanel(
				ctx,
				value,
			)

		if err == nil {
			if result == nil ||
				strings.TrimSpace(
					result.PanelID,
				) == "" {
				return nil, fmt.Errorf(
					"create panel returned empty panel_id",
				)
			}

			result.PanelID =
				strings.TrimSpace(
					result.PanelID,
				)

			return result, nil
		}

		lastErr = err

		if !isPanelBusyError(err) {
			return nil, err
		}
	}

	return nil, lastErr
}

func (m *Manager) restoreUpdatePanel(
	ctx context.Context,
	pacer *restoreWritePacer,
	panelID string,
	panel *dto.Panel,
) error {
	var lastErr error

	for attempt := 0;
		attempt < panelBusyRetryCount;
		attempt++ {

		if err := pacer.waitUpdate(
			ctx,
		); err != nil {
			return err
		}

		err := m.api.UpdatePanel(
			ctx,
			panelID,
			panel,
		)

		if err == nil {
			return nil
		}

		lastErr = err

		if !isPanelBusyError(err) {
			return err
		}
	}

	return lastErr
}

func (m *Manager) restoreDeletePanel(
	ctx context.Context,
	pacer *restoreWritePacer,
	panelID string,
) error {
	var lastErr error

	for attempt := 0;
		attempt < panelBusyRetryCount;
		attempt++ {

		if err := pacer.waitDelete(
			ctx,
		); err != nil {
			return err
		}

		err := m.api.DeletePanel(
			ctx,
			panelID,
		)

		if err == nil {
			return nil
		}

		lastErr = err

		if !isPanelBusyError(err) {
			return err
		}
	}

	return lastErr
}

func (m *Manager) restoreUpdateTargets(
	ctx context.Context,
	pacer *restoreWritePacer,
	panelID string,
	value *dto.PanelTargetToUpdate,
) error {
	var lastErr error

	for attempt := 0;
		attempt < panelBusyRetryCount;
		attempt++ {

		if err := pacer.waitTarget(
			ctx,
		); err != nil {
			return err
		}

		err := m.api.UpdatePanelTargets(
			ctx,
			panelID,
			value,
		)

		if err == nil {
			return nil
		}

		lastErr = err

		if !isPanelBusyError(err) {
			return err
		}
	}

	return lastErr
}