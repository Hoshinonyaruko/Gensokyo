package handlers

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/images"
	"github.com/hoshinonyaruko/gensokyo/mylog"

	"github.com/hoshinonyaruko/gensokyo/echo"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("send_guild_channel_msg", HandleSendGuildChannelMsg)
}

func HandleSendGuildChannelMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	var retmsg string
	var err error
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}
	// 检查GroupID是否为0
	checkZeroGroupID := func(id interface{}) bool {
		switch v := id.(type) {
		case int:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != "0" // 检查字符串形式的0
		default:
			return true // 如果不是int、int64或string，假定它不为0
		}
	}

	// 检查UserID是否为0
	checkZeroUserID := func(id interface{}) bool {
		switch v := id.(type) {
		case int:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != "0" // 同样检查字符串形式的0
		default:
			return true // 如果不是int、int64或string，假定它不为0
		}
	}

	if msgType == "" && message.Params.GroupID != nil && checkZeroGroupID(message.Params.GroupID) {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
	}
	if msgType == "" && message.Params.UserID != nil && checkZeroUserID(message.Params.UserID) {
		msgType = GetMessageTypeByUserid(config.GetAppIDStr(), message.Params.UserID)
	}
	if msgType == "" && message.Params.GroupID != nil && checkZeroGroupID(message.Params.GroupID) {
		msgType = GetMessageTypeByGroupidV2(message.Params.GroupID)
	}
	if msgType == "" && message.Params.UserID != nil && checkZeroUserID(message.Params.UserID) {
		msgType = GetMessageTypeByUseridV2(message.Params.UserID)
	}
	// New checks for UserID and GroupID being nil or 0
	if (message.Params.UserID == nil || !checkZeroUserID(message.Params.UserID)) &&
		(message.Params.GroupID == nil || !checkZeroGroupID(message.Params.GroupID)) {
		mylog.Printf("send_group_msgs接收到错误action: %v", message)
		return "", nil
	}
	//当不转换频道信息时(不支持频道私聊)
	if msgType == "" {
		msgType = "guild"
	}
	switch msgType {
	//原生guild信息
	case "guild":
		params := message.Params
		messageText, foundItems := parseMessageContent(params, message, client, api, apiv2)

		channelID := params.ChannelID
		// 使用 echo 获取消息ID
		var messageID string
		if config.GetLazyMessageId() {
			//由于实现了Params的自定义unmarshell 所以可以类型安全的断言为string
			messageID = echo.GetLazyMessagesId(channelID.(string))
			mylog.Printf("GetLazyMessagesId: %v", messageID)
		}
		if messageID == "" {
			if echoStr, ok := message.Echo.(string); ok {
				messageID = echo.GetMsgIDByKey(echoStr)
				mylog.Println("echo取频道发信息对应的message_id:", messageID)
			}
		}
		if messageID == "" {
			messageID = GetMessageIDByUseridOrGroupid(config.GetAppIDStr(), channelID)
			mylog.Println("通过GetMessageIDByUseridOrGroupid函数获取的message_id:", messageID)
		}
		//主动信息
		if messageID == "2000" {
			messageID = ""
			mylog.Println("通过lazymsgid发送频道主动信息,若非主动信息请提交issue")
		}
		//开发环境用
		if config.GetDevMsgID() {
			messageID = "1000"
		}
		mylog.Println("频道发信息messageText:", messageText)
		//mylog.Println("foundItems:", foundItems)
		var singleItem = make(map[string][]string)
		var imageType, imageUrl string
		imageCount := 0

		// 检查不同类型的图片并计算数量
		if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) == 1 {
			imageType = "local_image"
			imageUrl = imageURLs[0]
			imageCount++
		} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) == 1 {
			imageType = "url_image"
			imageUrl = imageURLs[0]
			imageCount++
		} else if imageURLs, ok := foundItems["url_images"]; ok && len(imageURLs) == 1 {
			imageType = "url_images"
			imageUrl = imageURLs[0]
			imageCount++
		} else if base64Images, ok := foundItems["base64_image"]; ok && len(base64Images) == 1 {
			imageType = "base64_image"
			imageUrl = base64Images[0]
			imageCount++
		}

		var resp *dto.Message
		if imageCount == 1 && messageText != "" {
			//我想优化一下这里,让它优雅一点
			mylog.Printf("发图文混合信息-频道")
			// 创建包含单个图片的 singleItem
			singleItem[imageType] = []string{imageUrl}
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			Reply, isbase64 := GenerateReplyMessage(messageID, singleItem, "", msgseq+1)
			if !isbase64 {
				// 创建包含文本和base64图像信息的消息
				msgseq = echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				newMessage := &dto.MessageToCreate{
					Content: messageText, // 添加文本内容
					Image:   Reply.Image,
					MsgID:   messageID,
					MsgSeq:  msgseq,
					MsgType: 0,
				}
				newMessage.Timestamp = time.Now().Unix() // 设置时间戳

				if _, err = api.PostMessage(context.TODO(), channelID.(string), newMessage); err != nil {
					mylog.Printf("发送图文混合信息失败: %v", err)
				}
				// 检查是否是 40003 错误
				if err != nil && strings.Contains(err.Error(), `"code":40003`) && len(newMessage.Image) > 0 {
					// 从 newMessage.Image 中提取图片地址
					imageURL := newMessage.Image

					// 调用下载并转换为 base64 的函数
					base64Image, err := downloadImageAndConvertToBase64(imageURL)
					if err != nil {
						mylog.Printf("下载或转换图片失败: %v", err)
					} else {
						// 清除原来的 HTTP 图片地址
						newMessage.Image = ""

						// 将base64内容从字符串转换回字节
						fileImageData, err := base64.StdEncoding.DecodeString(base64Image)
						if err != nil {
							mylog.Printf("Base64 解码失败: %v", err)
						} else {
							// 压缩图片
							compressedData, err := images.CompressSingleImage(fileImageData)
							if err != nil {
								mylog.Printf("Error compressing image: %v", err)
							}
							// 使用 Multipart 方法发送
							if _, err = api.PostMessageMultipart(context.TODO(), channelID.(string), newMessage, compressedData); err != nil {
								mylog.Printf("40003重试,使用 multipart 发送图文混合信息失败: %v message_id %v", err, messageID)
							}
						}
					}
				}
			} else {
				// 将base64内容从reply的Content转换回字节
				fileImageData, err := base64.StdEncoding.DecodeString(Reply.Content)
				if err != nil {
					mylog.Printf("Base64 解码失败: %v", err)
				}
				// 压缩图片
				compressedData, err := images.CompressSingleImage(fileImageData)
				if err != nil {
					mylog.Printf("Error compressing image: %v", err)
				}
				// 创建包含文本和图像信息的消息
				msgseq = echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				newMessage := &dto.MessageToCreate{
					Content: messageText,
					MsgID:   messageID,
					MsgSeq:  msgseq,
					MsgType: 0,
				}
				newMessage.Timestamp = time.Now().Unix() // 设置时间戳
				// 使用Multipart方法发送
				if resp, err = api.PostMessageMultipart(context.TODO(), channelID.(string), newMessage, compressedData); err != nil {
					mylog.Printf("使用multipart发送图文信息失败: %v message_id %v", err, messageID)
				}
			}
			// 发送成功回执
			retmsg, _ = SendGuildResponse(client, err, &message, resp)
			delete(foundItems, imageType) // 从foundItems中删除已处理的图片项
			messageText = ""
		}

		// 优先发送文本信息
		var err error
		if messageText != "" {
			msgseq := echo.GetMappingSeq(messageID)
			echo.AddMappingSeq(messageID, msgseq+1)
			textMsg, _ := GenerateReplyMessage(messageID, nil, messageText, msgseq+1)
			if resp, err = api.PostMessage(context.TODO(), channelID.(string), textMsg); err != nil {
				mylog.Printf("发送文本信息失败: %v", err)
			}
			//发送成功回执
			retmsg, _ = SendGuildResponse(client, err, &message, resp)
		}

		// 遍历foundItems并发送每种信息
		for key, urls := range foundItems {
			for _, url := range urls {
				singleItem[key] = []string{url} // 创建一个只有一个 URL 的 singleItem
				msgseq := echo.GetMappingSeq(messageID)
				echo.AddMappingSeq(messageID, msgseq+1)
				reply, isBase64Image := GenerateReplyMessage(messageID, singleItem, "", msgseq+1)

				if isBase64Image {
					// 将base64内容从reply的Content转换回字节
					fileImageData, err := base64.StdEncoding.DecodeString(reply.Content)
					if err != nil {
						mylog.Printf("Base64 解码失败: %v", err)
						continue // 跳过当前项，继续下一个
					}

					// 清除reply的Content
					reply.Content = ""
					// 压缩图片
					compressedData, err := images.CompressSingleImage(fileImageData)
					if err != nil {
						mylog.Printf("Error compressing image: %v", err)
					}
					// 使用Multipart方法发送
					if resp, err = api.PostMessageMultipart(context.TODO(), channelID.(string), reply, compressedData); err != nil {
						mylog.Printf("使用multipart发送 %s 信息失败: %v message_id %v", key, err, messageID)
					}
					//发送成功回执
					retmsg, _ = SendGuildResponse(client, err, &message, resp)
				} else {
					if _, err = api.PostMessage(context.TODO(), channelID.(string), reply); err != nil {
						mylog.Printf("发送 %s 信息失败: %v", key, err)
					}
					// 检查是否是 40003 错误
					if err != nil && strings.Contains(err.Error(), `"code":40003`) && len(reply.Image) > 0 {
						// 从 reply.Image 中提取图片地址
						imageURL := reply.Image

						// 调用下载并转换为 base64 的函数
						base64Image, err := downloadImageAndConvertToBase64(imageURL)
						if err != nil {
							mylog.Printf("下载或转换图片失败: %v", err)
						} else {
							// 清除原来的 HTTP 图片地址
							reply.Image = ""

							// 将base64内容从字符串转换回字节
							fileImageData, err := base64.StdEncoding.DecodeString(base64Image)
							if err != nil {
								mylog.Printf("Base64 解码失败: %v", err)
							} else {
								// 压缩图片
								compressedData, err := images.CompressSingleImage(fileImageData)
								if err != nil {
									mylog.Printf("Error compressing image: %v", err)
								}
								// 使用 Multipart 方法发送
								if resp, err = api.PostMessageMultipart(context.TODO(), channelID.(string), reply, compressedData); err != nil {
									mylog.Printf("40003重试,使用 multipart 发送 %s 信息失败: %v message_id %v", key, err, messageID)
								}
							}
						}
					}
					//发送成功回执
					retmsg, _ = SendGuildResponse(client, err, &message, resp)
				}
			}
		}
	//频道私信 此时直接取出
	case "guild_private":
		params := message.Params
		channelID := params.ChannelID
		guildID := params.GuildID
		var RChannelID string
		var err error
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err = idmap.RetrieveRowByIDv2(channelID.(string))
		if err != nil {
			mylog.Printf("error retrieving real UserID: %v", err)
		}
		RguildID := guildID.(string)
		retmsg, _ = HandleSendGuildChannelPrivateMsg(client, api, apiv2, message, &RguildID, &RChannelID)
	case "forum":
		//api一样的 直接丢进去试试
		retmsg, _ = HandleSendGuildChannelForum(client, api, apiv2, message)
	default:
		mylog.Printf("2Unknown message type: %s", msgType)
	}
	return retmsg, nil
}

