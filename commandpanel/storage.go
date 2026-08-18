package commandpanel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/tencent-connect/botgo/dto"
)

var (
	// ErrConfigAlreadyExists 表示 create-only 写入时目标配置已经存在。
	ErrConfigAlreadyExists = errors.New("config already exists")

	// ErrConfigRevisionMismatch 表示覆盖配置时磁盘版本已经发生变化。
	ErrConfigRevisionMismatch = errors.New("config revision mismatch")

	// ErrConfigWritePrecondition 表示条件写入缺少必要的并发控制条件。
	ErrConfigWritePrecondition = errors.New("config write precondition required")
)

const (
	configWriteModeCreate  = "create"
	configWriteModeReplace = "replace"
)

const (
	// ConfigFormat 配置文件格式标识.
	ConfigFormat = "gensokyo-cmd"

	// ConfigSchemaVersion 当前配置文件结构版本.
	ConfigSchemaVersion = 1

	configDirName     = "command_panels"
	autoBackupDirName = "autobackup"

	// 配置文件本身非常小。
	// 这里设置一个宽松上限，防止意外读取超大文件。
	maxConfigFileSize = 8 << 20 // 8 MiB
)

// ConfigFile 是 Gensokyo Menu / Command Panel 唯一的配置文件格式.
//
// 普通手工配置、自动备份、导入导出、完整恢复全部使用这一种结构。
// panel_id/version/created_at/updated_at 等远端元数据可以存在，
// 但后续 Restore 不会把 panel_id 当作必须恢复一致的业务配置。
type ConfigFile struct {
	Format        string             `json:"format"`
	SchemaVersion int                `json:"schema_version"`
	SourceAppID   string             `json:"source_appid,omitempty"`
	SavedAt       string             `json:"saved_at"`
	Menu          *dto.Menu          `json:"menu"`
	Panels        []*dto.PanelRecord `json:"panels"`
}

// ConfigInfo 是 WebUI 后续展示本地配置列表需要的最小文件信息.
type ConfigInfo struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

// Storage 管理 Gensokyo.exe 同级的 command_panels 目录.
//
// 不使用数据库，不缓存目录内容。
// WebUI 每次点击“刷新配置文件”时直接重新扫描目录。
type Storage struct {
	rootDir   string
	backupDir string

	// 所有写操作串行，避免用户连续点击造成多个临时文件互相覆盖。
	mu sync.Mutex
}

// NewStorage 创建默认 Storage.
//
// 目录结构:
//
//	Gensokyo.exe
//	command_panels/
//	    default.json
//	    qixi.json
//	    autobackup/
//	        20260818-123456-xxxxxxxxx.json
func NewStorage() (*Storage, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	return NewStorageAt(filepath.Dir(exe))
}

// NewStorageAt 使用指定的可执行目录创建 Storage.
//
// 主要用于测试，也让 Storage 本身不依赖当前工作目录。
func NewStorageAt(exeDir string) (*Storage, error) {
	if strings.TrimSpace(exeDir) == "" {
		return nil, fmt.Errorf("executable directory is empty")
	}

	absDir, err := filepath.Abs(exeDir)
	if err != nil {
		return nil, fmt.Errorf("resolve executable directory: %w", err)
	}

	rootDir := filepath.Join(absDir, configDirName)
	backupDir := filepath.Join(rootDir, autoBackupDirName)

	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create autobackup directory: %w", err)
	}

	storage := &Storage{
		rootDir:   rootDir,
		backupDir: backupDir,
	}

	// 如果上一次程序恰好在覆盖文件过程中退出，
	// 尝试恢复遗留的 .bak，并清理已经失效的 .tmp 文件。
	if err := storage.recoverInterruptedWrites(); err != nil {
		return nil, err
	}

	return storage, nil
}

// RootDir 返回普通配置文件目录.
func (s *Storage) RootDir() string {
	if s == nil {
		return ""
	}

	return s.rootDir
}

// BackupDir 返回自动备份目录.
func (s *Storage) BackupDir() string {
	if s == nil {
		return ""
	}

	return s.backupDir
}

// ConfigFromRemoteState 把 QQ 远端状态转换成统一配置文件.
//
// state 应该通常来自 FetchFullState，
// 这样 specific panel 的 user_openids/group_openids 才是完整的。
func ConfigFromRemoteState(
	sourceAppID string,
	state *RemoteState,
) (*ConfigFile, error) {
	if state == nil {
		return nil, fmt.Errorf("remote state is nil")
	}

	panels := state.Panels
	if panels == nil {
		panels = []*dto.PanelRecord{}
	}

	return &ConfigFile{
		Format:        ConfigFormat,
		SchemaVersion: ConfigSchemaVersion,
		SourceAppID:   sourceAppID,
		SavedAt:       time.Now().Format(time.RFC3339Nano),
		Menu:          state.Menu,
		Panels:        panels,
	}, nil
}

