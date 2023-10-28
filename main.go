package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hoshinonyaruko/gensokyo/Processor"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/server"
	"github.com/hoshinonyaruko/gensokyo/wsclient"

	"github.com/gin-gonic/gin"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"github.com/tencent-connect/botgo/websocket"
)

// 消息处理器，持有 openapi 对象
var p *Processor.Processors

func main() {
	if _, err := os.Stat("config.yml"); os.IsNotExist(err) {
		err = os.WriteFile("config.yml", []byte(configTemplate), 0644)
		if err != nil {
			fmt.Println("Error writing config.yml:", err)
			return
		}

		fmt.Println("请配置config.yml然后再次运行.")
		fmt.Print("按下 Enter 继续...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		os.Exit(0)
	}

	// 主逻辑
	// 加载配置
	conf, err := config.LoadConfig("config.yml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	//获取bot的token
	token := token.BotToken(conf.Settings.AppID, conf.Settings.ClientSecret, conf.Settings.Token, token.TypeQQBot)

	ctx := context.Background()
	if err := token.InitToken(ctx); err != nil {
		log.Fatalln(err)
	}

	//读取intent
	if len(conf.Settings.TextIntent) == 0 {
		// 如果 TextIntent 数组为空，抛出错误
		panic(errors.New("TextIntent is empty, at least one intent should be specified"))
	}

	// 创建 v1 版本的 OpenAPI 实例
	if err := botgo.SelectOpenAPIVersion(openapi.APIv1); err != nil {
		log.Fatalln(err)
	}
	api := botgo.NewOpenAPI(token).WithTimeout(3 * time.Second)

	// 创建 v2 版本的 OpenAPI 实例
	if err := botgo.SelectOpenAPIVersion(openapi.APIv2); err != nil {
		log.Fatalln(err)
	}
	apiV2 := botgo.NewOpenAPI(token).WithTimeout(3 * time.Second)

	// 执行API请求 显示机器人信息
	me, err := api.Me(ctx) // Adjusted to pass only the context
	if err != nil {
		fmt.Printf("Error fetching bot details: %v\n", err)
		return
	}
	fmt.Printf("Bot details: %+v\n", me)

	//初始化handlers
	handlers.BotID = me.ID
	//handlers.BotID = "1234"
	handlers.AppID = fmt.Sprintf("%d", conf.Settings.AppID)

	// 获取 websocket 信息 这里用哪一个api获取就是用哪一个api去连接ws
	// 测试群时候用api2 并且要注释掉api.me
	//似乎正式场景都可以用apiv2(群)的方式获取ws连接,包括频道的机器人
	//疑问: 为什么无法用apiv2的方式调用频道的getme接口,会报错
	wsInfo, err := apiV2.WS(ctx, nil, "")
	if err != nil {
		log.Fatalln(err)
	}

	// 定义和初始化intent变量
	var intent dto.Intent = 0

	//动态订阅intent
	for _, handlerName := range conf.Settings.TextIntent {
		handler, ok := getHandlerByName(handlerName)
		if !ok {
			fmt.Printf("Unknown handler: %s\n", handlerName)
			continue
		}

		//多次位与 并且订阅事件
		intent |= websocket.RegisterHandlers(handler)
	}

	fmt.Printf("注册 intents: %v\n", intent)

	// 启动session manager以管理websocket连接
	// 指定需要启动的分片数为 2 的话可以手动修改 wsInfo
	go func() {
		wsInfo.Shards = 1
		if err = botgo.NewSessionManager().Start(wsInfo, token, &intent); err != nil {
			log.Fatalln(err)
		}
	}()

	// 启动多个WebSocket客户端
	wsClients := []*wsclient.WebSocketClient{}
	wsClientChan := make(chan *wsclient.WebSocketClient, len(conf.Settings.WsAddress))
	errorChan := make(chan error, len(conf.Settings.WsAddress))

	for _, wsAddr := range conf.Settings.WsAddress {
		go func(address string) {
			wsClient, err := wsclient.NewWebSocketClient(address, conf.Settings.AppID, api, apiV2)
			if err != nil {
				fmt.Printf("Error creating WebSocketClient for address %s: %v\n", address, err)
				errorChan <- err
				return
			}
			wsClientChan <- wsClient
		}(wsAddr)
	}

	// 获取连接成功后的wsClient
	for i := 0; i < len(conf.Settings.WsAddress); i++ {
		select {
		case wsClient := <-wsClientChan:
			wsClients = append(wsClients, wsClient)
		case err := <-errorChan:
			fmt.Printf("Error encountered while initializing WebSocketClient: %v\n", err)
		}
	}

	// 确保所有wsClients都已初始化
	if len(wsClients) != len(conf.Settings.WsAddress) {
		fmt.Println("Error: Not all wsClients are initialized!")
		log.Fatalln("Failed to initialize all WebSocketClients.")
	} else {
		fmt.Println("All wsClients are successfully initialized.")
		p = Processor.NewProcessor(api, apiV2, &conf.Settings, wsClients)
	}

	//创建idmap服务器
	idmap.InitializeDB()
	defer idmap.CloseDB()

	//图片上传 调用次数限制
	rateLimiter := server.NewRateLimiter()
	//是否启动服务器
	shouldStartServer := !conf.Settings.Lotus || conf.Settings.EnableWsServer
	//如果连接到其他gensokyo,则不需要启动服务器
	if shouldStartServer {
		r := gin.Default()
		r.GET("/getid", server.GetIDHandler)
		r.POST("/uploadpic", server.UploadBase64ImageHandler(rateLimiter))
		r.Static("/channel_temp", "./channel_temp")
		r.GET("/ws", server.WsHandlerWithDependencies(api, apiV2, p))
		r.Run("0.0.0.0:" + conf.Settings.Port) // 监听0.0.0.0地址的Port端口
	}

	// 使用通道来等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞主线程，直到接收到信号
	<-sigCh

	// 关闭 WebSocket 连接
	// wsClients 是一个 *wsclient.WebSocketClient 的切片
	for _, client := range wsClients {
		err := client.Close()
		if err != nil {
			fmt.Printf("Error closing WebSocket connection: %v\n", err)
		}
	}
}

// ReadyHandler 自定义 ReadyHandler 感知连接成功事件
func ReadyHandler() event.ReadyHandler {
	return func(event *dto.WSPayload, data *dto.WSReadyData) {
		log.Println("连接成功,ready event receive: ", data)
	}
}

// ErrorNotifyHandler 处理当 ws 链接发送错误的事件
func ErrorNotifyHandler() event.ErrorNotifyHandler {
	return func(err error) {
		log.Println("error notify receive: ", err)
	}
}

// ATMessageEventHandler 实现处理 频道at 消息的回调
func ATMessageEventHandler() event.ATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSATMessageData) error {
		return p.ProcessGuildATMessage(data)
	}
}

