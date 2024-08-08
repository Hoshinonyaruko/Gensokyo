package event

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/tidwall/gjson" // 由于回包的 d 类型不确定，gjson 用于从回包json中提取 d 并进行针对性的解析

	"github.com/tencent-connect/botgo/dto"
)

func init() {
	// Start a goroutine for periodic cleaning
	go cleanProcessedIDs()
}

func cleanProcessedIDs() {
	ticker := time.NewTicker(5 * time.Minute) // Adjust the interval as needed
	defer ticker.Stop()

	for range ticker.C {
		// Clean processedIDs, remove entries which are no longer needed
		processedIDs.Range(func(key, value interface{}) bool {
			processedIDs.Delete(key)
			return true
		})
	}
}

var processedIDs sync.Map

var eventParseFuncMap = map[dto.OPCode]map[dto.EventType]eventParseFunc{
	dto.WSDispatchEvent: {
		dto.EventGuildCreate: guildHandler,
		dto.EventGuildUpdate: guildHandler,
		dto.EventGuildDelete: guildHandler,

		dto.EventChannelCreate: channelHandler,
		dto.EventChannelUpdate: channelHandler,
		dto.EventChannelDelete: channelHandler,

		dto.EventGuildMemberAdd:    guildMemberHandler,
		dto.EventGuildMemberUpdate: guildMemberHandler,
		dto.EventGuildMemberRemove: guildMemberHandler,

		dto.EventMessageCreate: messageHandler,
		dto.EventMessageDelete: messageDeleteHandler,

		dto.EventMessageReactionAdd:    messageReactionHandler,
		dto.EventMessageReactionRemove: messageReactionHandler,

		dto.EventAtMessageCreate:     atMessageHandler,
		dto.EventPublicMessageDelete: publicMessageDeleteHandler,

		dto.EventDirectMessageCreate: directMessageHandler,
		dto.EventDirectMessageDelete: directMessageDeleteHandler,

		dto.EventAudioStart:  audioHandler,
		dto.EventAudioFinish: audioHandler,
		dto.EventAudioOnMic:  audioHandler,
		dto.EventAudioOffMic: audioHandler,

		dto.EventMessageAuditPass:   messageAuditHandler,
		dto.EventMessageAuditReject: messageAuditHandler,

		dto.EventForumThreadCreate: threadHandler,
		dto.EventForumThreadUpdate: threadHandler,
		dto.EventForumThreadDelete: threadHandler,
		dto.EventForumPostCreate:   postHandler,
		dto.EventForumPostDelete:   postHandler,
		dto.EventForumReplyCreate:  replyHandler,
		dto.EventForumReplyDelete:  replyHandler,
		dto.EventForumAuditResult:  forumAuditHandler,

		dto.EventInteractionCreate:    interactionHandler,
		dto.EventGroupAtMessageCreate: groupAtMessageHandler,
		dto.EventC2CMessageCreate:     c2cMessageHandler,
		dto.EventGroupAddRobot:        groupaddbothandler,
		dto.EventGroupDelRobot:        groupdelbothandler,
		dto.EventGroupMsgReject:       groupMsgRejecthandler,
		dto.EventGroupMsgReceive:      groupMsgReceivehandler,
	},
}

type eventParseFunc func(event *dto.WSPayload, message []byte) error

// ParseAndHandle 处理回调事件
func ParseAndHandle(payload *dto.WSPayload) error {
	// 指定类型的 handler
	if h, ok := eventParseFuncMap[payload.OPCode][payload.Type]; ok {
		return h(payload, payload.RawMessage)
	}
	// 透传handler，如果未注册具体类型的 handler，会统一投递到这个 handler
	if DefaultHandlers.Plain != nil {
		return DefaultHandlers.Plain(payload, payload.RawMessage)
	}
	return nil
}

