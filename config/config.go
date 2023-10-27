// config/config.go

package config

import (
	"fmt"
	"log"
	"os"
	"sync"

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
		log.Println("Warning: instance is nil when trying to get upload directory.")
		return ""
	}
	return instance.Settings.Server_dir
}

// 获取lotus的值
func GetLotusValue() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		log.Println("Warning: instance is nil when trying to get lotus value.")
		return false
	}
	return instance.Settings.Lotus
}

// 获取port的值
func GetPortValue() string {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		log.Println("Warning: instance is nil when trying to get port value.")
		return ""
	}
	return instance.Settings.Port
}

// 获取Array的值
func GetArrayValue() bool {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		log.Println("Warning: instance is nil when trying to get array value.")
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
		log.Println("Warning: instance is nil when trying to get port value.")
		return false
	}
	return instance.Settings.EnableWsServer
}