// GuildEventHandler 处理频道事件
func GuildEventHandler() event.GuildEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGuildData) error {
		fmt.Println(data)
		return nil
	}
}

// ChannelEventHandler 处理子频道事件
func ChannelEventHandler() event.ChannelEventHandler {
	return func(event *dto.WSPayload, data *dto.WSChannelData) error {
		fmt.Println(data)
		return nil
	}
}

// MemberEventHandler 处理成员变更事件
func MemberEventHandler() event.GuildMemberEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGuildMemberData) error {
		fmt.Println(data)
		return nil
	}
}

// DirectMessageHandler 处理私信事件
func DirectMessageHandler() event.DirectMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSDirectMessageData) error {
		return p.ProcessChannelDirectMessage(data)
	}
}

// CreateMessageHandler 处理消息事件 私域的事件 不at信息
func CreateMessageHandler() event.MessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSMessageData) error {
		fmt.Println("收到私域信息", data)
		return p.ProcessGuildNormalMessage(data)
	}
}

// InteractionHandler 处理内联交互事件
func InteractionHandler() event.InteractionEventHandler {
	return func(event *dto.WSPayload, data *dto.WSInteractionData) error {
		fmt.Println(data)
		return p.ProcessInlineSearch(data)
	}
}

// GroupATMessageEventHandler 实现处理 群at 消息的回调
func GroupATMessageEventHandler() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		return p.ProcessGroupMessage(data)
	}
}

// C2CMessageEventHandler 实现处理 群私聊 消息的回调
func C2CMessageEventHandler() event.C2CMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
		return p.ProcessC2CMessage(data)
	}
}

func getHandlerByName(handlerName string) (interface{}, bool) {
	switch handlerName {
	case "ReadyHandler": //连接成功
		return ReadyHandler(), true
	case "ErrorNotifyHandler": //连接关闭
		return ErrorNotifyHandler(), true
	case "ATMessageEventHandler": //频道at信息
		return ATMessageEventHandler(), true
	case "GuildEventHandler": //频道事件
		return GuildEventHandler(), true
	case "MemberEventHandler": //频道成员新增
		return MemberEventHandler(), true
	case "ChannelEventHandler": //频道事件
		return ChannelEventHandler(), true
	case "DirectMessageHandler": //私域频道私信(dms)
		return DirectMessageHandler(), true
	case "CreateMessageHandler": //频道不at信息
		return CreateMessageHandler(), true
	case "InteractionHandler": //添加频道互动回应
		return InteractionHandler(), true
	case "ThreadEventHandler": //发帖事件 暂不支持 暂不支持
		return nil, false
		//return ThreadEventHandler(), true
	case "GroupATMessageEventHandler": //群at信息
		return GroupATMessageEventHandler(), true
	case "C2CMessageEventHandler": //群私聊
		return C2CMessageEventHandler(), true
	default:
		fmt.Printf("Unknown handler: %s\n", handlerName)
		return nil, false
	}
}
