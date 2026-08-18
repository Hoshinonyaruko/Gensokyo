package commandpanel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tencent-connect/botgo/dto"
)

const (
	// Apply / Full Restore 可能包含多次受 QPM 限制的 QQ 写操作。
	// 使用脱离浏览器连接生命周期的长超时 Context，避免浏览器刷新、
	// 标签页关闭或临时断线把已经开始的事务式操作截断在中间。
	commandPanelMutationTimeout = 60 * time.Minute
)

// HTTPHandler 是 Command Panel 本地 WebUI 的 HTTP 适配层。
//
// 它只做：
//   - localhost 安全边界
//   - HTTP 参数 / JSON 解析
//   - 调用 Manager / Storage
//   - 统一 JSON 响应
//
// QQ 业务规则、Apply / Restore 的写入顺序与最终校验仍由 Manager 负责。
type HTTPHandler struct {
	manager *Manager
	storage *Storage
	appID   string
}

type commandPanelConfigNameRequest struct {
	Name string `json:"name"`
}

type commandPanelRestoreRequest struct {
	Name         string `json:"name"`
	Source       string `json:"source"`
	ConfirmAppID string `json:"confirm_appid"`
}

// RegisterHTTPRoutes 把 Command Panel 页面与 API 注册到现有主 Gin Engine。
//
// 不创建第二套 HTTP Server，也不注册到正向 OneBot hr Engine。
// /cmd 与 /cmd/api 使用完全相同的 loopback 安全边界。
func RegisterHTTPRoutes(
	router *gin.Engine,
	manager *Manager,
	storage *Storage,
	appID string,
) error {
	if router == nil {
		return fmt.Errorf("command panel router is nil")
	}

	if manager == nil {
		return fmt.Errorf("command panel manager is nil")
	}

	if storage == nil {
		return fmt.Errorf("command panel storage is nil")
	}

	appID = strings.TrimSpace(appID)
	if appID == "" {
		return fmt.Errorf("command panel appid is empty")
	}

	h := &HTTPHandler{
		manager: manager,
		storage: storage,
		appID:   appID,
	}

	localOnly := commandPanelLoopbackOnly()

	// Command Panel 页面。
	router.GET(
		"/cmd",
		localOnly,
		h.getUI,
	)
	router.GET(
		"/cmd/",
		localOnly,
		h.getUI,
	)

	group := router.Group(
		"/cmd/api",
		localOnly,
	)

	// QQ 远端只读状态。
	group.GET("/remote", h.getRemote)
	group.GET("/menu", h.getMenu)
	group.GET("/panels", h.getPanels)

	// Menu 写入仍走完整 Apply 保护链。
	group.PUT("/menu", h.putMenu)

	// 普通配置。
	group.GET("/configs", h.getConfigs)
	group.GET("/configs/:name", h.getConfig)
	group.PUT("/configs/:name", h.putConfig)
	group.POST("/configs/snapshot", h.postConfigSnapshot)

	// Phase 7E：只读读取指定 autobackup。
	// LoadBackup 内部仍使用 normalizeConfigName，不能传任意文件系统路径。
	group.GET("/backups/:name", h.getBackup)

	// 远端 mutation。
	group.POST("/apply", h.postApply)
	group.POST("/backup", h.postBackup)
	group.POST("/restore", h.postRestore)

	return nil
}

// commandPanelLoopbackOnly 是 /cmd 的安全边界。
//
// 只信任实际 TCP peer（Request.RemoteAddr），绝不信任 ClientIP、
// X-Forwarded-For、X-Real-IP 等可由请求方伪造的代理头。
func commandPanelLoopbackOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")

		host := strings.TrimSpace(
			c.Request.RemoteAddr,
		)

		if parsedHost, _, err :=
			net.SplitHostPort(host); err == nil {
			host = parsedHost
		}

		host = strings.Trim(host, "[]")

		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{
					"ok":    false,
					"error": "command panel api is localhost-only",
				},
			)
			return
		}

		c.Next()
	}
}

