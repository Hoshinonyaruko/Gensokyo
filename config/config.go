package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/sys"
	"github.com/hoshinonyaruko/gensokyo/template"
	"gopkg.in/yaml.v3"
)

var (
	instance *Config
	mu       sync.Mutex
)

type Config struct {
	Version  int      `yaml:"version"`
	Settings Settings `yaml:"settings"`
}
type VisualPrefixConfig struct {
	Prefix          string   `yaml:"prefix"`
	WhiteList       []string `yaml:"whiteList"`
	NoWhiteResponse string   `yaml:"No_White_Response"`
}
type Settings struct {
	WsAddress              []string             `yaml:"ws_address"`
	AppID                  uint64               `yaml:"app_id"`
	Token                  string               `yaml:"token"`
	ClientSecret           string               `yaml:"client_secret"`
	TextIntent             []string             `yaml:"text_intent"`
	GlobalChannelToGroup   bool                 `yaml:"global_channel_to_group"`
	GlobalPrivateToChannel bool                 `yaml:"global_private_to_channel"`
	Array                  bool                 `yaml:"array"`
	Server_dir             string               `yaml:"server_dir"`
	Lotus                  bool                 `yaml:"lotus"`
	Port                   string               `yaml:"port"`
	WsToken                []string             `yaml:"ws_token,omitempty"`         // 连接wss时使用,不是wss可留空 一一对应
	MasterID               []string             `yaml:"master_id,omitempty"`        // 如果需要在群权限判断是管理员是,将user_id填入这里,master_id是一个文本数组
	EnableWsServer         bool                 `yaml:"enable_ws_server,omitempty"` //正向ws开关
	WsServerToken          string               `yaml:"ws_server_token,omitempty"`  //正向ws token
	IdentifyFile           bool                 `yaml:"identify_file"`              // 域名校验文件
	Crt                    string               `yaml:"crt"`
	Key                    string               `yaml:"key"`
	DeveloperLog           bool                 `yaml:"developer_log"`
	Username               string               `yaml:"server_user_name"`
	Password               string               `yaml:"server_user_password"`
	ImageLimit             int                  `yaml:"image_sizelimit"`
	RemovePrefix           bool                 `yaml:"remove_prefix"`
	BackupPort             string               `yaml:"backup_port"`
	DevlopAcDir            string               `yaml:"develop_access_token_dir"`
	RemoveAt               bool                 `yaml:"remove_at"`
	DevBotid               string               `yaml:"develop_bot_id"`
	SandBoxMode            bool                 `yaml:"sandbox_mode"`
	Title                  string               `yaml:"title"`
	HashID                 bool                 `yaml:"hash_id"`
	TwoWayEcho             bool                 `yaml:"twoway_echo"`
	LazyMessageId          bool                 `yaml:"lazy_message_id"`
	WhitePrefixMode        bool                 `yaml:"white_prefix_mode"`
	WhitePrefixs           []string             `yaml:"white_prefixs"`
	BlackPrefixMode        bool                 `yaml:"black_prefix_mode"`
	BlackPrefixs           []string             `yaml:"black_prefixs"`
	VisualPrefixs          []VisualPrefixConfig `yaml:"visual_prefixs"`
	VisibleIp              bool                 `yaml:"visible_ip"`
	ForwardMsgLimit        int                  `yaml:"forward_msg_limit"`
	DevMessgeID            bool                 `yaml:"dev_message_id"`
	LogLevel               int                  `yaml:"log_level"`
	SaveLogs               bool                 `yaml:"save_logs"`
	BindPrefix             string               `yaml:"bind_prefix"`
	MePrefix               string               `yaml:"me_prefix"`
	FrpPort                string               `yaml:"frp_port"`
	RemoveBotAtGroup       bool                 `yaml:"remove_bot_at_group"`
	ImageLimitB            int                  `yaml:"image_limit"`
	RecordSampleRate       int                  `yaml:"record_sampleRate"`
	RecordBitRate          int                  `yaml:"record_bitRate"`
	NoWhiteResponse        string               `yaml:"No_White_Response"`
	SendError              bool                 `yaml:"send_error"`
	AddAtGroup             bool                 `yaml:"add_at_group"`
	UrlPicTransfer         bool                 `yaml:"url_pic_transfer"`
	LotusPassword          string               `yaml:"lotus_password"`
	WsServerPath           string               `yaml:"ws_server_path"`
	IdmapPro               bool                 `yaml:"idmap_pro"`
	CardAndNick            string               `yaml:"card_nick"`
	AutoBind               bool                 `yaml:"auto_bind"`
	CustomBotName          string               `yaml:"custom_bot_name"`
	SendDelay              int                  `yaml:"send_delay"`
	AtoPCount              int                  `yaml:"AMsgRetryAsPMsg_Count"`
	ReconnecTimes          int                  `yaml:"reconnect_times"`
	HeartBeatInterval      int                  `yaml:"heart_beat_interval"`
	LaunchReconectTimes    int                  `yaml:"launch_reconnect_times"`
	UnlockPrefix           string               `yaml:"unlock_prefix"`
	WhiteBypass            []int64              `yaml:"white_bypass"`
	TransferUrl            bool                 `yaml:"transfer_url"`
	HttpAddress            string               `yaml:"http_address"`
	HttpVersion            int                  `yaml:"http_version"`
	HttpTimeOut            int                  `yaml:"http_timeout"`
	PostUrl                []string             `yaml:"post_url"`
	PostSecret             []string             `yaml:"post_secret"`
	PostMaxRetries         []int                `yaml:"post_max_retries"`
	PostRetriesInterval    []int                `yaml:"post_retries_interval"`
	NativeOb11             bool                 `yaml:"native_ob11"`
	RamDomSeq              bool                 `yaml:"ramdom_seq"`
	UrlToQrimage           bool                 `yaml:"url_to_qrimage"`
	QrSize                 int                  `yaml:"qr_size"`
	WhiteBypassRevers      bool                 `yaml:"white_bypass_reverse"`
	GuildUrlImageToBase64  bool                 `yaml:"guild_url_image_to_base64"`
	TencentBucketName      string               `yaml:"t_COS_BUCKETNAME"`
	TencentBucketRegion    string               `yaml:"t_COS_REGION"`
	TencentCosSecretid     string               `yaml:"t_COS_SECRETID"`
	TencentSecretKey       string               `yaml:"t_COS_SECRETKEY"`
	TencentAudit           bool                 `yaml:"t_audit"`
	OssType                int                  `yaml:"oss_type"`
	BaiduBOSBucketName     string               `yaml:"b_BOS_BUCKETNAME"`
	BaiduBCEAK             string               `yaml:"b_BCE_AK"`
	BaiduBCESK             string               `yaml:"b_BCE_SK"`
	BaiduAudit             int                  `yaml:"b_audit"`
	AliyunEndpoint         string               `yaml:"a_OSS_EndPoint"`
	AliyunAccessKeyId      string               `yaml:"a_OSS_AccessKeyId"`
	AliyunAccessKeySecret  string               `yaml:"a_OSS_AccessKeySecret"`
	AliyunBucketName       string               `yaml:"a_OSS_BucketName"`
	AliyunAudit            bool                 `yaml:"a_audit"`
	Alias                  []string             `yaml:"alias"`
	SelfIntroduce          []string             `yaml:"self_introduce"`
	WhiteEnable            []bool               `yaml:"white_enable"`
}

