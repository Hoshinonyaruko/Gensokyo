package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

type Settings struct {
	WsAddress              []string `yaml:"ws_address"`
	AppID                  uint64   `yaml:"app_id"`
	Token                  string   `yaml:"token"`
	ClientSecret           string   `yaml:"client_secret"`
	TextIntent             []string `yaml:"text_intent"`
	GlobalChannelToGroup   bool     `yaml:"global_channel_to_group"`
	GlobalPrivateToChannel bool     `yaml:"global_private_to_channel"`
	Array                  bool     `yaml:"array"`
	Server_dir             string   `yaml:"server_dir"`
	Lotus                  bool     `yaml:"lotus"`
	Port                   string   `yaml:"port"`
	WsToken                []string `yaml:"ws_token,omitempty"`         // 连接wss时使用,不是wss可留空 一一对应
	MasterID               []string `yaml:"master_id,omitempty"`        // 如果需要在群权限判断是管理员是,将user_id填入这里,master_id是一个文本数组
	EnableWsServer         bool     `yaml:"enable_ws_server,omitempty"` //正向ws开关
	WsServerToken          string   `yaml:"ws_server_token,omitempty"`  //正向ws token
	IdentifyFile           bool     `yaml:"identify_file"`              // 域名校验文件
	Crt                    string   `yaml:"crt"`
	Key                    string   `yaml:"key"`
	DeveloperLog           bool     `yaml:"developer_log"`
	Username               string   `yaml:"server_user_name"`
	Password               string   `yaml:"server_user_password"`
	ImageLimit             int      `yaml:"image_sizelimit"`
}

// LoadConfig 从文件中加载配置并初始化单例配置
func LoadConfig(path string) (*Config, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = yaml.Unmarshal(configData, conf)
	if err != nil {
		return nil, err
	}

	mu.Lock()
	if instance == nil {
		instance = conf
	}
	mu.Unlock()

	return conf, nil
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
func ComposeWebUIURL() string {
	serverDir := GetServer_dir()
	port := GetPortValue()

	// 判断端口是不是443，如果是，则使用https协议
	protocol := "http"
	if port == "443" {
		protocol = "https"
	}

	// 组合出完整的URL
	return fmt.Sprintf("%s://%s:%s/webui", protocol, serverDir, port)
}

// ComposeWebUIURL 组合webui的完整访问地址
func ComposeWebUIURLv2() string {
	ip, _ := sys.GetPublicIP()

	port := GetPortValue()

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