func commandPanelMutationContext() (
	context.Context,
	context.CancelFunc,
) {
	return context.WithTimeout(
		context.Background(),
		commandPanelMutationTimeout,
	)
}

func commandPanelSuccess(
	c *gin.Context,
	data interface{},
) {
	c.JSON(
		http.StatusOK,
		gin.H{
			"ok":   true,
			"data": data,
		},
	)
}

func commandPanelFailure(
	c *gin.Context,
	status int,
	err error,
	data interface{},
) {
	body := gin.H{
		"ok":    false,
		"error": err.Error(),
	}

	if data != nil {
		body["data"] = data
	}

	c.JSON(status, body)
}

func decodeCommandPanelJSON(
	c *gin.Context,
	value interface{},
	strict bool,
) error {
	decoder := json.NewDecoder(c.Request.Body)
	if strict {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(value); err != nil {
		return err
	}

	var extra interface{}
	err := decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf(
				"multiple top-level JSON values",
			)
		}
		return fmt.Errorf(
			"invalid trailing JSON: %w",
			err,
		)
	}

	return nil
}

func (h *HTTPHandler) getRemote(c *gin.Context) {
	state, err := h.manager.FetchFullState(
		c.Request.Context(),
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			fmt.Errorf("fetch remote state: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(
		c,
		gin.H{
			"app_id": h.appID,
			"menu":   state.Menu,
			"panels": state.Panels,
		},
	)
}

func (h *HTTPHandler) getMenu(c *gin.Context) {
	state, err := h.manager.FetchFullState(
		c.Request.Context(),
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			fmt.Errorf("fetch remote state: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(c, state.Menu)
}

// putMenu 不直接调用 UpdateGlobalMenu。
//
// 即使只改 Menu，也必须复用 ApplyConfig 的：
//   - writeMu
//   - 完整远端读取
//   - 自动备份
//   - 最终 verify
func (h *HTTPHandler) putMenu(c *gin.Context) {
	var menu *dto.Menu
	if err := decodeCommandPanelJSON(
		c,
		&menu,
		false,
	); err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("decode menu: %w", err),
			nil,
		)
		return
	}

	ctx, cancel := commandPanelMutationContext()
	defer cancel()

	state, err := h.manager.FetchFullState(ctx)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			fmt.Errorf("fetch remote state before menu apply: %w", err),
			nil,
		)
		return
	}

	config, err := ConfigFromRemoteState(
		h.appID,
		state,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusInternalServerError,
			err,
			nil,
		)
		return
	}

	config.Menu = menu

	result, err := h.manager.ApplyConfig(
		ctx,
		h.storage,
		h.appID,
		config,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			err,
			result,
		)
		return
	}

	commandPanelSuccess(c, result)
}

func (h *HTTPHandler) getPanels(c *gin.Context) {
	state, err := h.manager.FetchFullState(
		c.Request.Context(),
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			fmt.Errorf("fetch remote state: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(c, state.Panels)
}

func (h *HTTPHandler) getConfigs(c *gin.Context) {
	configs, err := h.storage.ListConfigs()
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusInternalServerError,
			fmt.Errorf("list configs: %w", err),
			nil,
		)
		return
	}

	backups, err := h.storage.ListBackups()
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusInternalServerError,
			fmt.Errorf("list backups: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(
		c,
		gin.H{
			"configs": configs,
			"backups": backups,
		},
	)
}