// SaveRemoteState 把完整远端状态保存为一个普通配置文件.
func (s *Storage) SaveRemoteState(
	name string,
	sourceAppID string,
	state *RemoteState,
) (*ConfigFile, error) {
	config, err := ConfigFromRemoteState(sourceAppID, state)
	if err != nil {
		return nil, err
	}

	if err := s.SaveConfig(name, config); err != nil {
		return nil, err
	}

	// SaveConfig 会刷新 format/schema_version/saved_at。
	// 重新读取，保证返回值与真正落盘的数据完全一致。
	return s.LoadConfig(name)
}

// SaveConfig 保存普通配置.
//
// name 可以写:
//
//	default
//	default.json
//	七夕前
//
// 未提供 .json 时自动补齐。
func (s *Storage) SaveConfig(
	name string,
	config *ConfigFile,
) error {
	if s == nil {
		return fmt.Errorf("storage is nil")
	}

	fileName, err := normalizeConfigName(name)
	if err != nil {
		return err
	}

	if config == nil {
		return fmt.Errorf("config is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	output := prepareConfigForWrite(config)

	path := filepath.Join(s.rootDir, fileName)

	if err := writeJSONAtomic(path, output); err != nil {
		return fmt.Errorf(
			"save config %s: %w",
			fileName,
			err,
		)
	}

	return nil
}

// SaveRemoteStateConditional 把完整远端状态按乐观并发条件保存为普通配置。
//
// create: 仅当目标配置仍不存在时创建。
// replace: 仅当目标配置当前 saved_at 与 expectedSavedAt 完全一致时覆盖。
//
// 这个方法专门供 WebUI 使用，防止多个标签页或外部手工改文件时
// 旧页面静默覆盖较新的磁盘配置。
func (s *Storage) SaveRemoteStateConditional(
	name string,
	sourceAppID string,
	state *RemoteState,
	writeMode string,
	expectedSavedAt string,
) (*ConfigFile, error) {
	config, err := ConfigFromRemoteState(sourceAppID, state)
	if err != nil {
		return nil, err
	}

	return s.saveConfigConditional(
		name,
		config,
		writeMode,
		expectedSavedAt,
	)
}

// ImportConfigConditional 与 ImportConfig 使用同一 ConfigFile 格式，
// 但额外要求调用方声明 create / replace 条件。
func (s *Storage) ImportConfigConditional(
	name string,
	data []byte,
	writeMode string,
	expectedSavedAt string,
) (*ConfigFile, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("config data is empty")
	}

	if len(data) > maxConfigFileSize {
		return nil, fmt.Errorf(
			"config file exceeds maximum size %d bytes",
			maxConfigFileSize,
		)
	}

	config, err := decodeConfig(data)
	if err != nil {
		return nil, err
	}

	if err := validateConfigEnvelope(config); err != nil {
		return nil, err
	}

	return s.saveConfigConditional(
		name,
		config,
		writeMode,
		expectedSavedAt,
	)
}

// saveConfigConditional 在 Storage 写锁内完成“检查当前版本 + 原子写入”。
//
// 不能把存在性检查放在 HTTPHandler 再调用 SaveConfig，
// 否则两个标签页仍可能同时通过检查后互相覆盖。
func (s *Storage) saveConfigConditional(
	name string,
	config *ConfigFile,
	writeMode string,
	expectedSavedAt string,
) (*ConfigFile, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	fileName, err := normalizeConfigName(name)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	writeMode = strings.TrimSpace(writeMode)
	expectedSavedAt = strings.TrimSpace(expectedSavedAt)

	if writeMode != configWriteModeCreate &&
		writeMode != configWriteModeReplace {
		return nil, fmt.Errorf(
			"unsupported config write mode %q",
			writeMode,
		)
	}

	if writeMode == configWriteModeReplace && expectedSavedAt == "" {
		return nil, ErrConfigWritePrecondition
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.rootDir, fileName)
	existing, loadErr := loadConfigFile(path)

	switch writeMode {
	case configWriteModeCreate:
		if loadErr == nil {
			return nil, fmt.Errorf(
				"%w: %s",
				ErrConfigAlreadyExists,
				fileName,
			)
		}
		if !errors.Is(loadErr, os.ErrNotExist) {
			return nil, fmt.Errorf(
				"check existing config %s: %w",
				fileName,
				loadErr,
			)
		}

	case configWriteModeReplace:
		if loadErr != nil {
			return nil, fmt.Errorf(
				"load config before replace %s: %w",
				fileName,
				loadErr,
			)
		}

		currentSavedAt := strings.TrimSpace(existing.SavedAt)
		if currentSavedAt != expectedSavedAt {
			return nil, fmt.Errorf(
				"%w: %s expected_saved_at=%q current_saved_at=%q",
				ErrConfigRevisionMismatch,
				fileName,
				expectedSavedAt,
				currentSavedAt,
			)
		}
	}

	output := prepareConfigForWrite(config)
	if err := writeJSONAtomic(path, output); err != nil {
		return nil, fmt.Errorf(
			"save config %s: %w",
			fileName,
			err,
		)
	}

	return loadConfigFile(path)
}

// ImportConfig 导入浏览器上传的 JSON.
//
// 它与“用户手工把文件复制进 command_panels”并不冲突。
// 手工复制以后 ListConfigs 会直接发现；
// 浏览器上传文件时则走这个方法。
func (s *Storage) ImportConfig(
	name string,
	data []byte,
) (*ConfigFile, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("config data is empty")
	}

	if len(data) > maxConfigFileSize {
		return nil, fmt.Errorf(
			"config file exceeds maximum size %d bytes",
			maxConfigFileSize,
		)
	}

	config, err := decodeConfig(data)
	if err != nil {
		return nil, err
	}

	if err := validateConfigEnvelope(config); err != nil {
		return nil, err
	}

	if err := s.SaveConfig(name, config); err != nil {
		return nil, err
	}

	// 返回真正落盘后的版本，避免 saved_at 与磁盘内容不一致。
	return s.LoadConfig(name)
}

