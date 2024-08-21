// 处理收到的信息事件
package Processor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/structs"
	"github.com/hoshinonyaruko/gensokyo/wsclient"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/keyboard"
	"github.com/tencent-connect/botgo/openapi"
)

// Processor 结构体用于处理消息
type Processors struct {
	Api             openapi.OpenAPI                   // API 类型
	Apiv2           openapi.OpenAPI                   //群的API
	Settings        *structs.Settings                 // 使用指针
	Wsclient        []*wsclient.WebSocketClient       // 指针的切片
	WsServerClients []callapi.WebSocketServerClienter //ws server被连接的客户端
}

type Sender struct {
	Nickname string `json:"nickname"`
	TinyID   string `json:"tiny_id"`
	UserID   int64  `json:"user_id"`
	Role     string `json:"role,omitempty"`
	Card     string `json:"card,omitempty"`
	Sex      string `json:"sex,omitempty"`
	Age      int32  `json:"age,omitempty"`
	Area     string `json:"area,omitempty"`
	Level    string `json:"level,omitempty"`
	Title    string `json:"title,omitempty"`
}

// 频道信息事件
type OnebotChannelMessage struct {
	ChannelID       string      `json:"channel_id"`
	GuildID         string      `json:"guild_id"`
	Message         interface{} `json:"message"`
	MessageID       string      `json:"message_id"`
	MessageType     string      `json:"message_type"`
	PostType        string      `json:"post_type"`
	SelfID          int64       `json:"self_id"`
	SelfTinyID      string      `json:"self_tiny_id"`
	Sender          Sender      `json:"sender"`
	SubType         string      `json:"sub_type"`
	Time            int64       `json:"time"`
	Avatar          string      `json:"avatar,omitempty"`
	UserID          int64       `json:"user_id"`
	RawMessage      string      `json:"raw_message"`
	Echo            string      `json:"echo,omitempty"`
	RealMessageType string      `json:"real_message_type,omitempty"` //当前信息的真实类型 表情表态
}

// 群信息事件
type OnebotGroupMessage struct {
	RawMessage      string      `json:"raw_message"`
	MessageID       int         `json:"message_id"`
	GroupID         int64       `json:"group_id"` // Can be either string or int depending on p.Settings.CompleteFields
	MessageType     string      `json:"message_type"`
	PostType        string      `json:"post_type"`
	SelfID          int64       `json:"self_id"` // Can be either string or int
	Sender          Sender      `json:"sender"`
	SubType         string      `json:"sub_type"`
	Time            int64       `json:"time"`
	Avatar          string      `json:"avatar,omitempty"`
	Echo            string      `json:"echo,omitempty"`
	Message         interface{} `json:"message"` // For array format
	MessageSeq      int         `json:"message_seq"`
	Font            int         `json:"font"`
	UserID          int64       `json:"user_id"`
	RealMessageType string      `json:"real_message_type,omitempty"`  //当前信息的真实类型 group group_private guild guild_private
	RealUserID      string      `json:"real_user_id,omitempty"`       //当前真实uid
	RealGroupID     string      `json:"real_group_id,omitempty"`      //当前真实gid
	IsBindedGroupId bool        `json:"is_binded_group_id,omitempty"` //当前群号是否是binded后的
	IsBindedUserId  bool        `json:"is_binded_user_id,omitempty"`  //当前用户号号是否是binded后的
}

type OnebotGroupMessageS struct {
	RawMessage      string      `json:"raw_message"`
	MessageID       string      `json:"message_id"`
	GroupID         string      `json:"group_id"` // Can be either string or int depending on p.Settings.CompleteFields
	MessageType     string      `json:"message_type"`
	PostType        string      `json:"post_type"`
	SelfID          int64       `json:"self_id"` // Can be either string or int
	Sender          Sender      `json:"sender"`
	SubType         string      `json:"sub_type"`
	Time            int64       `json:"time"`
	Avatar          string      `json:"avatar,omitempty"`
	Echo            string      `json:"echo,omitempty"`
	Message         interface{} `json:"message"` // For array format
	MessageSeq      int         `json:"message_seq"`
	Font            int         `json:"font"`
	UserID          string      `json:"user_id"`
	RealMessageType string      `json:"real_message_type,omitempty"`  //当前信息的真实类型 group group_private guild guild_private
	RealUserID      string      `json:"real_user_id,omitempty"`       //当前真实uid
	RealGroupID     string      `json:"real_group_id,omitempty"`      //当前真实gid
	IsBindedGroupId bool        `json:"is_binded_group_id,omitempty"` //当前群号是否是binded后的
	IsBindedUserId  bool        `json:"is_binded_user_id,omitempty"`  //当前用户号号是否是binded后的
}

// 私聊信息事件
type OnebotPrivateMessage struct {
	RawMessage      string        `json:"raw_message"`
	MessageID       int           `json:"message_id"` // Can be either string or int depending on logic
	MessageType     string        `json:"message_type"`
	PostType        string        `json:"post_type"`
	SelfID          int64         `json:"self_id"` // Can be either string or int depending on logic
	Sender          PrivateSender `json:"sender"`
	SubType         string        `json:"sub_type"`
	Time            int64         `json:"time"`
	Avatar          string        `json:"avatar,omitempty"`
	Echo            string        `json:"echo,omitempty"`
	Message         interface{}   `json:"message"`                     // For array format
	MessageSeq      int           `json:"message_seq"`                 // Optional field
	Font            int           `json:"font"`                        // Optional field
	UserID          int64         `json:"user_id"`                     // Can be either string or int depending on logic
	RealMessageType string        `json:"real_message_type,omitempty"` //当前信息的真实类型 group group_private guild guild_private
	RealUserID      string        `json:"real_user_id,omitempty"`      //当前真实uid
	IsBindedUserId  bool          `json:"is_binded_user_id,omitempty"` //当前用户号号是否是binded后的
}