// LoadConfig 从文件中加载配置并初始化单例配置
func LoadConfig(path string) (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	// 如果单例已经被初始化了，直接返回
	if instance != nil {
		return instance, nil
	}

	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	//todo remove it 破坏性变更的擦屁股代码
	var ischange bool
	configData, ischange = replaceVisualPrefixsLine(configData)
	if ischange {
		err = os.WriteFile(path, configData, 0644)
		if err != nil {
			// 处理写入错误
			return nil, err
		}
	}
	//mylog.Printf("dev_ischange:%v", ischange)
	conf := &Config{}
	err = yaml.Unmarshal(configData, conf)
	if err != nil {
		return nil, err
	}

	// 确保配置完整性
	if err := ensureConfigComplete(conf, path); err != nil {
		return nil, err
	}

	// 设置单例实例
	instance = conf
	return instance, nil
}

// 确保配置完整性
func ensureConfigComplete(conf *Config, path string) error {
	// 读取配置文件到缓冲区
	configData, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// 将现有的配置解析到结构体中
	currentConfig := &Config{}
	err = yaml.Unmarshal(configData, currentConfig)
	if err != nil {
		return err
	}

	// 解析默认配置模板到结构体中
	defaultConfig := &Config{}
	err = yaml.Unmarshal([]byte(template.ConfigTemplate), defaultConfig)
	if err != nil {
		return err
	}

	// 使用反射找出结构体中缺失的设置
	missingSettingsByReflection, err := getMissingSettingsByReflection(currentConfig, defaultConfig)
	if err != nil {
		return err
	}

	// 使用文本比对找出缺失的设置
	missingSettingsByText, err := getMissingSettingsByText(template.ConfigTemplate, string(configData))
	if err != nil {
		return err
	}

	// 合并缺失的设置
	allMissingSettings := mergeMissingSettings(missingSettingsByReflection, missingSettingsByText)

	// 如果存在缺失的设置，处理缺失的配置行
	if len(allMissingSettings) > 0 {
		fmt.Println("缺失的设置:", allMissingSettings)
		missingConfigLines, err := extractMissingConfigLines(allMissingSettings, template.ConfigTemplate)
		if err != nil {
			return err
		}

		// 将缺失的配置追加到现有配置文件
		err = appendToConfigFile(path, missingConfigLines)
		if err != nil {
			return err
		}

		fmt.Println("检测到配置文件缺少项。已经更新配置文件，正在重启程序以应用新的配置。")
		sys.RestartApplication()
	}

	return nil
}