// LoadConfig 读取普通配置.
func (s *Storage) LoadConfig(name string) (*ConfigFile, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	fileName, err := normalizeConfigName(name)
	if err != nil {
		return nil, err
	}

	return loadConfigFile(
		filepath.Join(s.rootDir, fileName),
	)
}

// LoadBackup 读取自动备份.
func (s *Storage) LoadBackup(name string) (*ConfigFile, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	fileName, err := normalizeConfigName(name)
	if err != nil {
		return nil, err
	}

	return loadConfigFile(
		filepath.Join(s.backupDir, fileName),
	)
}

// ListConfigs 每次直接重新扫描 command_panels 根目录.
//
// 不递归，不缓存。
// 因此用户在 Gensokyo 运行期间直接复制 JSON 进去，
// 下一次调用即可发现。
func (s *Storage) ListConfigs() ([]ConfigInfo, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	return listJSONFiles(s.rootDir)
}

// ListBackups 扫描 autobackup 目录.
func (s *Storage) ListBackups() ([]ConfigInfo, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	return listJSONFiles(s.backupDir)
}

// BackupRemoteState 保存完整远端自动备份.
//
// 文件名使用纳秒，避免连续操作时发生名字碰撞。
func (s *Storage) BackupRemoteState(
	sourceAppID string,
	state *RemoteState,
) (*ConfigInfo, error) {
	if s == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	config, err := ConfigFromRemoteState(
		sourceAppID,
		state,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	config.SavedAt = now.Format(time.RFC3339Nano)

	fileName := now.Format(
		"20060102-150405-000000000",
	) + ".json"

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(
		s.backupDir,
		fileName,
	)

	if err := writeJSONAtomic(path, config); err != nil {
		return nil, fmt.Errorf(
			"write autobackup %s: %w",
			fileName,
			err,
		)
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf(
			"stat autobackup %s: %w",
			fileName,
			err,
		)
	}

	return &ConfigInfo{
		Name:       fileName,
		Size:       stat.Size(),
		ModifiedAt: stat.ModTime().Format(time.RFC3339Nano),
	}, nil
}

// prepareConfigForWrite 为当前保存动作生成规范化副本.
//
// 不直接修改调用方传入的对象。
func prepareConfigForWrite(config *ConfigFile) *ConfigFile {
	output := *config

	output.Format = ConfigFormat
	output.SchemaVersion = ConfigSchemaVersion
	output.SavedAt = time.Now().Format(time.RFC3339Nano)

	if output.Panels == nil {
		output.Panels = []*dto.PanelRecord{}
	}

	return &output
}

func validateConfigEnvelope(config *ConfigFile) error {
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
			"unsupported schema_version %d, current version is %d",
			config.SchemaVersion,
			ConfigSchemaVersion,
		)
	}

	return nil
}

