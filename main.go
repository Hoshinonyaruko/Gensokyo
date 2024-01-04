package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/hoshinonyaruko/gensokyo/Processor"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/httpapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/server"
	"github.com/hoshinonyaruko/gensokyo/sys"
	"github.com/hoshinonyaruko/gensokyo/template"
	"github.com/hoshinonyaruko/gensokyo/url"
	"github.com/hoshinonyaruko/gensokyo/webui"
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
	// 定义faststart命令行标志。默认为false。
	fastStart := flag.Bool("faststart", false, "start without initialization if set")

	// 解析命令行参数到定义的标志。
	flag.Parse()

	// 检查是否使用了-faststart参数
	if !*fastStart {
		sys.InitBase() // 如果不是faststart模式，则执行初始化
	}
	if _, err := os.Stat("config.yml"); os.IsNotExist(err) {
		var ip string
		var err error
		// 检查操作系统是否为Android
		if runtime.GOOS == "android" {
			ip = "127.0.0.1"
		} else {
			// 获取内网IP地址
			ip, err = sys.GetLocalIP()
			if err != nil {
				log.Println("Error retrieving the local IP address:", err)
				ip = "127.0.0.1"
			}
		}
		// 将 <YOUR_SERVER_DIR> 替换成实际的内网IP地址 确保初始状态webui能够被访问
		configData := strings.Replace(template.ConfigTemplate, "<YOUR_SERVER_DIR>", ip, -1)

		// 将修改后的配置写入 config.yml
		err = os.WriteFile("config.yml", []byte(configData), 0644)
		if err != nil {
			log.Println("Error writing config.yml:", err)
			return
		}

		log.Println("请配置config.yml然后再次运行.")
		log.Print("按下 Enter 继续...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		os.Exit(0)
	}

	// 主逻辑
	// 加载配置
	conf, err := config.LoadConfig("config.yml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	sys.SetTitle(conf.Settings.Title)
	webuiURL := config.ComposeWebUIURL(conf.Settings.Lotus)     // 调用函数获取URL
	webuiURLv2 := config.ComposeWebUIURLv2(conf.Settings.Lotus) // 调用函数获取URL

	var api openapi.OpenAPI
	var apiV2 openapi.OpenAPI
	var wsClients []*wsclient.WebSocketClient
	var nologin bool

	//logger
	logLevel := mylog.GetLogLevelFromConfig(config.GetLogLevel())
	loggerAdapter := mylog.NewMyLogAdapter(logLevel, config.GetSaveLogs())
	botgo.SetLogger(loggerAdapter)

	if conf.Settings.AppID == 12345 {
		// 输出天蓝色文本
		cyan := color.New(color.FgCyan)
		cyan.Printf("欢迎来到Gensokyo, 控制台地址: %s\n", webuiURL)
		log.Println("请完成机器人配置后重启框架。")

	} else {
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

		//创建api
		if !conf.Settings.SandBoxMode {
			// 创建 v1 版本的 OpenAPI 实例
			if err := botgo.SelectOpenAPIVersion(openapi.APIv1); err != nil {
				log.Fatalln(err)
			}
			api = botgo.NewOpenAPI(token).WithTimeout(3 * time.Second)
			log.Println("创建 apiv1 成功")

			// 创建 v2 版本的 OpenAPI 实例
			if err := botgo.SelectOpenAPIVersion(openapi.APIv2); err != nil {
				log.Fatalln(err)
			}
			apiV2 = botgo.NewOpenAPI(token).WithTimeout(3 * time.Second)
			log.Println("创建 apiv2 成功")
		} else {
			// 创建 v1 版本的 OpenAPI 实例
			if err := botgo.SelectOpenAPIVersion(openapi.APIv1); err != nil {
				log.Fatalln(err)
			}
			api = botgo.NewSandboxOpenAPI(token).WithTimeout(3 * time.Second)
			log.Println("创建 沙箱 apiv1 成功")

			// 创建 v2 版本的 OpenAPI 实例
			if err := botgo.SelectOpenAPIVersion(openapi.APIv2); err != nil {
				log.Fatalln(err)
			}
			apiV2 = botgo.NewSandboxOpenAPI(token).WithTimeout(3 * time.Second)
			log.Println("创建 沙箱 apiv2 成功")
		}

		configURL := config.GetDevelop_Acdir()
		var me *dto.User
		if configURL == "" { // 执行API请求 显示机器人信息
			me, err = api.Me(ctx) // Adjusted to pass only the context
			if err != nil {
				log.Printf("Error fetching bot details: %v\n", err)
				//return
				nologin = true
			}
			log.Printf("Bot details: %+v\n", me)
		} else {
			log.Printf("自定义ac地址模式...请从日志手动获取bot的真实id并设置,不然at会不正常")
		}
		if !nologin {
			if configURL == "" { //初始化handlers
				handlers.BotID = me.ID
			} else { //初始化handlers
				handlers.BotID = config.GetDevBotid()
			}

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
					log.Printf("Unknown handler: %s\n", handlerName)
					continue
				}

				//多次位与 并且订阅事件
				intent |= websocket.RegisterHandlers(handler)
			}

			log.Printf("注册 intents: %v\n", intent)

			// 启动session manager以管理websocket连接
			// 指定需要启动的分片数为 2 的话可以手动修改 wsInfo
			go func() {
				wsInfo.Shards = 1
				if err = botgo.NewSessionManager().Start(wsInfo, token, &intent); err != nil {
					log.Fatalln(err)
				}
			}()

			// 启动多个WebSocket客户端的逻辑
			if !allEmpty(conf.Settings.WsAddress) {
				wsClientChan := make(chan *wsclient.WebSocketClient, len(conf.Settings.WsAddress))
				errorChan := make(chan error, len(conf.Settings.WsAddress))
				// 定义计数器跟踪尝试建立的连接数
				attemptedConnections := 0
				for _, wsAddr := range conf.Settings.WsAddress {
					if wsAddr == "" {
						continue // Skip empty addresses
					}
					attemptedConnections++ // 增加尝试连接的计数
					go func(address string) {
						retry := config.GetLaunchReconectTimes()
						wsClient, err := wsclient.NewWebSocketClient(address, conf.Settings.AppID, api, apiV2, retry)
						if err != nil {
							log.Printf("Error creating WebSocketClient for address(连接到反向ws失败) %s: %v\n", address, err)
							errorChan <- err
							return
						}
						wsClientChan <- wsClient
					}(wsAddr)
				}
				// 获取连接成功后的wsClient
				for i := 0; i < attemptedConnections; i++ {
					select {
					case wsClient := <-wsClientChan:
						wsClients = append(wsClients, wsClient)
					case err := <-errorChan:
						log.Printf("Error encountered while initializing WebSocketClient: %v\n", err)
					}
				}

				// 确保所有尝试建立的连接都有对应的wsClient
				if len(wsClients) == 0 {
					log.Println("Error: Not all wsClients are initialized!(反向ws未设置或全部连接失败)")
					// 处理连接失败的情况 只启动正向
					p = Processor.NewProcessorV2(api, apiV2, &conf.Settings)
				} else {
					log.Println("All wsClients are successfully initialized.")
					// 所有客户端都成功初始化
					p = Processor.NewProcessor(api, apiV2, &conf.Settings, wsClients)
				}
			} else if conf.Settings.EnableWsServer {
				log.Println("只启动正向ws")
				p = Processor.NewProcessorV2(api, apiV2, &conf.Settings)
			}
		} else {
			// 设置颜色为红色
			red := color.New(color.FgRed)
			// 输出红色文本
			red.Println("请设置正确的appid、token、clientsecret再试")
		}

	}

	//创建idmap服务器 数据库
	idmap.InitializeDB()
	//创建webui数据库
	webui.InitializeDB()
	defer idmap.CloseDB()
	defer webui.CloseDB()

	//图片上传 调用次数限制
	rateLimiter := server.NewRateLimiter()
	// 根据 lotus 的值选择端口
	var serverPort string
	if !conf.Settings.Lotus {
		serverPort = conf.Settings.Port
	} else {
		serverPort = conf.Settings.BackupPort
	}
	var r *gin.Engine
	var hr *gin.Engine
	if config.GetDeveloperLog() { // 是否启动调试状态
		r = gin.Default()
		hr = gin.Default()
	} else {
		r = gin.New()
		r.Use(gin.Recovery()) // 添加恢复中间件，但不添加日志中间件
		hr = gin.New()
		hr.Use(gin.Recovery())
	}
	r.GET("/getid", server.GetIDHandler)
	r.GET("/updateport", server.HandleIpupdate)
	r.POST("/uploadpic", server.UploadBase64ImageHandler(rateLimiter))
	r.POST("/uploadrecord", server.UploadBase64RecordHandler(rateLimiter))
	r.Static("/channel_temp", "./channel_temp")
	if config.GetFrpPort() == "0" {
		//webui和它的api
		webuiGroup := r.Group("/webui")
		{
			webuiGroup.GET("/*filepath", webui.CombinedMiddleware(api, apiV2))
			webuiGroup.POST("/*filepath", webui.CombinedMiddleware(api, apiV2))
			webuiGroup.PUT("/*filepath", webui.CombinedMiddleware(api, apiV2))
			webuiGroup.DELETE("/*filepath", webui.CombinedMiddleware(api, apiV2))
			webuiGroup.PATCH("/*filepath", webui.CombinedMiddleware(api, apiV2))
		}
	}
	//正向http api
	http_api_address := config.GetHttpAddress()
	if http_api_address != "" {
		mylog.Println("正向http api启动成功,监听" + http_api_address + "若有需要,请对外放通端口...")
		HttpApiGroup := hr.Group("/")
		{
			HttpApiGroup.GET("/*filepath", httpapi.CombinedMiddleware(api, apiV2))
			HttpApiGroup.POST("/*filepath", httpapi.CombinedMiddleware(api, apiV2))
			HttpApiGroup.PUT("/*filepath", httpapi.CombinedMiddleware(api, apiV2))
			HttpApiGroup.DELETE("/*filepath", httpapi.CombinedMiddleware(api, apiV2))
			HttpApiGroup.PATCH("/*filepath", httpapi.CombinedMiddleware(api, apiV2))
		}
	}
	//正向ws
	if conf.Settings.AppID != 12345 {
		if conf.Settings.EnableWsServer {
			wspath := config.GetWsServerPath()
			if wspath == "nil" {
				r.GET("", server.WsHandlerWithDependencies(api, apiV2, p))
				mylog.Println("正向ws启动成功,监听0.0.0.0:" + serverPort + "请注意设置ws_server_token(可空),并对外放通端口...")
			} else {
				r.GET("/"+wspath, server.WsHandlerWithDependencies(api, apiV2, p))
				mylog.Println("正向ws启动成功,监听0.0.0.0:" + serverPort + "/" + wspath + "请注意设置ws_server_token(可空),并对外放通端口...")
			}
		}
	}
	r.POST("/url", url.CreateShortURLHandler)
	r.GET("/url/:shortURL", url.RedirectFromShortURLHandler)
	if config.GetIdentifyFile() {
		appIDStr := config.GetAppIDStr()
		fileName := appIDStr + ".json"
		r.GET("/"+fileName, func(c *gin.Context) {
			content := fmt.Sprintf(`{"bot_appid":%d}`, config.GetAppID())
			c.Header("Content-Type", "application/json")
			c.String(200, content)
		})

		// 调用 config.GetIdentifyAppids 获取 appid 数组
		identifyAppids := config.GetIdentifyAppids()

		// 如果 identifyAppids 不是 nil 且有多个元素
		if len(identifyAppids) >= 1 {
			// 从数组中去除 config.GetAppID() 来避免重复
			var filteredAppids []int64
			for _, appid := range identifyAppids {
				if appid != int64(config.GetAppID()) {
					filteredAppids = append(filteredAppids, appid)
				}
			}

			// 为每个 appid 设置路由
			for _, appid := range filteredAppids {
				fileName := fmt.Sprintf("%d.json", appid)
				r.GET("/"+fileName, func(c *gin.Context) {
					content := fmt.Sprintf(`{"bot_appid":%d}`, appid)
					c.Header("Content-Type", "application/json")
					c.String(200, content)
				})
			}
		}
	}
	// 创建一个http.Server实例（主服务器）
	httpServer := &http.Server{
		Addr:    "0.0.0.0:" + serverPort,
		Handler: r,
	}
	mylog.Printf("gin运行在%v端口", serverPort)
	// 在一个新的goroutine中启动主服务器
	go func() {
		if serverPort == "443" {
			// 使用HTTPS
			crtPath := config.GetCrtPath()
			keyPath := config.GetKeyPath()
			if crtPath == "" || keyPath == "" {
				log.Fatalf("crt or key path is missing for HTTPS")
				return
			}
			if err := httpServer.ListenAndServeTLS(crtPath, keyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen (HTTPS): %s\n", err)
			}
		} else {
			// 使用HTTP
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}
	}()

	// 如果主服务器使用443端口，同时在一个新的goroutine中启动444端口的HTTP服务器 todo 更优解
	if serverPort == "443" {
		go func() {
			// 创建另一个http.Server实例（用于444端口）
			httpServer444 := &http.Server{
				Addr:    "0.0.0.0:444",
				Handler: r,
			}

			// 启动444端口的HTTP服务器
			if err := httpServer444.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen (HTTP 444): %s\n", err)
			}
		}()
	}
	// 创建 httpapi 的http server
	if http_api_address != "" {
		go func() {
			// 创建一个http.Server实例（Http Api服务器）
			httpServerHttpApi := &http.Server{
				Addr:    http_api_address,
				Handler: hr,
			}
			// 使用HTTP
			if err := httpServerHttpApi.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("http apilisten: %s\n", err)
			}
		}()
	}

	// 使用color库输出天蓝色的文本
	cyan := color.New(color.FgCyan)
	cyan.Printf("欢迎来到Gensokyo, 控制台地址: %s\n", webuiURL)
	cyan.Printf("%s\n", template.Logo)
	cyan.Printf("欢迎来到Gensokyo, 公网控制台地址(需开放端口): %s\n", webuiURLv2)

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
			log.Printf("Error closing WebSocket connection: %v\n", err)
		}
	}

	// 关闭BoltDB数据库
	url.CloseDB()
	idmap.CloseDB()

	// 在关闭WebSocket客户端之前
	for _, wsClient := range p.WsServerClients {
		if err := wsClient.Close(); err != nil {
			log.Printf("Error closing WebSocket server client: %v\n", err)
		}
	}

	// 使用一个5秒的超时优雅地关闭Gin服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
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
		log.Println(data)
		return nil
	}
}