// onebotv11标准扩展
type OnebotInteractionNotice struct {
	GroupID     int64                  `json:"group_id,omitempty"`
	NoticeType  string                 `json:"notice_type,omitempty"`
	PostType    string                 `json:"post_type,omitempty"`
	SelfID      int64                  `json:"self_id,omitempty"`
	SubType     string                 `json:"sub_type,omitempty"`
	Time        int64                  `json:"time,omitempty"`
	UserID      int64                  `json:"user_id,omitempty"`
	Data        *dto.WSInteractionData `json:"data,omitempty"`
	RealUserID  string                 `json:"real_user_id,omitempty"`  //当前真实uid
	RealGroupID string                 `json:"real_group_id,omitempty"` //当前真实gid
}

// onebotv11标准扩展
type OnebotGroupRejectNotice struct {
	GroupID    int64                    `json:"group_id,omitempty"`
	NoticeType string                   `json:"notice_type,omitempty"`
	PostType   string                   `json:"post_type,omitempty"`
	SelfID     int64                    `json:"self_id,omitempty"`
	SubType    string                   `json:"sub_type,omitempty"`
	Time       int64                    `json:"time,omitempty"`
	UserID     int64                    `json:"user_id,omitempty"`
	Data       *dto.GroupMsgRejectEvent `json:"data,omitempty"`
}

// onebotv11标准扩展
type OnebotGroupReceiveNotice struct {
	GroupID    int64                     `json:"group_id,omitempty"`
	NoticeType string                    `json:"notice_type,omitempty"`
	PostType   string                    `json:"post_type,omitempty"`
	SelfID     int64                     `json:"self_id,omitempty"`
	SubType    string                    `json:"sub_type,omitempty"`
	Time       int64                     `json:"time,omitempty"`
	UserID     int64                     `json:"user_id,omitempty"`
	Data       *dto.GroupMsgReceiveEvent `json:"data,omitempty"`
}

type PrivateSender struct {
	Nickname string `json:"nickname"`
	UserID   int64  `json:"user_id"` // Can be either string or int depending on logic
}

// 打印结构体的函数
func PrintStructWithFieldNames(v interface{}) {
	val := reflect.ValueOf(v)

	// 如果是指针，获取其指向的元素
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	// 确保我们传入的是一个结构体
	if typ.Kind() != reflect.Struct {
		mylog.Println("Input is not a struct")
		return
	}

	// 迭代所有的字段并打印字段名和值
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)
		mylog.Printf("%s: %v\n", field.Name, value.Interface())
	}
}

// 将结构体转换为 map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	json.Unmarshal(j, &out)
	return out
}

// 修改函数的返回类型为 *Processor
func NewProcessor(api openapi.OpenAPI, apiv2 openapi.OpenAPI, settings *structs.Settings, wsclient []*wsclient.WebSocketClient) *Processors {
	return &Processors{
		Api:      api,
		Apiv2:    apiv2,
		Settings: settings,
		Wsclient: wsclient,
	}
}

// 修改函数的返回类型为 *Processor
func NewProcessorV2(api openapi.OpenAPI, apiv2 openapi.OpenAPI, settings *structs.Settings) *Processors {
	return &Processors{
		Api:      api,
		Apiv2:    apiv2,
		Settings: settings,
	}
}

// 发信息给所有连接正向ws的客户端
func (p *Processors) SendMessageToAllClients(message map[string]interface{}) error {
	var result *multierror.Error

	for _, client := range p.WsServerClients {
		// 使用接口的方法
		err := client.SendMessage(message)
		if err != nil {
			// Append the error to our result
			result = multierror.Append(result, fmt.Errorf("failed to send to client: %w", err))
		}
	}

	// This will return nil if no errors were added
	return result.ErrorOrNil()
}

// 方便快捷的发信息函数
func (p *Processors) BroadcastMessageToAllFAF(message map[string]interface{}, api openapi.MessageAPI, data interface{}) error {
	// 并发发送到我们作为客户端的Wsclient
	for _, client := range p.Wsclient {
		go func(c callapi.WebSocketServerClienter) {
			_ = c.SendMessage(message) // 忽略错误
		}(client)
	}

	// 并发发送到我们作为服务器连接到我们的WsServerClients
	for _, serverClient := range p.WsServerClients {
		go func(sc callapi.WebSocketServerClienter) {
			_ = sc.SendMessage(message) // 忽略错误
		}(serverClient)
	}

	// 不再等待所有 goroutine 完成，直接返回
	return nil
}