func loadConfigFile(path string) (*ConfigFile, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, fmt.Errorf(
			"%s is a directory",
			filepath.Base(path),
		)
	}

	if stat.Size() > maxConfigFileSize {
		return nil, fmt.Errorf(
			"config file exceeds maximum size %d bytes",
			maxConfigFileSize,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"read %s: %w",
			filepath.Base(path),
			err,
		)
	}

	config, err := decodeConfig(data)
	if err != nil {
		return nil, fmt.Errorf(
			"decode %s: %w",
			filepath.Base(path),
			err,
		)
	}

	if err := validateConfigEnvelope(config); err != nil {
		return nil, fmt.Errorf(
			"validate %s: %w",
			filepath.Base(path),
			err,
		)
	}

	if config.Panels == nil {
		config.Panels = []*dto.PanelRecord{}
	}

	return config, nil
}

func decodeConfig(data []byte) (*ConfigFile, error) {
	var config ConfigFile

	decoder := json.NewDecoder(
		strings.NewReader(string(data)),
	)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf(
			"invalid JSON: %w",
			err,
		)
	}

	// 防止:
	//
	//	{...}
	//	{...}
	//
	// 这种第一个对象正常、后面还藏有内容的文件被静默接受。
	var extra interface{}

	err := decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf(
				"invalid JSON: multiple top-level values",
			)
		}

		return nil, fmt.Errorf(
			"invalid trailing JSON: %w",
			err,
		)
	}

	return &config, nil
}

func listJSONFiles(dir string) ([]ConfigInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := make([]ConfigInfo, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !strings.EqualFold(
			filepath.Ext(name),
			".json",
		) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf(
				"stat config %s: %w",
				name,
				err,
			)
		}

		result = append(
			result,
			ConfigInfo{
				Name:       name,
				Size:       info.Size(),
				ModifiedAt: info.ModTime().Format(time.RFC3339Nano),
			},
		)
	}

	// 新修改的配置排在最前面。
	sort.Slice(
		result,
		func(i, j int) bool {
			if result[i].ModifiedAt == result[j].ModifiedAt {
				return result[i].Name < result[j].Name
			}

			return result[i].ModifiedAt > result[j].ModifiedAt
		},
	)

	return result, nil
}

func normalizeConfigName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return "", fmt.Errorf(
			"config name is empty",
		)
	}

	if filepath.Base(name) != name ||
		strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf(
			"invalid config name %q",
			name,
		)
	}

	for _, r := range name {
		if unicode.IsControl(r) ||
			strings.ContainsRune(`:*?"<>|`, r) {
			return "", fmt.Errorf(
				"invalid character in config name %q",
				name,
			)
		}
	}

	if strings.HasSuffix(name, ".") ||
		strings.HasSuffix(name, " ") {
		return "", fmt.Errorf(
			"invalid config name %q",
			name,
		)
	}

	ext := filepath.Ext(name)

	switch {
	case ext == "":
		name += ".json"

	case !strings.EqualFold(ext, ".json"):
		return "", fmt.Errorf(
			"config file must use .json extension",
		)
	}

	base := strings.TrimSuffix(
		name,
		filepath.Ext(name),
	)

	if isWindowsReservedName(base) {
		return "", fmt.Errorf(
			"reserved config name %q",
			base,
		)
	}

	return name, nil
}

func isWindowsReservedName(name string) bool {
	upper := strings.ToUpper(
		strings.TrimSpace(name),
	)

	// Windows 的 CON.xxx、NUL.xxx、COM1.xxx 等仍然是设备保留名。
	if dot := strings.IndexByte(upper, '.'); dot >= 0 {
		upper = upper[:dot]
	}

	switch upper {
	case "CON",
		"PRN",
		"AUX",
		"NUL",
		"COM1",
		"COM2",
		"COM3",
		"COM4",
		"COM5",
		"COM6",
		"COM7",
		"COM8",
		"COM9",
		"LPT1",
		"LPT2",
		"LPT3",
		"LPT4",
		"LPT5",
		"LPT6",
		"LPT7",
		"LPT8",
		"LPT9":
		return true
	}

	return false
}