// mergeMissingSettings 合并由反射和文本比对找到的缺失设置
func mergeMissingSettings(reflectionSettings, textSettings map[string]string) map[string]string {
	for k, v := range textSettings {
		reflectionSettings[k] = v
	}
	return reflectionSettings
}

// getMissingSettingsByReflection 使用反射来对比结构体并找出缺失的设置
func getMissingSettingsByReflection(currentConfig, defaultConfig *Config) (map[string]string, error) {
	missingSettings := make(map[string]string)
	currentVal := reflect.ValueOf(currentConfig).Elem()
	defaultVal := reflect.ValueOf(defaultConfig).Elem()

	for i := 0; i < currentVal.NumField(); i++ {
		field := currentVal.Type().Field(i)
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || field.Type.Kind() == reflect.Int || field.Type.Kind() == reflect.Bool {
			continue // 跳过没有yaml标签的字段，或者字段类型为int或bool
		}
		yamlKeyName := strings.SplitN(yamlTag, ",", 2)[0]
		if isZeroOfUnderlyingType(currentVal.Field(i).Interface()) && !isZeroOfUnderlyingType(defaultVal.Field(i).Interface()) {
			missingSettings[yamlKeyName] = "missing"
		}
	}

	return missingSettings, nil
}

// getMissingSettingsByText compares settings in two strings line by line, looking for missing keys.
func getMissingSettingsByText(templateContent, currentConfigContent string) (map[string]string, error) {
	templateKeys := extractKeysFromString(templateContent)
	currentKeys := extractKeysFromString(currentConfigContent)

	missingSettings := make(map[string]string)
	for key := range templateKeys {
		if _, found := currentKeys[key]; !found {
			missingSettings[key] = "missing"
		}
	}

	return missingSettings, nil
}

// extractKeysFromString reads a string and extracts the keys (text before the colon).
func extractKeysFromString(content string) map[string]bool {
	keys := make(map[string]bool)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			key := strings.TrimSpace(strings.Split(line, ":")[0])
			keys[key] = true
		}
	}
	return keys
}

func extractMissingConfigLines(missingSettings map[string]string, configTemplate string) ([]string, error) {
	var missingConfigLines []string

	lines := strings.Split(configTemplate, "\n")
	for yamlKey := range missingSettings {
		found := false
		// Create a regex to match the line with optional spaces around the colon
		regexPattern := fmt.Sprintf(`^\s*%s\s*:\s*`, regexp.QuoteMeta(yamlKey))
		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %s", err)
		}

		for _, line := range lines {
			if regex.MatchString(line) {
				missingConfigLines = append(missingConfigLines, line)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("missing configuration for key: %s", yamlKey)
		}
	}

	return missingConfigLines, nil
}

func appendToConfigFile(path string, lines []string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("打开文件错误:", err)
		return err
	}
	defer file.Close()

	// 写入缺失的配置项
	for _, line := range lines {
		if _, err := file.WriteString("\n" + line); err != nil {
			fmt.Println("写入配置错误:", err)
			return err
		}
	}

	// 输出写入状态
	fmt.Println("配置已更新，写入到文件:", path)

	return nil
}

func isZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

// UpdateConfig 将配置写入文件
func UpdateConfig(conf *Config, path string) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// WriteYAMLToFile 将YAML格式的字符串写入到指定的文件路径
func WriteYAMLToFile(yamlContent string) error {
	// 获取当前执行的可执行文件的路径
	exePath, err := os.Executable()
	if err != nil {
		log.Println("Error getting executable path:", err)
		return err
	}

	// 获取可执行文件所在的目录
	exeDir := filepath.Dir(exePath)

	// 构建config.yml的完整路径
	configPath := filepath.Join(exeDir, "config.yml")

	// 写入文件
	os.WriteFile(configPath, []byte(yamlContent), 0644)

	sys.RestartApplication()
	return nil
}

// DeleteConfig 删除配置文件并创建一个新的配置文件模板
func DeleteConfig() error {
	// 获取当前执行的可执行文件的路径
	exePath, err := os.Executable()
	if err != nil {
		mylog.Println("Error getting executable path:", err)
		return err
	}

	// 获取可执行文件所在的目录
	exeDir := filepath.Dir(exePath)

	// 构建config.yml的完整路径
	configPath := filepath.Join(exeDir, "config.yml")

	// 删除配置文件
	if err := os.Remove(configPath); err != nil {
		mylog.Println("Error removing config file:", err)
		return err
	}

	// 获取内网IP地址
	ip, err := sys.GetLocalIP()
	if err != nil {
		mylog.Println("Error retrieving the local IP address:", err)
		return err
	}

	// 将 <YOUR_SERVER_DIR> 替换成实际的内网IP地址
	configData := strings.Replace(template.ConfigTemplate, "<YOUR_SERVER_DIR>", ip, -1)

	// 创建一个新的配置文件模板 写到配置
	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		mylog.Println("Error writing config.yml:", err)
		return err
	}

	sys.RestartApplication()

	return nil
}

// 获取ws地址数组
func GetWsAddress() []string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.WsAddress
	}
	return nil // 返回nil，如果instance为nil
}

// 获取gensokyo服务的地址
func GetServer_dir() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get upload directory.")
		return ""
	}
	return instance.Settings.Server_dir
}

// 获取DevBotid
func GetDevBotid() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get DevBotid.")
		return "1234"
	}
	return instance.Settings.DevBotid
}

// 获取GetForwardMsgLimit
func GetForwardMsgLimit() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get GetForwardMsgLimit.")
		return 3
	}
	return instance.Settings.ForwardMsgLimit
}

// 获取Develop_Acdir服务的地址
func GetDevelop_Acdir() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get DevlopAcDir.")
		return ""
	}
	return instance.Settings.DevlopAcDir
}

// 获取lotus的值
func GetLotusValue() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get lotus value.")
		return false
	}
	return instance.Settings.Lotus
}

// 获取双向ehco
func GetTwoWayEcho() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get lotus value.")
		return false
	}
	return instance.Settings.TwoWayEcho
}

// 获取白名单开启状态
func GetWhitePrefixMode() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetWhitePrefixModes value.")
		return false
	}
	return instance.Settings.WhitePrefixMode
}

// 获取白名单指令数组
func GetWhitePrefixs() []string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.WhitePrefixs
	}
	return nil // 返回nil，如果instance为nil
}

// 获取黑名单开启状态
func GetBlackPrefixMode() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetBlackPrefixMode value.")
		return false
	}
	return instance.Settings.BlackPrefixMode
}

// 获取黑名单指令数组
func GetBlackPrefixs() []string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.BlackPrefixs
	}
	return nil // 返回nil，如果instance为nil
}

// 获取IPurl显示开启状态
func GetVisibleIP() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetVisibleIP value.")
		return false
	}
	return instance.Settings.VisibleIp
}

// 修改 GetVisualkPrefixs 函数以返回新类型
func GetVisualkPrefixs() []VisualPrefixConfig {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		var varvisualPrefixes []VisualPrefixConfig
		for _, vp := range instance.Settings.VisualPrefixs {
			varvisualPrefixes = append(varvisualPrefixes, VisualPrefixConfig{
				Prefix:          vp.Prefix,
				WhiteList:       vp.WhiteList,
				NoWhiteResponse: vp.NoWhiteResponse,
			})
		}
		return varvisualPrefixes
	}
	return nil // 返回nil，如果instance为nil
}