// ParseData 解析数据
func ParseData(message []byte, target interface{}) error {
	// 获取数据部分
	data := gjson.Get(string(message), "d")
	// 外层ID 与内层ID不同 外层id是event_id 用于发送参数 d内层id是id,用于put回调接口
	eventid := gjson.Get(string(message), "id").String()

	// 使用switch语句处理不同类型
	switch v := target.(type) {
	case *dto.WSThreadData:
		// 特殊处理dto.WSThreadData
		if err := json.Unmarshal([]byte(data.String()), v); err != nil {
			return err
		}
		// 设置ID字段
		v.EventID = eventid
		return nil

	case *dto.GroupAddBotEvent:
		// 特殊处理dto.GroupAddBotEvent
		if err := json.Unmarshal([]byte(data.String()), v); err != nil {
			return err
		}
		// 设置ID字段
		v.EventID = eventid
		return nil

	case *dto.WSInteractionData:
		// 特殊处理dto.WSInteractionData
		if err := json.Unmarshal([]byte(data.String()), v); err != nil {
			return err
		}
		// 设置ID字段
		v.EventID = eventid
		return nil

	case *dto.GroupMsgRejectEvent:
		// 特殊处理dto.GroupMsgRejectEvent
		if err := json.Unmarshal([]byte(data.String()), v); err != nil {
			return err
		}
		// 设置ID字段
		v.EventID = eventid
		return nil

	case *dto.GroupMsgReceiveEvent:
		// 特殊处理dto.GroupMsgReceiveEvent
		if err := json.Unmarshal([]byte(data.String()), v); err != nil {
			return err
		}
		// 设置ID字段
		v.EventID = eventid
		return nil

	default:
		// 对于其他类型，继续原有逻辑
		return json.Unmarshal([]byte(data.String()), target)
	}
}

func guildHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSGuildData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Guild != nil {
		return DefaultHandlers.Guild(payload, data)
	}
	return nil
}

func channelHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSChannelData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Channel != nil {
		return DefaultHandlers.Channel(payload, data)
	}
	return nil
}

func guildMemberHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSGuildMemberData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.GuildMember != nil {
		return DefaultHandlers.GuildMember(payload, data)
	}
	return nil
}

func messageHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSMessageData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Message != nil {
		return DefaultHandlers.Message(payload, data)
	}
	return nil
}

func messageDeleteHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSMessageDeleteData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.MessageDelete != nil {
		return DefaultHandlers.MessageDelete(payload, data)
	}
	return nil
}

func messageReactionHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSMessageReactionData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.MessageReaction != nil {
		return DefaultHandlers.MessageReaction(payload, data)
	}
	return nil
}

func atMessageHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSATMessageData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.ATMessage != nil {
		return DefaultHandlers.ATMessage(payload, data)
	}
	return nil
}

func groupAtMessageHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSGroupATMessageData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if _, loaded := processedIDs.LoadOrStore(data.ID, struct{}{}); loaded {
		return nil
	}
	if DefaultHandlers.GroupATMessage != nil {
		return DefaultHandlers.GroupATMessage(payload, data)
	}
	return nil
}

func c2cMessageHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSC2CMessageData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.C2CMessage != nil {
		return DefaultHandlers.C2CMessage(payload, data)
	}
	return nil
}

func groupaddbothandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.GroupAddBotEvent{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.GroupAddbot != nil {
		return DefaultHandlers.GroupAddbot(payload, data)
	}
	return nil
}

func groupdelbothandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.GroupAddBotEvent{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.GroupDelbot != nil {
		return DefaultHandlers.GroupDelbot(payload, data)
	}
	return nil
}

func publicMessageDeleteHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSPublicMessageDeleteData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.PublicMessageDelete != nil {
		return DefaultHandlers.PublicMessageDelete(payload, data)
	}
	return nil
}

func directMessageHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSDirectMessageData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.DirectMessage != nil {
		return DefaultHandlers.DirectMessage(payload, data)
	}
	return nil
}

func directMessageDeleteHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSDirectMessageDeleteData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.DirectMessageDelete != nil {
		return DefaultHandlers.DirectMessageDelete(payload, data)
	}
	return nil
}

func audioHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSAudioData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Audio != nil {
		return DefaultHandlers.Audio(payload, data)
	}
	return nil
}

func threadHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSThreadData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Thread != nil {
		return DefaultHandlers.Thread(payload, data)
	}
	return nil
}

func postHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSPostData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Post != nil {
		return DefaultHandlers.Post(payload, data)
	}
	return nil
}

func replyHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSReplyData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Reply != nil {
		return DefaultHandlers.Reply(payload, data)
	}
	return nil
}

func forumAuditHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSForumAuditData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.ForumAudit != nil {
		return DefaultHandlers.ForumAudit(payload, data)
	}
	return nil
}

func messageAuditHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSMessageAuditData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.MessageAudit != nil {
		return DefaultHandlers.MessageAudit(payload, data)
	}
	return nil
}

func interactionHandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.WSInteractionData{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.Interaction != nil {
		return DefaultHandlers.Interaction(payload, data)
	}
	return nil
}

func groupMsgRejecthandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.GroupMsgRejectEvent{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.GroupMsgReject != nil {
		return DefaultHandlers.GroupMsgReject(payload, data)
	}
	return nil
}

func groupMsgReceivehandler(payload *dto.WSPayload, message []byte) error {
	data := &dto.GroupMsgReceiveEvent{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.GroupMsgReceive != nil {
		return DefaultHandlers.GroupMsgReceive(payload, data)
	}
	return nil
}