// writeJSONAtomic 先把完整 JSON 写进同目录临时文件，
// fsync 成功以后再替换目标文件.
//
// 对“目标不存在”的情况:
//
//	tmp -> target
//
// 对“目标已经存在”的情况:
//
//	target -> target.bak
//	tmp    -> target
//	success -> 删除 target.bak
//
// 如果第二步失败，会立即把 .bak 恢复回来。
//
// 这种方式避免 Windows 上直接 Rename 覆盖已有文件的兼容问题，
// 同时也比“先 Remove(target) 再 Rename”具有更好的可恢复性。
func writeJSONAtomic(
	path string,
	value interface{},
) error {
	data, err := json.MarshalIndent(
		value,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"marshal JSON: %w",
			err,
		)
	}

	data = append(data, '\n')

	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(
		dir,
		"."+filepath.Base(path)+".tmp-*",
	)
	if err != nil {
		return fmt.Errorf(
			"create temporary file: %w",
			err,
		)
	}

	tmpPath := tmp.Name()

	// 无论成功失败，最后都尝试清理遗留 tmp。
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()

		return fmt.Errorf(
			"chmod temporary file: %w",
			err,
		)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()

		return fmt.Errorf(
			"write temporary file: %w",
			err,
		)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()

		return fmt.Errorf(
			"sync temporary file: %w",
			err,
		)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf(
			"close temporary file: %w",
			err,
		)
	}

	_, statErr := os.Stat(path)

	if errors.Is(statErr, os.ErrNotExist) {
		if err := os.Rename(tmpPath, path); err != nil {
			return fmt.Errorf(
				"install new config: %w",
				err,
			)
		}

		return nil
	}

	if statErr != nil {
		return fmt.Errorf(
			"stat target config: %w",
			statErr,
		)
	}

	backupPath := path + ".bak"

	// 正常情况下不会存在。
	// 如果存在，说明是历史中断操作留下来的；
	// 当前 target 又已经存在，因此这个 bak 已经是陈旧副本。
	_ = os.Remove(backupPath)

	if err := os.Rename(
		path,
		backupPath,
	); err != nil {
		return fmt.Errorf(
			"protect old config: %w",
			err,
		)
	}

	if err := os.Rename(
		tmpPath,
		path,
	); err != nil {
		rollbackErr := os.Rename(
			backupPath,
			path,
		)

		if rollbackErr != nil {
			return fmt.Errorf(
				"replace config failed: %v; rollback failed: %w",
				err,
				rollbackErr,
			)
		}

		return fmt.Errorf(
			"replace config: %w",
			err,
		)
	}

	// 新文件已经完整落位。
	// 即使这里删除失败，也只会留下一个 .bak，
	// 下次启动 recoverInterruptedWrites 会清理它。
	_ = os.Remove(backupPath)

	return nil
}

// recoverInterruptedWrites 恢复上次异常退出留下的文件.
//
// 规则:
//
//	target.json 存在 + target.json.bak 存在
//	    -> 新 target 已经成功落位，删除陈旧 bak
//
//	target.json 不存在 + target.json.bak 存在
//	    -> 上次中断在替换中间，恢复 bak
//
//	*.tmp-*
//	    -> 永远只是未完成的临时文件，删除
func (s *Storage) recoverInterruptedWrites() error {
	if err := recoverDirectory(
		s.rootDir,
	); err != nil {
		return fmt.Errorf(
			"recover config directory: %w",
			err,
		)
	}

	if err := recoverDirectory(
		s.backupDir,
	); err != nil {
		return fmt.Errorf(
			"recover autobackup directory: %w",
			err,
		)
	}

	return nil
}

func recoverDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(dir, name)
		lowerName := strings.ToLower(name)

		if strings.HasPrefix(name, ".") &&
			strings.Contains(lowerName, ".json.tmp-") {
			if err := os.Remove(path); err != nil &&
				!errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf(
					"remove stale temporary file %s: %w",
					name,
					err,
				)
			}

			continue
		}

		if !strings.HasSuffix(
			lowerName,
			".json.bak",
		) {
			continue
		}

		targetPath := strings.TrimSuffix(
			path,
			".bak",
		)

		_, targetErr := os.Stat(targetPath)

		switch {
		case targetErr == nil:
			// 新目标文件已经存在，bak 已经没有作用。
			if err := os.Remove(path); err != nil {
				return fmt.Errorf(
					"remove stale backup %s: %w",
					name,
					err,
				)
			}

		case errors.Is(targetErr, os.ErrNotExist):
			// 目标丢失，说明上次退出发生在:
			//
			// target -> bak
			// tmp    -> target
			//
			// 两步之间。
			if err := os.Rename(
				path,
				targetPath,
			); err != nil {
				return fmt.Errorf(
					"restore interrupted config %s: %w",
					name,
					err,
				)
			}

		default:
			return fmt.Errorf(
				"stat interrupted config %s: %w",
				name,
				targetErr,
			)
		}
	}

	return nil
}