// 方便快捷的发信息函数
func (p *Processors) BroadcastMessageToAll(message map[string]interface{}, api openapi.MessageAPI, data interface{}) error {
	var wg sync.WaitGroup
	errorCh := make(chan string, len(p.Wsclient)+len(p.WsServerClients))
	defer close(errorCh)

	// 并发发送到我们作为客户端的Wsclient
	for _, client := range p.Wsclient {
		wg.Add(1)
		go func(c callapi.WebSocketServerClienter) {
			defer wg.Done()
			if err := c.SendMessage(message); err != nil {
				errorCh <- fmt.Sprintf("error sending message via wsclient: %v", err)
			}
		}(client)
	}

	// 并发发送到我们作为服务器连接到我们的WsServerClients
	for _, serverClient := range p.WsServerClients {
		wg.Add(1)
		go func(sc callapi.WebSocketServerClienter) {
			defer wg.Done()
			if err := sc.SendMessage(message); err != nil {
				errorCh <- fmt.Sprintf("error sending message via WsServerClient: %v", err)
			}
		}(serverClient)
	}

	wg.Wait() // 等待所有goroutine完成

	var errors []string
	failed := 0
	for len(errorCh) > 0 {
		err := <-errorCh
		errors = append(errors, err)
		failed++
	}

	// 仅对连接正反ws的bot应用这个判断
	if !p.Settings.HttpOnlyBot {
		// 检查是否所有尝试都失败了
		if failed == len(p.Wsclient)+len(p.WsServerClients) {
			// 处理全部失败的情况
			fmt.Println("All ws event sending attempts failed.")
			downtimemessgae := config.GetDowntimeMessage()
			switch v := data.(type) {
			case *dto.WSGroupATMessageData:
				msgtocreate := &dto.MessageToCreate{
					Content: downtimemessgae,
					MsgID:   v.ID,
					MsgSeq:  1,
					MsgType: 0, // 默认文本类型
				}
				api.PostGroupMessage(context.Background(), v.GroupID, msgtocreate)
			case *dto.WSATMessageData:
				msgtocreate := &dto.MessageToCreate{
					Content: downtimemessgae,
					MsgID:   v.ID,
					MsgSeq:  1,
					MsgType: 0, // 默认文本类型
				}
				api.PostMessage(context.Background(), v.ChannelID, msgtocreate)
			case *dto.WSMessageData:
				msgtocreate := &dto.MessageToCreate{
					Content: downtimemessgae,
					MsgID:   v.ID,
					MsgSeq:  1,
					MsgType: 0, // 默认文本类型
				}
				api.PostMessage(context.Background(), v.ChannelID, msgtocreate)
			case *dto.WSDirectMessageData:
				msgtocreate := &dto.MessageToCreate{
					Content: downtimemessgae,
					MsgID:   v.ID,
					MsgSeq:  1,
					MsgType: 0, // 默认文本类型
				}
				api.PostMessage(context.Background(), v.GuildID, msgtocreate)
			case *dto.WSC2CMessageData:
				msgtocreate := &dto.MessageToCreate{
					Content: downtimemessgae,
					MsgID:   v.ID,
					MsgSeq:  1,
					MsgType: 0, // 默认文本类型
				}
				api.PostC2CMessage(context.Background(), v.Author.ID, msgtocreate)
			}
		}
	}

	// 判断是否填写了反向post地址
	if !allEmpty(config.GetPostUrl()) {
		go PostMessageToUrls(message)
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
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

// PostMessageToUrls 使用并发 goroutines 上报信息给多个反向 HTTP URL
func PostMessageToUrls(message map[string]interface{}) {
	// 获取上报 URL 列表
	postUrls := config.GetPostUrl()

	// 检查 postUrls 是否为空
	if len(postUrls) == 0 {
		return
	}

	// 转换 message 为 JSON 字符串
	jsonString, err := handlers.ConvertMapToJSONString(message)
	if err != nil {
		mylog.Printf("Error converting message to JSON: %v", err)
		return
	}

	// 使用 WaitGroup 等待所有 goroutines 完成
	var wg sync.WaitGroup
	for _, url := range postUrls {
		wg.Add(1)
		// 启动一个 goroutine
		go func(url string) {
			defer wg.Done() // 确保减少 WaitGroup 的计数器
			sendPostRequest(jsonString, url)
		}(url)
	}
	wg.Wait() // 等待所有 goroutine 完成
}

// sendPostRequest 发送单个 POST 请求
func sendPostRequest(jsonString, url string) {
	// 创建请求体
	reqBody := bytes.NewBufferString(jsonString)

	// 创建 POST 请求
	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		mylog.Printf("Error creating POST request to %s: %v", url, err)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	// 设置 X-Self-ID
	var selfid string
	if config.GetUseUin() {
		selfid = config.GetUinStr()
	} else {
		selfid = config.GetAppIDStr()
	}
	req.Header.Set("X-Self-ID", selfid)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		mylog.Printf("Error sending POST request to %s: %v", url, err)
		return
	}
	defer resp.Body.Close() // 确保释放网络资源

	// 可以在此处添加更多的响应处理逻辑
	mylog.Printf("Posted to %s successfully", url)
}

func (p *Processors) HandleFrameworkCommand(messageText string, data interface{}, Type string) error {
	// 正则表达式匹配转换后的 CQ 码
	cqRegex := regexp.MustCompile(`\[CQ:at,qq=\d+\]`)

	// 使用正则表达式替换所有的 CQ 码为 ""
	cleanedMessage := cqRegex.ReplaceAllString(messageText, "")

	// 去除字符串前后的空格
	cleanedMessage = strings.TrimSpace(cleanedMessage)
	if cleanedMessage == "t" {
		// 生成临时指令
		tempCmd := handleNoPermission()
		mylog.Printf("临时bind指令: %s 可忽略权限检查1次,或将masterid设置为空数组", tempCmd)
	}
	var err error
	var now, new, newpro1, newpro2 string
	var nowgroup, newgroup string
	var realid, realid2 string
	var guildid, guilduserid string
	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		realid = v.Author.ID
	case *dto.WSATMessageData:
		realid = v.Author.ID
		guildid = v.GuildID
		guilduserid = v.Author.ID
	case *dto.WSMessageData:
		realid = v.Author.ID
		guildid = v.GuildID
		guilduserid = v.Author.ID
	case *dto.WSDirectMessageData:
		realid = v.Author.ID
	case *dto.WSC2CMessageData:
		realid = v.Author.ID
	}

	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		realid2 = v.GroupID
	case *dto.WSATMessageData:
		realid2 = v.ChannelID
	case *dto.WSMessageData:
		realid2 = v.ChannelID
	case *dto.WSDirectMessageData:
		realid2 = v.ChannelID
	case *dto.WSC2CMessageData:
		realid2 = "group_private"
	}

	// 获取MasterID数组
	masterIDs := config.GetMasterID()

	// idmaps-pro获取群和用户id
	if config.GetIdmapPro() {
		newpro1, newpro2, err = idmap.RetrieveVirtualValuev2Pro(realid2, realid)
		if err != nil {
			mylog.Printf("idmaps-pro获取群和用户id 错误:%v", err)
		}
	} else {
		// 根据realid获取new(用户id)
		now, new, err = idmap.RetrieveVirtualValuev2(realid)
		if err != nil {
			mylog.Printf("根据realid获取new(用户id) 错误:%v", err)
		}
		// 根据realid获取new(群id)
		nowgroup, newgroup, err = idmap.RetrieveVirtualValuev2(realid2)
		if err != nil {
			mylog.Printf("根据realid获取new(群id)错误:%v", err)
		}
	}
	// 检查真实值或虚拟值是否在数组中
	var realValueIncluded, virtualValueIncluded bool

	// 如果 masterIDs 数组为空，则这两个值恒为 true
	if len(masterIDs) == 0 {
		realValueIncluded = true
		virtualValueIncluded = true
	} else {
		// 否则，检查真实值或虚拟值是否在数组中
		realValueIncluded = contains(masterIDs, realid)
		virtualValueIncluded = contains(masterIDs, new)
	}

	//unlock指令
	if Type == "guild" && strings.HasPrefix(cleanedMessage, config.GetUnlockPrefix()) {
		dm := &dto.DirectMessageToCreate{
			SourceGuildID: guildid,
			RecipientID:   guilduserid,
		}
		cdm, err := p.Api.CreateDirectMessage(context.TODO(), dm)
		if err != nil {
			mylog.Printf("unlock指令创建dm失败:%v", err)
		}
		msg := &dto.MessageToCreate{
			Content: "欢迎使用Gensokyo框架部署QQ机器人",
			MsgType: 0,
			MsgID:   "",
		}
		_, err = p.Api.PostDirectMessage(context.TODO(), cdm, msg)
		if err != nil {
			mylog.Printf("unlock指令发送失败:%v", err)
		}
	}

	// me指令处理逻辑
	if strings.HasPrefix(cleanedMessage, config.GetMePrefix()) {
		if err != nil {
			// 发送错误信息
			SendMessage(err.Error(), data, Type, p.Api, p.Apiv2)
			return err
		}
		// 发送成功信息
		if config.GetIdmapPro() {
			// 构造清晰的对应关系信息
			userMapping := fmt.Sprintf("当前真实值（用户）/当前虚拟值（用户） = [%s/%s]", realid, newpro2)
			groupMapping := fmt.Sprintf("当前真实值（群/频道）/当前虚拟值（群/频道） = [%s/%s]", realid2, newpro1)

			// 构造 bind 指令的使用说明
			bindInstruction := fmt.Sprintf("bind 指令: %s 当前虚拟值(用户) 目标虚拟值(用户) [当前虚拟值(群/频道) 目标虚拟值(群/频道)]", config.GetBindPrefix())

			// 发送整合后的消息
			message := fmt.Sprintf("idmaps-pro状态:\n%s\n%s\n%s", userMapping, groupMapping, bindInstruction)
			SendMessage(message, data, Type, p.Api, p.Apiv2)
		} else {
			SendMessage("目前状态:\n当前真实值(用户) "+now+"\n当前虚拟值(用户) "+new+"\n当前真实值(群/频道) "+nowgroup+"\n当前虚拟值(群/频道) "+newgroup+"\nbind指令:"+config.GetBindPrefix()+" 当前虚拟值"+" 目标虚拟值", data, Type, p.Api, p.Apiv2)
		}
		return nil
	}

	fields := strings.Fields(cleanedMessage)

	// 首先确保消息不是空的，然后检查是否是有效的临时指令
	if len(fields) > 0 && isValidTemporaryCommand(fields[0]) {
		// 执行 bind 操作
		if config.GetIdmapPro() {
			err := performBindOperationV2(cleanedMessage, data, Type, p.Api, p.Apiv2, newpro1)
			if err != nil {
				mylog.Printf("bind遇到错误:%v", err)
			}
		} else {
			err := performBindOperation(cleanedMessage, data, Type, p.Api, p.Apiv2)
			if err != nil {
				mylog.Printf("bind遇到错误:%v", err)
			}
		}
		return nil
	}

	// 如果不是临时指令，检查是否具有执行bind操作的权限并且消息以特定前缀开始
	if (realValueIncluded || virtualValueIncluded) && strings.HasPrefix(cleanedMessage, config.GetBindPrefix()) {
		// 执行 bind 操作
		if config.GetIdmapPro() {
			err := performBindOperationV2(cleanedMessage, data, Type, p.Api, p.Apiv2, newpro1)
			if err != nil {
				mylog.Printf("bind遇到错误:%v", err)
			}
		} else {
			err := performBindOperation(cleanedMessage, data, Type, p.Api, p.Apiv2)
			if err != nil {
				mylog.Printf("bind遇到错误:%v", err)
			}
		}
		return nil
	} else if strings.HasPrefix(cleanedMessage, config.GetBindPrefix()) {
		// 生成临时指令
		tempCmd := handleNoPermission()
		mylog.Printf("您没有权限,使用临时指令：%s 忽略权限检查,或将masterid设置为空数组", tempCmd)
		SendMessage("您没有权限,请配置config.yml或查看日志,使用临时指令", data, Type, p.Api, p.Apiv2)
	}

	//link指令
	if strings.HasPrefix(cleanedMessage, config.GetLinkPrefix()) {
		md, kb := generateMdByConfig()
		SendMessageMd(md, kb, data, Type, p.Api, p.Apiv2)
	}

	return nil
}