// 获取LazyMessageId状态
func GetLazyMessageId() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get LazyMessageId value.")
		return false
	}
	return instance.Settings.LazyMessageId
}

// 获取HashID
func GetHashIDValue() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get hashid value.")
		return false
	}
	return instance.Settings.HashID
}

// 获取RemoveAt的值
func GetRemoveAt() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get RemoveAt value.")
		return false
	}
	return instance.Settings.RemoveAt
}

// 获取port的值
func GetPortValue() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get port value.")
		return ""
	}
	return instance.Settings.Port
}

// 获取Array的值
func GetArrayValue() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get array value.")
		return false
	}
	return instance.Settings.Array
}

// 获取AppID
func GetAppID() uint64 {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.AppID
	}
	return 0 // or whatever default value you'd like to return if instance is nil
}

// 获取AppID String
func GetAppIDStr() string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return fmt.Sprintf("%d", instance.Settings.AppID)
	}
	return "0"
}

// 获取WsToken
func GetWsToken() []string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.WsToken
	}
	return nil // 返回nil，如果instance为nil
}

// 获取MasterID数组
func GetMasterID() []string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.MasterID
	}
	return nil // 返回nil，如果instance为nil
}

// 获取port的值
func GetEnableWsServer() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get port value.")
		return false
	}
	return instance.Settings.EnableWsServer
}

// 获取WsServerToken的值
func GetWsServerToken() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get WsServerToken value.")
		return ""
	}
	return instance.Settings.WsServerToken
}

// 获取identify_file的值
func GetIdentifyFile() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get identify file name.")
		return false
	}
	return instance.Settings.IdentifyFile
}

// 获取crt路径
func GetCrtPath() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get crt path.")
		return ""
	}
	return instance.Settings.Crt
}

// 获取key路径
func GetKeyPath() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get key path.")
		return ""
	}
	return instance.Settings.Key
}

// 开发者日志
func GetDeveloperLog() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get developer log status.")
		return false
	}
	return instance.Settings.DeveloperLog
}

// ComposeWebUIURL 组合webui的完整访问地址
// 参数 useBackupPort 控制是否使用备用端口
func ComposeWebUIURL(useBackupPort bool) string {
	serverDir := GetServer_dir()

	var port string
	if useBackupPort {
		port = GetBackupPort()
	} else {
		port = GetPortValue()
	}

	// 判断端口是不是443，如果是，则使用https协议
	protocol := "http"
	if port == "443" {
		protocol = "https"
	}

	// 组合出完整的URL
	return fmt.Sprintf("%s://%s:%s/webui", protocol, serverDir, port)
}

// ComposeWebUIURLv2 组合webui的完整访问地址
// 参数 useBackupPort 控制是否使用备用端口
func ComposeWebUIURLv2(useBackupPort bool) string {
	ip, _ := sys.GetPublicIP()

	var port string
	if useBackupPort {
		port = GetBackupPort()
	} else {
		port = GetPortValue()
	}

	// 判断端口是不是443，如果是，则使用https协议
	protocol := "http"
	if port == "443" {
		protocol = "https"
	}

	// 组合出完整的URL
	return fmt.Sprintf("%s://%s:%s/webui", protocol, ip, port)
}

// GetServerUserName 获取服务器用户名
func GetServerUserName() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get server user name.")
		return ""
	}
	return instance.Settings.Username
}

// GetServerUserPassword 获取服务器用户密码
func GetServerUserPassword() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get server user password.")
		return ""
	}
	return instance.Settings.Password
}

// GetImageLimit 返回 ImageLimit 的值
func GetImageLimit() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get image limit value.")
		return 0 // 或者返回一个默认的 ImageLimit 值
	}

	return instance.Settings.ImageLimit
}

// GetRemovePrefixValue 函数用于获取 remove_prefix 的配置值
func GetRemovePrefixValue() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get remove_prefix value.")
		return false // 或者可能是默认值，取决于您的应用程序逻辑
	}
	return instance.Settings.RemovePrefix
}

// GetLotusPort retrieves the LotusPort setting from your singleton instance.
func GetBackupPort() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get LotusPort.")
		return ""
	}

	return instance.Settings.BackupPort
}

// 获取GetDevMsgID的值
func GetDevMsgID() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetDevMsgID value.")
		return false
	}
	return instance.Settings.DevMessgeID
}