func (h *HTTPHandler) getConfig(c *gin.Context) {
	config, err := h.storage.LoadConfig(
		c.Param("name"),
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}

		commandPanelFailure(
			c,
			status,
			fmt.Errorf("load config: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(c, config)
}

// getBackup 是 Phase 7E 唯一新增的只读后端能力。
//
// 它只允许通过 Storage.LoadBackup 读取 autobackup 目录中的单个 JSON，
// 不接受绝对路径、不接受目录穿越，也不产生 QQ / 磁盘写操作。
func (h *HTTPHandler) getBackup(c *gin.Context) {
	config, err := h.storage.LoadBackup(
		c.Param("name"),
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("load backup: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(c, config)
}

func (h *HTTPHandler) putConfig(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxConfigFileSize,
	)

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		status := http.StatusBadRequest
		if errors.As(err, &maxErr) {
			status = http.StatusRequestEntityTooLarge
		}

		commandPanelFailure(
			c,
			status,
			fmt.Errorf("read config body: %w", err),
			nil,
		)
		return
	}

	config, err := h.storage.ImportConfig(
		c.Param("name"),
		data,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("save config: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(c, config)
}

func (h *HTTPHandler) postConfigSnapshot(c *gin.Context) {
	var request commandPanelConfigNameRequest
	if err := decodeCommandPanelJSON(
		c,
		&request,
		true,
	); err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("decode snapshot request: %w", err),
			nil,
		)
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("snapshot name is empty"),
			nil,
		)
		return
	}

	state, err := h.manager.FetchFullState(
		c.Request.Context(),
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			fmt.Errorf("fetch remote state for snapshot: %w", err),
			nil,
		)
		return
	}

	config, err := h.storage.SaveRemoteState(
		request.Name,
		h.appID,
		state,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("save snapshot: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(c, config)
}

func (h *HTTPHandler) postApply(c *gin.Context) {
	var request commandPanelConfigNameRequest
	if err := decodeCommandPanelJSON(
		c,
		&request,
		true,
	); err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("decode apply request: %w", err),
			nil,
		)
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("config name is empty"),
			nil,
		)
		return
	}

	config, err := h.storage.LoadConfig(
		request.Name,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("load apply config: %w", err),
			nil,
		)
		return
	}

	ctx, cancel := commandPanelMutationContext()
	defer cancel()

	result, err := h.manager.ApplyConfig(
		ctx,
		h.storage,
		h.appID,
		config,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			err,
			result,
		)
		return
	}

	commandPanelSuccess(c, result)
}

// postBackup 只读取 QQ 完整状态并写本地 autobackup，绝不写 QQ。
func (h *HTTPHandler) postBackup(c *gin.Context) {
	state, err := h.manager.FetchFullState(
		c.Request.Context(),
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			fmt.Errorf("fetch remote state for backup: %w", err),
			nil,
		)
		return
	}

	info, err := h.storage.BackupRemoteState(
		h.appID,
		state,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusInternalServerError,
			fmt.Errorf("backup remote state: %w", err),
			nil,
		)
		return
	}

	commandPanelSuccess(c, info)
}

func (h *HTTPHandler) postRestore(c *gin.Context) {
	var request commandPanelRestoreRequest
	if err := decodeCommandPanelJSON(
		c,
		&request,
		true,
	); err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("decode restore request: %w", err),
			nil,
		)
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Source = strings.TrimSpace(request.Source)
	request.ConfirmAppID = strings.TrimSpace(
		request.ConfirmAppID,
	)

	if request.Name == "" {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("restore name is empty"),
			nil,
		)
		return
	}

	if request.ConfirmAppID != h.appID {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf(
				"confirm_appid mismatch: got=%s current=%s",
				request.ConfirmAppID,
				h.appID,
			),
			nil,
		)
		return
	}

	var (
		config *ConfigFile
		err    error
	)

	switch request.Source {
	case "backup":
		config, err = h.storage.LoadBackup(
			request.Name,
		)

	case "config":
		config, err = h.storage.LoadConfig(
			request.Name,
		)

	default:
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf(
				"restore source must be backup or config",
			),
			nil,
		)
		return
	}

	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadRequest,
			fmt.Errorf("load restore source: %w", err),
			nil,
		)
		return
	}

	ctx, cancel := commandPanelMutationContext()
	defer cancel()

	result, err := h.manager.RestoreConfig(
		ctx,
		h.storage,
		h.appID,
		config,
	)
	if err != nil {
		commandPanelFailure(
			c,
			http.StatusBadGateway,
			err,
			result,
		)
		return
	}

	commandPanelSuccess(c, result)
}
