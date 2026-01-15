package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

// WakeupResponseNotice 伪装成 Notice 的响应结构
type WakeupResponseNotice struct {
	PostType   string `json:"post_type"`
	NoticeType string `json:"notice_type"`

	UserID     int64  `json:"user_id"`                // 【修改】改成 int64，兼容 NoticeEvent
	RealUserID string `json:"real_user_id,omitempty"` // 【新增】用来放 32 位 OpenID String

	Status    string `json:"status"`
	MessageID string `json:"message_id,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	Time      int64  `json:"time"`
	SelfID    int64  `json:"self_id"`
}

func init() {
	callapi.RegisterHandler("send_private_msg_wakeup", HandleSendPrivateMsgWakeup)
}

// HandleSendPrivateMsgWakeup 处理私聊互动召回消息
// 逻辑高度复刻 HandleSendPrivateMsg，但适配 IsWakeup 参数并移除 ID 转换逻辑
func HandleSendPrivateMsgWakeup(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 1. 获取 UserID (直接断言，无需转换)
	userID, ok := message.Params.UserID.(string)
	if !ok || len(userID) != 32 {
		mylog.Printf("send_private_msg_wakeup 错误: UserID 必须是 32 位字符串")
		return "", nil
	}

	// 此时 selfID 最好从配置或 message 中获取，这里演示用 0 或 message.SelfID (如果你的ActionMessage里有)
	var selfID int64 = int64(config.GetAppID())

	// 2. 解析消息内容
	messageText, foundItems := parseMessageContent(message.Params, message, client, api, apiv2)

	mylog.Printf("发送互动召回消息 UserID:[%s]", userID)

	// 定义 KeyMap (复刻原逻辑)
	keyMap := map[string]bool{
		"markdown":      true,
		"qqmusic":       true,
		"local_image":   true,
		"local_record":  true,
		"url_image":     true,
		"url_images":    true, // 注意：原代码中有 url_image 和 url_images
		"base64_record": true,
		"base64_image":  true,
	}

	var singleItem = make(map[string][]string)
	var imageType, imageUrl string
	imageCount := 0

	// 检查图片并计算数量 (复刻原逻辑)
	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) == 1 {
		imageType = "local_image"
		imageUrl = imageURLs[0]
		imageCount++
	} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) == 1 {
		imageType = "url_image"
		imageUrl = imageURLs[0]
		imageCount++
	} else if base64Images, ok := foundItems["base64_image"]; ok && len(base64Images) == 1 {
		imageType = "base64_image"
		imageUrl = base64Images[0]
		imageCount++
	}

	// --- 场景 A: 单图文混合消息 ---
	if imageCount == 1 && messageText != "" {
		mylog.Printf("发送召回图文混合信息")
		singleItem[imageType] = []string{imageUrl}

		// 生成消息对象 (MsgID/EventID 传空，因为是主动召回)
		groupReply := generatePrivateMessage("", "", singleItem, "", 0, apiv2, userID)

		// 类型断言
		richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
		if !ok {
			mylog.Printf("Error: Expected RichMediaMessage type for key")
			return "", nil
		}

		// 上传图片 (不需要 IsWakeup，只是为了拿 FileInfo)
		fileInfo, err := uploadMediaPrivate(context.TODO(), userID, richMediaMessage, apiv2)
		if err != nil {
			mylog.Printf("上传图片失败: %v", err)
			sendWakeupNotice(client, userID, nil, err, selfID)
			return "", nil
		}

		// 构造 MessageToCreate
		groupMessage := &dto.MessageToCreate{
			Content: messageText,
			Media: dto.Media{
				FileInfo: fileInfo,
			},
			MsgType:  7,    // 富媒体类型
			IsWakeup: true, // [重点] 标记为召回
			MsgID:    "",   // [重点] 互斥
			EventID:  "",   // [重点] 互斥
		}

		// 发送
		resp, err := postC2CWakeupMessageWithRetry(apiv2, userID, groupMessage)
		sendWakeupNotice(client, userID, resp, err, selfID)

		delete(foundItems, imageType)
		messageText = ""
	}

	// --- 场景 B: 纯文本消息 ---
	if messageText != "" {
		// 复用 generatePrivateMessage 获取基础结构 (虽然可以直接构造，但为了保持风格一致)
		groupReply := generatePrivateMessage("", "", nil, messageText, 0, apiv2, userID)

		if groupMessage, ok := groupReply.(*dto.MessageToCreate); ok {
			// 强制覆盖为召回模式
			groupMessage.IsWakeup = true
			groupMessage.MsgID = ""
			groupMessage.EventID = ""

			resp, err := postC2CWakeupMessageWithRetry(apiv2, userID, groupMessage)
			sendWakeupNotice(client, userID, resp, err, selfID)
		} else {
			mylog.Println("Error: Expected MessageToCreate type for text.")
		}
	}

	// --- 场景 C: 遍历 foundItems (核心复刻部分) ---
	for key, urls := range foundItems {
		for _, url := range urls {
			var singleItem = make(map[string][]string)
			singleItem[key] = []string{url}

			// 生成消息对象
			groupReply := generatePrivateMessage("", "", singleItem, "", 0, apiv2, userID)

			// 1. 尝试断言为 RichMediaMessage (通常是需要上传的媒体)
			richMediaMessage, ok := groupReply.(*dto.RichMediaMessage)
			if !ok {
				// 如果不是 RichMediaMessage，检查是否在 KeyMap 中 (Markdown, QQMusic 等)
				if _, exists := keyMap[key]; exists {
					// 断言为 MessageToCreate
					groupMessage, ok := groupReply.(*dto.MessageToCreate)
					if !ok {
						mylog.Println("Error: Expected MessageToCreate type.")
						continue
					}

					// [关键修改] 适配召回消息格式
					groupMessage.IsWakeup = true
					groupMessage.MsgID = ""
					groupMessage.EventID = ""

					// 发送特殊类型的消息 (Markdown 等)
					resp, err := postC2CWakeupMessageWithRetry(apiv2, userID, groupMessage)

					// 错误处理逻辑复刻
					if err != nil {
						mylog.Printf("发送 MessageToCreate 召回信息失败: %v", err)
						if config.GetSaveError() {
							mylog.ErrLogToFile("type", "PostC2CWakeup-Special")
							mylog.ErrInterfaceToFile("request", groupMessage)
							mylog.ErrLogToFile("error", err.Error())
						}
					}

					// 发送通知
					sendWakeupNotice(client, userID, resp, err, selfID)
				} else {
					mylog.Printf("Error: Expected RichMediaMessage type for key %s.", key)
				}
				continue // 继续下一个
			}

			// 2. 如果是 RichMediaMessage，执行上传 + 发送流程
			// 上传媒体 (这里不需要 IsWakeup)
			messageReturn, err := apiv2.PostC2CMessage(context.TODO(), userID, richMediaMessage)
			if err != nil {
				mylog.Printf("发送 %s 信息失败_upload: %v", key, err)
				if config.GetSaveError() {
					mylog.ErrLogToFile("type", "PostC2CWakeup-Upload")
					mylog.ErrInterfaceToFile("request", richMediaMessage)
					mylog.ErrLogToFile("error", err.Error())
				}
			}

			// 富媒体上传超时重试 (复刻原逻辑)
			if err != nil && (strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "富媒体文件上传超时")) {
				// 注意：这里重试的是上传接口，不是发送接口
				// postC2CRichMediaMessageWithRetry 是你原有代码中的函数，可以直接复用
				// 如果那个函数没导出，可能需要复制一份改名，这里假设能调用或者逻辑一致
				messageReturn, err = postC2CRichMediaMessageWithRetry(apiv2, userID, richMediaMessage)
			}

			// 上传成功，构造最终的 MessageToCreate 进行发送
			if messageReturn != nil && messageReturn.MediaResponse != nil && messageReturn.MediaResponse.FileInfo != "" {
				media := dto.Media{
					FileInfo: messageReturn.MediaResponse.FileInfo,
				}
				groupMessage := &dto.MessageToCreate{
					Content:  " ", // 媒体消息通常带个空格
					MsgType:  7,   // 富媒体
					Media:    media,
					IsWakeup: true, // [重点]
					MsgID:    "",
					EventID:  "",
				}

				// 发送最终的召回消息
				resp, err := postC2CWakeupMessageWithRetry(apiv2, userID, groupMessage)
				if err != nil {
					mylog.Printf("发送 %s 召回私聊信息失败: %v", key, err)
				}

				// 发送通知
				sendWakeupNotice(client, userID, resp, err, selfID)
			} else {
				// 上传都失败了，发一个失败通知
				sendWakeupNotice(client, userID, nil, fmt.Errorf("media upload failed for %s: %v", key, err), selfID)
			}
		}
	}

	return `{"status": "ok", "retcode": 0}`, nil
}

// postC2CWakeupMessageWithRetry 召回消息专用重试逻辑
// 保持了 3 次重试和详细的错误落盘
func postC2CWakeupMessageWithRetry(apiv2 openapi.OpenAPI, userID string, msg *dto.MessageToCreate) (resp *dto.C2CMessageResponse, err error) {
	retryCount := 3
	for i := 0; i < retryCount; i++ {
		// 召回消息不需要 MsgSeq
		resp, err = apiv2.PostC2CMessage(context.TODO(), userID, msg)

		if err != nil && (strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "富媒体文件上传超时")) {
			mylog.Printf("召回消息超时重试第 %d 次: %v", i+1, err)
			if config.GetSaveError() {
				mylog.ErrLogToFile("type", "PostC2CWakeup-Retry-"+strconv.Itoa(i+1))
				mylog.ErrInterfaceToFile("request", msg)
				mylog.ErrLogToFile("error", err.Error())
			}
			time.Sleep(3 * time.Second)
			continue
		} else {
			// 成功 或 非超时错误
			mylog.Printf("召回消息请求结束: %v", err)
			if config.GetSaveError() {
				suffix := "-successed"
				if err != nil {
					suffix = "-failed"
				}
				mylog.ErrLogToFile("type", "PostC2CWakeup-Retry-"+strconv.Itoa(i+1)+suffix)
				mylog.ErrInterfaceToFile("request", msg)
				if resp != nil {
					mylog.ErrLogToFile("msg_id", resp.Message.ID)
				}
				if err != nil {
					mylog.ErrLogToFile("error", err.Error())
				}
			}
		}
		break
	}
	return resp, err
}

// sendWakeupNotice 将发送结果伪装成 Notice 发送给应用端
func sendWakeupNotice(client callapi.Client, userID string, resp *dto.C2CMessageResponse, err error, selfID int64) {

	// 1. 构造结构体
	notice := WakeupResponseNotice{
		PostType:   "notice",
		NoticeType: "c2c_wakeup_resp",

		UserID:     0,      // 【关键】设为 0，防止 json.Unmarshal 报错
		RealUserID: userID, // 【关键】真正的 String ID 放这里

		Time:   time.Now().Unix(),
		SelfID: selfID,
	}

	if err != nil {
		notice.Status = "failed"
		notice.ErrorMsg = err.Error()
	} else {
		notice.Status = "success"
		if resp != nil && resp.Message != nil {
			notice.MessageID = resp.Message.ID
		}
	}

	// 2. 转换为 Map
	outputMap := structToMap(notice)

	// 3. 推送给应用端
	if sendErr := client.SendMessage(outputMap); sendErr != nil {
		mylog.Printf("发送召回结果通知失败: %v", sendErr)
	}
}