// 获取GetSaveLogs的值
func GetSaveLogs() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetSaveLogs value.")
		return false
	}
	return instance.Settings.SaveLogs
}

// 获取GetSaveLogs的值
func GetLogLevel() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetLogLevel value.")
		return 2
	}
	return instance.Settings.LogLevel
}

// 获取GetBindPrefix的值
func GetBindPrefix() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetBindPrefix value.")
		return "/bind"
	}
	return instance.Settings.BindPrefix
}

// 获取GetMePrefix的值
func GetMePrefix() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetMePrefix value.")
		return "/me"
	}
	return instance.Settings.MePrefix
}

// 获取FrpPort的值
func GetFrpPort() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetFrpPort value.")
		return "0"
	}
	return instance.Settings.FrpPort
}

// 获取GetRemoveBotAtGroup的值
func GetRemoveBotAtGroup() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetRemoveBotAtGroup value.")
		return false
	}
	return instance.Settings.RemoveBotAtGroup
}

// 获取ImageLimitB的值
func GetImageLimitB() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to ImageLimitB value.")
		return 100
	}
	return instance.Settings.ImageLimitB
}

// GetRecordSampleRate 返回 RecordSampleRate的值
func GetRecordSampleRate() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetRecordSampleRate value.")
		return 0 // 或者返回一个默认的 ImageLimit 值
	}

	return instance.Settings.RecordSampleRate
}

// GetRecordBitRate 返回 RecordBitRate
func GetRecordBitRate() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetRecordBitRate value.")
		return 0 // 或者返回一个默认的 ImageLimit 值
	}

	return instance.Settings.RecordBitRate
}

// 获取NoWhiteResponse的值
func GetNoWhiteResponse() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to NoWhiteResponse value.")
		return ""
	}
	return instance.Settings.NoWhiteResponse
}

// 获取GetSendError的值
func GetSendError() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetSendError value.")
		return true
	}
	return instance.Settings.SendError
}

// 获取GetAddAtGroup的值
func GetAddAtGroup() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetAddGroupAt value.")
		return true
	}
	return instance.Settings.AddAtGroup
}

// 获取GetUrlPicTransfer的值
func GetUrlPicTransfer() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetUrlPicTransfer value.")
		return true
	}
	return instance.Settings.UrlPicTransfer
}

// 获取GetLotusPassword的值
func GetLotusPassword() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetLotusPassword value.")
		return ""
	}
	return instance.Settings.LotusPassword
}

// 获取GetWsServerPath的值
func GetWsServerPath() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetWsServerPath value.")
		return ""
	}
	return instance.Settings.WsServerPath
}

// 获取GetIdmapPro的值
func GetIdmapPro() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetIdmapPro value.")
		return false
	}
	return instance.Settings.IdmapPro
}

// 获取GetCardAndNick的值
func GetCardAndNick() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetCardAndNick value.")
		return ""
	}
	return instance.Settings.CardAndNick
}

// 获取GetAutoBind的值
func GetAutoBind() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetAutoBind value.")
		return false
	}
	return instance.Settings.AutoBind
}

// 获取GetCustomBotName的值
func GetCustomBotName() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetCustomBotName value.")
		return "Gensokyo全域机器人"
	}
	return instance.Settings.CustomBotName
}

// 获取send_delay的值
func GetSendDelay() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetSendDelay value.")
		return 300
	}
	return instance.Settings.SendDelay
}

// 获取GetAtoPCount的值
func GetAtoPCount() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to AtoPCount value.")
		return 5
	}
	return instance.Settings.AtoPCount
}

// 获取GetReconnecTimes的值
func GetReconnecTimes() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to ReconnecTimes value.")
		return 50
	}
	return instance.Settings.ReconnecTimes
}

// 获取GetHeartBeatInterval的值
func GetHeartBeatInterval() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to HeartBeatInterval value.")
		return 5
	}
	return instance.Settings.HeartBeatInterval
}

// 获取LaunchReconectTimes
func GetLaunchReconectTimes() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to LaunchReconectTimes value.")
		return 3
	}
	return instance.Settings.LaunchReconectTimes
}

// 获取GetUnlockPrefix
func GetUnlockPrefix() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to UnlockPrefix value.")
		return "/unlock"
	}
	return instance.Settings.UnlockPrefix
}

// 获取白名单例外群数组
func GetWhiteBypass() []int64 {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.WhiteBypass
	}
	return nil // 返回nil，如果instance为nil
}