// 组合发频道信息需要的MessageToCreate 支持base64
func GenerateReplyMessage(id string, foundItems map[string][]string, messageText string, msgseq int) (*dto.MessageToCreate, bool) {
	var reply dto.MessageToCreate
	var isBase64 bool

	if imageURLs, ok := foundItems["local_image"]; ok && len(imageURLs) > 0 {
		// 从本地图路径读取图片
		imageData, err := os.ReadFile(imageURLs[0])
		if err != nil {
			// 读入文件,如果是本地图,应用端和gensokyo需要在一台电脑
			mylog.Printf("Error reading the image from path %s: %v", imageURLs[0], err)
			// 发文本信息，提示图片文件不存在
			reply = dto.MessageToCreate{
				Content: "错误: 图片文件不存在",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}
			return &reply, false
		}
		// 首先压缩图片
		compressedData, err := images.CompressSingleImage(imageData)
		if err != nil {
			mylog.Printf("Error compressing image: %v", err)
			return &dto.MessageToCreate{
				Content: "错误: 压缩图片失败",
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0, // 默认文本类型
			}, false
		}
		//base64编码
		base64Encoded := base64.StdEncoding.EncodeToString(compressedData)

		// 当作base64图来处理
		reply = dto.MessageToCreate{
			Content: base64Encoded,
			MsgID:   id,
			MsgSeq:  msgseq,
			MsgType: 0, // Assuming type 0 for images
		}
		isBase64 = true
	} else if imageURLs, ok := foundItems["url_image"]; ok && len(imageURLs) > 0 {
		// 判断是否需要将图片转换为 base64 编码
		if config.GetGuildUrlImageToBase64() {
			base64Image, err := downloadImageAndConvertToBase64("http://" + imageURLs[0])
			if err == nil {
				reply = dto.MessageToCreate{
					Content: base64Image, // 使用转换后的 base64 图片作为 Content
					MsgID:   id,
					MsgSeq:  msgseq,
					MsgType: 0, // 默认类型为文本
				}
				isBase64 = true
			} // 错误处理逻辑
		} else {
			// 原有逻辑
			reply = dto.MessageToCreate{
				Image:   "http://" + imageURLs[0],
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0,
			}
		}
	} else if imageURLs, ok := foundItems["url_images"]; ok && len(imageURLs) > 0 {
		// 判断是否需要将图片转换为 base64 编码
		if config.GetGuildUrlImageToBase64() {
			base64Image, err := downloadImageAndConvertToBase64("https://" + imageURLs[0])
			if err == nil {
				reply = dto.MessageToCreate{
					Content: base64Image, // 使用转换后的 base64 图片作为 Content
					MsgID:   id,
					MsgSeq:  msgseq,
					MsgType: 0, // 默认类型为文本
				}
				isBase64 = true
			} // 错误处理逻辑
		} else {
			// 原有逻辑
			reply = dto.MessageToCreate{
				Image:   "https://" + imageURLs[0],
				MsgID:   id,
				MsgSeq:  msgseq,
				MsgType: 0,
			}
		}
	} else if voiceURLs, ok := foundItems["base64_record"]; ok && len(voiceURLs) > 0 {
		//频道 还不支持发语音
		// Sending a voice message
		// reply = dto.MessageToCreate{
		// 	EventID: id,
		// 	Embed: &dto.Embed{
		// 		URL: voiceURLs[0], // Assuming voice is embedded using a URL
		// 	},
		// 	MsgID:   id,
		// 	MsgType: 0, // Adjust type as needed for voice
		// }
	} else if base64_image, ok := foundItems["base64_image"]; ok && len(base64_image) > 0 {
		// base64图片
		reply = dto.MessageToCreate{
			Content: base64_image[0], // 直接使用base64_image[0]作为Content
			MsgID:   id,
			MsgSeq:  msgseq,
			MsgType: 0, // Default type for text
		}
		isBase64 = true
	} else {
		// 发文本信息
		reply = dto.MessageToCreate{
			//EventID: id,
			Content: messageText,
			MsgID:   id,
			MsgSeq:  msgseq,
			MsgType: 0, // Default type for text
		}
	}

	return &reply, isBase64
}

// downloadImageAndConvertToBase64 下载图片并转换为 base64 编码字符串
func downloadImageAndConvertToBase64(url string) (string, error) {
	// 发送 HTTP GET 请求以获取图片数据
	resp, err := http.Get(url)
	if err != nil {
		return "", err // 返回错误
	}
	defer resp.Body.Close()

	// 读取响应的内容
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err // 返回错误
	}

	// 将图片数据转换为 base64 编码
	base64Image := base64.StdEncoding.EncodeToString(data)

	// 返回 base64 编码的字符串和 nil（表示没有错误）
	return base64Image, nil
}