// 生成由两个英文字母构成的唯一临时指令
func generateTemporaryCommand() (string, error) {
	bytes := make([]byte, 1) // 生成1字节的随机数，足以表示2个十六进制字符
	if _, err := rand.Read(bytes); err != nil {
		return "", err // 处理随机数生成错误
	}
	command := hex.EncodeToString(bytes)[:2] // 将1字节转换为2个十六进制字符
	return command, nil
}

// 生成并添加一个新的临时指令
func handleNoPermission() string {
	idmap.MutexT.Lock()
	defer idmap.MutexT.Unlock()

	cmd, _ := generateTemporaryCommand()
	idmap.TemporaryCommands = append(idmap.TemporaryCommands, cmd)
	return cmd
}

// 检查指令是否是有效的临时指令
func isValidTemporaryCommand(cmd string) bool {
	idmap.MutexT.Lock()
	defer idmap.MutexT.Unlock()

	for i, tempCmd := range idmap.TemporaryCommands {
		if tempCmd == cmd {
			// 删除已验证的临时指令
			idmap.TemporaryCommands = append(idmap.TemporaryCommands[:i], idmap.TemporaryCommands[i+1:]...)
			return true
		}
	}
	return false
}

// 执行 bind 操作的逻辑
func performBindOperation(cleanedMessage string, data interface{}, Type string, p openapi.OpenAPI, p2 openapi.OpenAPI) error {
	// 分割指令以获取参数
	parts := strings.Fields(cleanedMessage)
	if len(parts) != 3 {
		mylog.Printf("bind指令参数错误\n正确的格式" + config.GetBindPrefix() + " 当前虚拟值 新虚拟值")
		return nil
	}

	// 将字符串转换为 int64
	oldRowValue, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}

	newRowValue, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return err
	}

	// 调用 UpdateVirtualValue
	err = idmap.UpdateVirtualValuev2(oldRowValue, newRowValue)
	if err != nil {
		SendMessage(err.Error(), data, Type, p, p2)
		return err
	}
	now, new, err := idmap.RetrieveRealValuev2(newRowValue)
	if err != nil {
		SendMessage(err.Error(), data, Type, p, p2)
	} else {
		SendMessage("绑定成功,目前状态:\n当前真实值 "+new+"\n当前虚拟值 "+now, data, Type, p, p2)
	}

	return nil
}