// 获取GetTransferUrl的值
func GetTransferUrl() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetTransferUrl value.")
		return false
	}
	return instance.Settings.TransferUrl
}

// 获取 HTTP 地址
func GetHttpAddress() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get HTTP address.")
		return ""
	}
	return instance.Settings.HttpAddress
}

// 获取 HTTP 版本
func GetHttpVersion() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get HTTP version.")
		return 11
	}
	return instance.Settings.HttpVersion
}

// 获取 HTTP 超时时间
func GetHttpTimeOut() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get HTTP timeout.")
		return 5
	}
	return instance.Settings.HttpTimeOut
}

// 获取 POST URL 数组
func GetPostUrl() []string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get POST URL.")
		return nil
	}
	return instance.Settings.PostUrl
}

// 获取 POST 密钥数组
func GetPostSecret() []string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get POST secret.")
		return nil
	}
	return instance.Settings.PostSecret
}

// 获取 POST 最大重试次数数组
func GetPostMaxRetries() []int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get POST max retries.")
		return nil
	}
	return instance.Settings.PostMaxRetries
}

// 获取 POST 重试间隔数组
func GetPostRetriesInterval() []int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get POST retries interval.")
		return nil
	}
	return instance.Settings.PostRetriesInterval
}

// 获取GetTransferUrl的值
func GetNativeOb11() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to NativeOb11 value.")
		return false
	}
	return instance.Settings.NativeOb11
}

// 获取GetRamDomSeq的值
func GetRamDomSeq() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetRamDomSeq value.")
		return false
	}
	return instance.Settings.RamDomSeq
}

// 获取GetUrlToQrimage的值
func GetUrlToQrimage() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetUrlToQrimage value.")
		return false
	}
	return instance.Settings.UrlToQrimage
}

// 获取GetQrSize的值
func GetQrSize() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to QrSize value.")
		return 200
	}
	return instance.Settings.QrSize
}

func replaceVisualPrefixsLine(configData []byte) ([]byte, bool) {
	// 定义新的 visual_prefixs 部分
	newVisualPrefixs := `  visual_prefixs :                  #虚拟前缀 与white_prefixs配合使用 处理流程自动忽略该前缀 remove_prefix remove_at 需为true时生效
  - prefix: ""                      #虚拟前缀开头 例 你有3个指令 帮助 测试 查询 将 prefix 设置为 工具类 后 则可通过 工具类 帮助 触发机器人
    whiteList: [""]                 #开关状态取决于 white_prefix_mode 为每一个二级指令头设计独立的白名单
    No_White_Response : "" 
  - prefix: ""
    whiteList: [""]
    No_White_Response : "" 
  - prefix: ""
    whiteList: [""]
    No_White_Response : "" `

	// 将 byte 数组转换为字符串
	configStr := string(configData)

	// 按行分割 configStr
	lines := strings.Split(configStr, "\n")

	// 创建一个新的字符串构建器
	var newConfigData strings.Builder

	// 标记是否进行了替换
	replaced := false

	// 遍历所有行
	for _, line := range lines {
		// 检查是否是 visual_prefixs 开头的行
		if strings.HasPrefix(strings.TrimSpace(line), "visual_prefixs : [") {
			// 替换为新的 visual_prefixs 部分
			newConfigData.WriteString(newVisualPrefixs + "\n")
			replaced = true
			continue // 跳过原有行
		}
		newConfigData.WriteString(line + "\n")
	}

	// 返回新配置和是否发生了替换的标记
	return []byte(newConfigData.String()), replaced
}

// 获取GetWhiteBypassRevers的值
func GetWhiteBypassRevers() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GetWhiteBypassRevers value.")
		return false
	}
	return instance.Settings.WhiteBypassRevers
}

// 获取GetGuildUrlImageToBase64的值
func GetGuildUrlImageToBase64() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to GuildUrlImageToBase64 value.")
		return false
	}
	return instance.Settings.GuildUrlImageToBase64
}

// GetTencentBucketURL 获取 TencentBucketURL
func GetTencentBucketURL() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get TencentBucketURL.")
		return ""
	}

	bucketName := instance.Settings.TencentBucketName
	bucketRegion := instance.Settings.TencentBucketRegion

	// 构建并返回URL
	if bucketName == "" || bucketRegion == "" {
		mylog.Println("Warning: Tencent bucket name or region is not configured.")
		return ""
	}

	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucketName, bucketRegion)
}