// ChannelEventHandler 处理子频道事件
func ChannelEventHandler() event.ChannelEventHandler {
	return func(event *dto.WSPayload, data *dto.WSChannelData) error {
		log.Println(data)
		return nil
	}
}

// MemberEventHandler 处理成员变更事件
func MemberEventHandler() event.GuildMemberEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGuildMemberData) error {
		log.Println(data)
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
		log.Println("收到私域信息", data)
		return p.ProcessGuildNormalMessage(data)
	}
}

// InteractionHandler 处理内联交互事件
func InteractionHandler() event.InteractionEventHandler {
	return func(event *dto.WSPayload, data *dto.WSInteractionData) error {
		log.Println(data)
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

// GroupAddRobotEventHandler 实现处理 群机器人新增 事件的回调
func GroupAddRobotEventHandler() event.GroupAddRobotEventHandler {
	return func(event *dto.WSPayload, data *dto.GroupAddBotEvent) error {
		return p.ProcessGroupAddBot(data)
	}
}

// GroupDelRobotEventHandler 实现处理 群机器人删除 事件的回调
func GroupDelRobotEventHandler() event.GroupDelRobotEventHandler {
	return func(event *dto.WSPayload, data *dto.GroupAddBotEvent) error {
		return p.ProcessGroupDelBot(data)
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
	case "GroupAddRobotEventHandler": //群私聊
		return GroupAddRobotEventHandler(), true
	case "GroupDelRobotEventHandler": //群私聊
		return GroupDelRobotEventHandler(), true
	default:
		log.Printf("Unknown handler: %s\n", handlerName)
		return nil, false
	}
}

// allEmpty checks if all the strings in the slice are empty.
func allEmpty(addresses []string) bool {
	for _, addr := range addresses {
		if addr != "" {
			return false
		}
	}
	return true
}