func performBindOperationV2(cleanedMessage string, data interface{}, Type string, p openapi.OpenAPI, p2 openapi.OpenAPI, GroupVir string) error {
	// 分割指令以获取参数
	parts := strings.Fields(cleanedMessage)

	// 检查参数数量
	if len(parts) < 3 || len(parts) > 5 {
		mylog.Printf("bind指令参数错误\n正确的格式: " + config.GetBindPrefix() + " 当前虚拟值(用户) 新虚拟值(用户) [当前虚拟值(群) 新虚拟值(群)]")
		return nil
	}

	// 当前虚拟值 用户
	oldVirtualValue1, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}
	//新的虚拟值 用户
	newVirtualValue1, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return err
	}

	// 设置默认值
	var oldRowValue, newRowValue int64

	// 如果提供了第3个和第4个参数，则解析它们
	if len(parts) > 3 {
		oldRowValue, err = parseOrDefault(parts[3], GroupVir)
		if err != nil {
			return err
		}

		newRowValue, err = parseOrDefault(parts[4], GroupVir)
		if err != nil {
			return err
		}
	} else {
		// 如果没有提供这些参数，则直接使用 GroupVir
		oldRowValue, err = strconv.ParseInt(GroupVir, 10, 64)
		if err != nil {
			return err
		}
		newRowValue = oldRowValue // 使用相同的值
	}
	// 调用 UpdateVirtualValue(兼顾老转换)
	err = idmap.UpdateVirtualValuev2(oldVirtualValue1, newVirtualValue1)
	if err != nil {
		SendMessage(err.Error(), data, Type, p, p2)
		return err
	}
	// 调用 UpdateVirtualValuev2Pro
	err = idmap.UpdateVirtualValuev2Pro(oldRowValue, newRowValue, oldVirtualValue1, newVirtualValue1)
	if err != nil {
		SendMessage(err.Error(), data, Type, p, p2)
		return err
	}

	now, new, err := idmap.RetrieveRealValuesv2Pro(newRowValue, newVirtualValue1)
	if err != nil {
		SendMessage(err.Error(), data, Type, p, p2)
	} else {
		newVirtualValue1Str := strconv.FormatInt(newRowValue, 10)
		newVirtualValue2Str := strconv.FormatInt(newVirtualValue1, 10)
		SendMessage("绑定成功,目前状态:\n当前真实值(群)"+now+"\n当前真实值(用户)"+new+"\n当前虚拟值(群)"+newVirtualValue1Str+"当前虚拟值(用户)"+newVirtualValue2Str, data, Type, p, p2)
	}

	return nil
}

// parseOrDefault 将字符串转换为int64，如果无法转换或为0，则使用默认值
func parseOrDefault(s string, defaultValue string) (int64, error) {
	value, err := strconv.ParseInt(s, 10, 64)
	if err == nil && value != 0 {
		return value, nil
	}

	return strconv.ParseInt(defaultValue, 10, 64)
}