// GetTencentCosSecretid 获取 TencentCosSecretid
func GetTencentCosSecretid() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get TencentCosSecretid.")
		return ""
	}
	return instance.Settings.TencentCosSecretid
}

// GetTencentSecretKey 获取 TencentSecretKey
func GetTencentSecretKey() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get TencentSecretKey.")
		return ""
	}
	return instance.Settings.TencentSecretKey
}

// 获取GetTencentAudit的值
func GetTencentAudit() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to TencentAudit value.")
		return false
	}
	return instance.Settings.TencentAudit
}

// 获取 Oss 模式
func GetOssType() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get ExtraPicAuditingType version.")
		return 0
	}
	return instance.Settings.OssType
}

// 获取BaiduBOSBucketName
func GetBaiduBOSBucketName() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get BaiduBOSBucketName.")
		return ""
	}
	return instance.Settings.BaiduBOSBucketName
}

// 获取BaiduBCEAK
func GetBaiduBCEAK() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get BaiduBCEAK.")
		return ""
	}
	return instance.Settings.BaiduBCEAK
}

// 获取BaiduBCESK
func GetBaiduBCESK() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get BaiduBCESK.")
		return ""
	}
	return instance.Settings.BaiduBCESK
}

// 获取BaiduAudit
func GetBaiduAudit() int {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get BaiduAudit.")
		return 0
	}
	return instance.Settings.BaiduAudit
}

// 获取阿里云的oss地址 外网的
func GetAliyunEndpoint() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get AliyunEndpoint.")
		return ""
	}
	return instance.Settings.AliyunEndpoint
}

// GetRegionID 从 AliyunEndpoint 获取 regionId
func GetRegionID() string {
	endpoint := GetAliyunEndpoint()
	if endpoint == "" {
		return ""
	}

	// 去除协议头（如 "https://"）
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// 将 endpoint 按照 "." 分割
	parts := strings.Split(endpoint, ".")
	if len(parts) >= 2 {
		// 第一部分应该是包含 regionId 的信息（例如 "oss-cn-hangzhou"）
		regionInfo := parts[0]
		// 进一步提取 regionId
		regionParts := strings.SplitN(regionInfo, "-", 3)
		if len(regionParts) >= 3 {
			// 返回 "cn-hangzhou" 部分
			return regionParts[1] + "-" + regionParts[2]
		}
	}
	return ""
}

// GetAliyunAccessKeyId 获取阿里云OSS的AccessKeyId
func GetAliyunAccessKeyId() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get AliyunAccessKeyId.")
		return ""
	}
	return instance.Settings.AliyunAccessKeyId
}

// GetAliyunAccessKeySecret 获取阿里云OSS的AccessKeySecret
func GetAliyunAccessKeySecret() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get AliyunAccessKeySecret.")
		return ""
	}
	return instance.Settings.AliyunAccessKeySecret
}

// GetAliyunBucketName 获取阿里云OSS的AliyunBucketName
func GetAliyunBucketName() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to get AliyunBucketName.")
		return ""
	}
	return instance.Settings.AliyunBucketName
}

// 获取GetAliyunAudit的值
func GetAliyunAudit() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		mylog.Println("Warning: instance is nil when trying to AliyunAudit value.")
		return false
	}
	return instance.Settings.AliyunAudit
}

// 获取Alias的值
func GetAlias() []string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.Alias
	}
	return nil // 返回nil，如果instance为nil
}

// 获取SelfIntroduce的值
func GetSelfIntroduce() []string {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance.Settings.SelfIntroduce
	}
	return nil // 返回nil，如果instance为nil
}

// 获取WhiteEnable的值
func GetWhiteEnable(index int) bool {
	mu.Lock()
	defer mu.Unlock()

	// 检查instance或instance.Settings.WhiteEnable是否为nil
	if instance == nil || instance.Settings.WhiteEnable == nil {
		return true // 如果为nil，返回默认值true
	}

	// 调整索引以符合从0开始的数组索引
	adjustedIndex := index - 1

	// 检查索引是否在数组范围内
	if adjustedIndex >= 0 && adjustedIndex < len(instance.Settings.WhiteEnable) {
		return instance.Settings.WhiteEnable[adjustedIndex]
	}

	// 如果索引超出范围，返回默认值true
	return true
}