// contains 检查数组中是否包含指定的字符串
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// SendMessage 发送消息根据不同的类型
func SendMessage(messageText string, data interface{}, messageType string, api openapi.OpenAPI, apiv2 openapi.OpenAPI) error {
	// 强制类型转换，获取Message结构
	var msg *dto.Message
	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSDirectMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSC2CMessageData:
		msg = (*dto.Message)(v)
	default:
		return nil
	}
	switch messageType {
	case "guild":
		// 处理公会消息
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
		if _, err := api.PostMessage(context.TODO(), msg.ChannelID, textMsg); err != nil {
			mylog.Printf("发送文本信息失败: %v", err)
			return err
		}

	case "group":
		// 处理群组消息
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
		_, err := apiv2.PostGroupMessage(context.TODO(), msg.GroupID, textMsg)
		if err != nil {
			mylog.Printf("发送文本群组信息失败: %v", err)
			return err
		}

	case "guild_private":
		// 处理私信
		timestamp := time.Now().Unix()
		timestampStr := fmt.Sprintf("%d", timestamp)
		dm := &dto.DirectMessage{
			GuildID:    msg.GuildID,
			ChannelID:  msg.ChannelID,
			CreateTime: timestampStr,
		}
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
		if _, err := apiv2.PostDirectMessage(context.TODO(), dm, textMsg); err != nil {
			mylog.Printf("发送文本信息失败: %v", err)
			return err
		}

	case "group_private":
		// 处理群组私聊消息
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
		_, err := apiv2.PostC2CMessage(context.TODO(), msg.Author.ID, textMsg)
		if err != nil {
			mylog.Printf("发送文本私聊信息失败: %v", err)
			return err
		}

	default:
		return errors.New("未知的消息类型")
	}

	return nil
}

// SendMessageMd  发送Md消息根据不同的类型
func SendMessageMd(md *dto.Markdown, kb *keyboard.MessageKeyboard, data interface{}, messageType string, api openapi.OpenAPI, apiv2 openapi.OpenAPI) error {
	// 强制类型转换，获取Message结构
	var msg *dto.Message
	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSATMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSDirectMessageData:
		msg = (*dto.Message)(v)
	case *dto.WSC2CMessageData:
		msg = (*dto.Message)(v)
	default:
		return nil
	}
	switch messageType {
	case "guild":
		// 处理公会消息
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		Message := &dto.MessageToCreate{
			MsgID:    msg.ID,
			MsgSeq:   msgseq,
			Markdown: md,
			Keyboard: kb,
			MsgType:  2, //md信息
		}
		Message.Timestamp = time.Now().Unix() // 设置时间戳
		if _, err := api.PostMessage(context.TODO(), msg.ChannelID, Message); err != nil {
			mylog.Printf("发送文本信息失败: %v", err)
			return err
		}

	case "group":
		// 处理群组消息
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		Message := &dto.MessageToCreate{
			Content:  "markdown",
			MsgID:    msg.ID,
			MsgSeq:   msgseq,
			Markdown: md,
			Keyboard: kb,
			MsgType:  2, //md信息
		}
		Message.Timestamp = time.Now().Unix() // 设置时间戳
		_, err := apiv2.PostGroupMessage(context.TODO(), msg.GroupID, Message)
		if err != nil {
			mylog.Printf("发送文本群组信息失败: %v", err)
			return err
		}

	case "guild_private":
		// 处理私信
		timestamp := time.Now().Unix()
		timestampStr := fmt.Sprintf("%d", timestamp)
		dm := &dto.DirectMessage{
			GuildID:    msg.GuildID,
			ChannelID:  msg.ChannelID,
			CreateTime: timestampStr,
		}
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		Message := &dto.MessageToCreate{
			MsgID:    msg.ID,
			MsgSeq:   msgseq,
			Markdown: md,
			Keyboard: kb,
			MsgType:  2, //md信息
		}
		Message.Timestamp = time.Now().Unix() // 设置时间戳
		if _, err := apiv2.PostDirectMessage(context.TODO(), dm, Message); err != nil {
			mylog.Printf("发送文本信息失败: %v", err)
			return err
		}

	case "group_private":
		// 处理群组私聊消息
		msgseq := echo.GetMappingSeq(msg.ID)
		echo.AddMappingSeq(msg.ID, msgseq+1)
		Message := &dto.MessageToCreate{
			Content:  "markdown",
			MsgID:    msg.ID,
			MsgSeq:   msgseq,
			Markdown: md,
			Keyboard: kb,
			MsgType:  2, //md信息
		}
		Message.Timestamp = time.Now().Unix() // 设置时间戳
		_, err := apiv2.PostC2CMessage(context.TODO(), msg.Author.ID, Message)
		if err != nil {
			mylog.Printf("发送文本私聊信息失败: %v", err)
			return err
		}

	default:
		return errors.New("未知的消息类型")
	}

	return nil
}

// SendMessageMdAddBot  发送Md消息在AddBot事件
func SendMessageMdAddBot(md *dto.Markdown, kb *keyboard.MessageKeyboard, data *dto.GroupAddBotEvent, api openapi.OpenAPI, apiv2 openapi.OpenAPI) error {

	// 处理群组消息
	msgseq := echo.GetMappingSeq(data.EventID)
	echo.AddMappingSeq(data.ID, msgseq+1)
	Message := &dto.MessageToCreate{
		Content:  "markdown",
		EventID:  data.EventID,
		MsgSeq:   msgseq,
		Markdown: md,
		Keyboard: kb,
		MsgType:  2, //md信息
	}

	Message.Timestamp = time.Now().Unix() // 设置时间戳
	_, err := apiv2.PostGroupMessage(context.TODO(), data.GroupOpenID, Message)
	if err != nil {
		mylog.Printf("发送文本群组信息失败: %v", err)
		return err
	}

	return nil
}

// autobind 函数接受 interface{} 类型的数据
// commit by 紫夜 2023-11-19
func (p *Processors) Autobind(data interface{}) error {
	var realID string
	var groupID string
	var attachmentURL string

	// 群at
	switch v := data.(type) {
	case *dto.WSGroupATMessageData:
		realID = v.Author.ID
		groupID = v.GroupID
		attachmentURL = v.Attachments[0].URL
		//群私聊
	case *dto.WSC2CMessageData:
		realID = v.Author.ID
		groupID = v.GroupID
		attachmentURL = v.Attachments[0].URL
	default:
		return fmt.Errorf("未知的数据类型")
	}

	// 从 URL 中提取 newRowValue (vuin)
	vuinRegex := regexp.MustCompile(`vuin=(\d+)`)
	vuinMatches := vuinRegex.FindStringSubmatch(attachmentURL)
	if len(vuinMatches) < 2 {
		mylog.Errorf("无法从 URL 中提取 vuin")
		return nil
	}
	vuinstr := vuinMatches[1]
	vuinValue, err := strconv.ParseInt(vuinMatches[1], 10, 64)
	if err != nil {
		return err
	}
	// 从 URL 中提取第二个 newRowValue (群号)
	idRegex := regexp.MustCompile(`gchatpic_new/(\d+)/`)
	idMatches := idRegex.FindStringSubmatch(attachmentURL)
	if len(idMatches) < 2 {
		mylog.Errorf("无法从 URL 中提取 ID")
		return nil
	}
	idValuestr := idMatches[1]
	idValue, err := strconv.ParseInt(idMatches[1], 10, 64)
	if err != nil {
		return err
	}
	var GroupID64, userid64 int64
	//获取虚拟值
	// 映射str的GroupID到int
	GroupID64, err = idmap.StoreIDv2(groupID)
	if err != nil {
		mylog.Errorf("failed to convert ChannelID to int: %v", err)
		return nil
	}
	// 映射str的userid到int
	userid64, err = idmap.StoreIDv2(realID)
	if err != nil {
		mylog.Printf("Error storing ID: %v", err)
		return nil
	}
	//覆盖赋值
	if config.GetIdmapPro() {
		//转换idmap-pro 虚拟值
		//将真实id转为int userid64
		GroupID64, userid64, err = idmap.StoreIDv2Pro(groupID, realID)
		if err != nil {
			mylog.Errorf("Error storing ID689: %v", err)
		}
	}
	// 单独检查vuin和gid的绑定状态
	vuinBound := strconv.FormatInt(userid64, 10) == vuinstr
	gidBound := strconv.FormatInt(GroupID64, 10) == idValuestr
	// 根据不同情况进行处理
	if !vuinBound && !gidBound {
		// 两者都未绑定，更新两个映射
		if err := updateMappings(userid64, vuinValue, GroupID64, idValue); err != nil {
			mylog.Printf("Error updateMappings for both: %v", err)
			//return err
		}
		// idmaps pro也更新
		err = idmap.UpdateVirtualValuev2Pro(GroupID64, idValue, userid64, vuinValue)
		if err != nil {
			mylog.Errorf("Error storing ID703: %v", err)
		}
	} else if !vuinBound {
		// 只有vuin未绑定，更新vuin映射
		if err := idmap.UpdateVirtualValuev2(userid64, vuinValue); err != nil {
			mylog.Printf("Error UpdateVirtualValuev2 for vuin: %v", err)
			//return err
		}
		// idmaps pro也更新,但只更新vuin
		idmap.UpdateVirtualValuev2Pro(GroupID64, idValue, userid64, vuinValue)
	} else if !gidBound {
		// 只有gid未绑定，更新gid映射
		if err := idmap.UpdateVirtualValuev2(GroupID64, idValue); err != nil {
			mylog.Printf("Error UpdateVirtualValuev2 for gid: %v", err)
			//return err
		}
		// idmaps pro也更新,但只更新gid
		idmap.UpdateVirtualValuev2Pro(GroupID64, idValue, userid64, vuinValue)
	} else {
		// 两者都已绑定，不执行任何操作
		mylog.Errorf("Both vuin and gid are already binded")
	}

	return nil
}

// 更新映射的辅助函数
func updateMappings(userid64, vuinValue, GroupID64, idValue int64) error {
	if err := idmap.UpdateVirtualValuev2(userid64, vuinValue); err != nil {
		mylog.Printf("Error UpdateVirtualValuev2 for vuin: %v", err)
		return err
	}
	if err := idmap.UpdateVirtualValuev2(GroupID64, idValue); err != nil {
		mylog.Printf("Error UpdateVirtualValuev2 for gid: %v", err)
		return err
	}
	return nil
}

// GenerateAvatarURL 生成根据给定 userID 和随机 q 值组合的 QQ 头像 URL
func GenerateAvatarURL(userID int64) (string, error) {
	// 使用 crypto/rand 生成更安全的随机数
	n, err := rand.Int(rand.Reader, big.NewInt(5))
	if err != nil {
		return "", err
	}
	qNumber := n.Int64() + 1 // 产生 1 到 5 的随机数

	// 构建并返回 URL
	return fmt.Sprintf("http://q%d.qlogo.cn/g?b=qq&nk=%d&s=640", qNumber, userID), nil
}

// GenerateAvatarURLV2 生成根据32位ID 和 Appid 组合的 新QQ 头像 URL
func GenerateAvatarURLV2(openid string) (string, error) {

	appidstr := config.GetAppIDStr()
	// 构建并返回 URL
	return fmt.Sprintf("https://q.qlogo.cn/qqapp/%s/%s/640", appidstr, openid), nil
}

// 生成link卡片
func generateMdByConfig() (md *dto.Markdown, kb *keyboard.MessageKeyboard) {
	//相关配置获取
	mdtext := config.GetLinkText()
	mdtext = "\r" + mdtext
	CustomTemplateID := config.GetCustomTemplateID()
	linkBots := config.GetLinkBots()
	imgURL := config.GetLinkPic()

	linknum := config.GetLinkNum()

	//超过n个时候随机显示
	if len(linkBots) > linknum {
		linkBots = getRandomSelection(linkBots, linknum)
	}

	var mdParams []*dto.MarkdownParams
	if !config.GetNativeMD() {
		//组合 mdParams
		if imgURL != "" {
			height, width, err := images.GetImageDimensions(imgURL)
			if err != nil {
				mylog.Printf("获取图片宽高出错")
			}
			imgDesc := fmt.Sprintf("图片 #%dpx #%dpx", width, height)
			// 创建 MarkdownParams 的实例
			mdParams = []*dto.MarkdownParams{
				{Key: "img_dec", Values: []string{imgDesc}},
				{Key: "img_url", Values: []string{imgURL}},
				{Key: "text_end", Values: []string{mdtext}},
			}
		} else {
			mdParams = []*dto.MarkdownParams{
				{Key: "text_end", Values: []string{mdtext}},
			}
		}

		// 组合模板 Markdown
		md = &dto.Markdown{
			CustomTemplateID: CustomTemplateID,
			Params:           mdParams,
		}
	} else {
		// 使用原生Markdown格式
		var content string
		if imgURL != "" {
			height, width, err := images.GetImageDimensions(imgURL)
			if err != nil {
				mylog.Printf("获取图片宽高出错")
			}
			imgDesc := fmt.Sprintf("图片 #%dpx #%dpx", width, height)
			content = fmt.Sprintf("![%s](%s)\n%s", imgDesc, imgURL, mdtext)
		} else {
			content = mdtext
		}

		// 原生 Markdown
		md = &dto.Markdown{
			Content: content,
		}
	}

	// 创建自定义键盘
	customKeyboard := &keyboard.CustomKeyboard{}
	var currentRow *keyboard.Row
	var buttonCount int

	for _, bot := range linkBots {
		parts := strings.SplitN(bot, "-", 3)
		if len(parts) != 3 && len(parts) != 2 {
			continue // 跳过无效的格式
		}
		var button *keyboard.Button
		if len(parts) == 3 {
			name := parts[2]
			botuin := parts[1]
			botappid := parts[0]
			boturl := handlers.BuildQQBotShareLink(botuin, botappid)

			button = &keyboard.Button{
				RenderData: &keyboard.RenderData{
					Label:        name,
					VisitedLabel: name,
					Style:        1, // 蓝色边缘
				},
				Action: &keyboard.Action{
					Type:          0,                             // 链接类型
					Permission:    &keyboard.Permission{Type: 2}, // 所有人可操作
					Data:          boturl,
					UnsupportTips: "请升级新版手机QQ",
				},
			}
		} else if len(parts) == 2 {
			boturl := parts[0]
			name := parts[1]

			button = &keyboard.Button{
				RenderData: &keyboard.RenderData{
					Label:        name,
					VisitedLabel: name,
					Style:        1, // 蓝色边缘
				},
				Action: &keyboard.Action{
					Type:          0,                             // 链接类型
					Permission:    &keyboard.Permission{Type: 2}, // 所有人可操作
					Data:          boturl,
					UnsupportTips: "请升级新版手机QQ",
				},
			}
		}

		lines := config.GetLinkLines()

		// 如果当前行为空或已满（lines个按钮），则创建一个新行
		if currentRow == nil || buttonCount == lines {
			currentRow = &keyboard.Row{}
			customKeyboard.Rows = append(customKeyboard.Rows, currentRow)
			buttonCount = 0
		}

		// 将按钮添加到当前行
		currentRow.Buttons = append(currentRow.Buttons, button)
		buttonCount++
	}

	// 创建 MessageKeyboard 并设置其 Content
	kb = &keyboard.MessageKeyboard{
		Content: customKeyboard,
	}

	return md, kb
}

// getRandomSelection 处理bots数组，分类选择随机bots
func getRandomSelection(bots []string, max int) []string {
	threePartBots := []string{}
	twoPartBots := []string{}

	// 分类
	for _, bot := range bots {
		parts := strings.SplitN(bot, "-", 3)
		if len(parts) == 3 {
			threePartBots = append(threePartBots, bot)
		} else if len(parts) == 2 {
			twoPartBots = append(twoPartBots, bot)
		}
	}

	// 对每个分类应用随机选择
	selectedThreePartBots := selectRandomItems(threePartBots, max)
	selectedTwoPartBots := selectRandomItems(twoPartBots, max)

	// 合并结果
	return append(selectedThreePartBots, selectedTwoPartBots...)
}

// selectRandomItems 从给定slice中随机选择最多max个元素
func selectRandomItems(slice []string, max int) []string {
	if len(slice) <= max {
		return slice
	}

	selected := make(map[int]bool)
	var result []string
	for len(result) < max {
		index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(slice))))
		idx := int(index.Int64())
		if !selected[idx] {
			selected[idx] = true
			result = append(result, slice[idx])
		}
	}
	return result
}
