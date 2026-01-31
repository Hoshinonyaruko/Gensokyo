package dto

func init() {
	eventIntentMap = transposeIntentEventMap(intentEventMap)
}

// 事件类型
const (
	EventGuildCreate           EventType = "GUILD_CREATE"
	EventGuildUpdate           EventType = "GUILD_UPDATE"
	EventGuildDelete           EventType = "GUILD_DELETE"
	EventChannelCreate         EventType = "CHANNEL_CREATE"
	EventChannelUpdate         EventType = "CHANNEL_UPDATE"
	EventChannelDelete         EventType = "CHANNEL_DELETE"
	EventGuildMemberAdd        EventType = "GUILD_MEMBER_ADD"
	EventGuildMemberUpdate     EventType = "GUILD_MEMBER_UPDATE"
	EventGuildMemberRemove     EventType = "GUILD_MEMBER_REMOVE"
	EventMessageCreate         EventType = "MESSAGE_CREATE"
	EventMessageReactionAdd    EventType = "MESSAGE_REACTION_ADD"
	EventMessageReactionRemove EventType = "MESSAGE_REACTION_REMOVE"
	EventAtMessageCreate       EventType = "AT_MESSAGE_CREATE"
	EventPublicMessageDelete   EventType = "PUBLIC_MESSAGE_DELETE"
	EventDirectMessageCreate   EventType = "DIRECT_MESSAGE_CREATE"
	EventDirectMessageDelete   EventType = "DIRECT_MESSAGE_DELETE"
	EventAudioStart            EventType = "AUDIO_START"
	EventAudioFinish           EventType = "AUDIO_FINISH"
	EventAudioOnMic            EventType = "AUDIO_ON_MIC"
	EventAudioOffMic           EventType = "AUDIO_OFF_MIC"
	EventMessageAuditPass      EventType = "MESSAGE_AUDIT_PASS"
	EventMessageAuditReject    EventType = "MESSAGE_AUDIT_REJECT"
	EventMessageDelete         EventType = "MESSAGE_DELETE"
	EventForumThreadCreate     EventType = "FORUM_THREAD_CREATE"
	EventForumThreadUpdate     EventType = "FORUM_THREAD_UPDATE"
	EventForumThreadDelete     EventType = "FORUM_THREAD_DELETE"
	EventForumPostCreate       EventType = "FORUM_POST_CREATE"
	EventForumPostDelete       EventType = "FORUM_POST_DELETE"
	EventForumReplyCreate      EventType = "FORUM_REPLY_CREATE"
	EventForumReplyDelete      EventType = "FORUM_REPLY_DELETE"
	EventForumAuditResult      EventType = "FORUM_PUBLISH_AUDIT_RESULT"
	EventInteractionCreate     EventType = "INTERACTION_CREATE"
	EventGroupAtMessageCreate  EventType = "GROUP_AT_MESSAGE_CREATE"
	EventC2CMessageCreate      EventType = "C2C_MESSAGE_CREATE"
	EventGroupAddRobot         EventType = "GROUP_ADD_ROBOT"
	EventGroupDelRobot         EventType = "GROUP_DEL_ROBOT"
	EventGroupMsgReject        EventType = "GROUP_MSG_REJECT"
	EventGroupMsgReceive       EventType = "GROUP_MSG_RECEIVE"
	// [新增] 用户关系链与C2C开关事件
	EventFriendAdd     EventType = "FRIEND_ADD"
	EventFriendDel     EventType = "FRIEND_DEL"
	EventC2CMsgReject  EventType = "C2C_MSG_REJECT"
	EventC2CMsgReceive EventType = "C2C_MSG_RECEIVE"
)

// intentEventMap 不同 intent 对应的事件定义
var intentEventMap = map[Intent][]EventType{
	IntentGuilds: {
		EventGuildCreate, EventGuildUpdate, EventGuildDelete,
		EventChannelCreate, EventChannelUpdate, EventChannelDelete,
	},
	IntentGuildMembers:  {EventGuildMemberAdd, EventGuildMemberUpdate, EventGuildMemberRemove},
	IntentGuildMessages: {EventMessageCreate, EventMessageDelete},

	// [修改] 将新的事件加入 IntentGroupMessages
	IntentGroupMessages: {
		EventGroupAtMessageCreate, EventC2CMessageCreate, // 原有
		EventGroupAddRobot, EventGroupDelRobot, // 原有
		EventGroupMsgReject, EventGroupMsgReceive, // 补全群消息开关事件
		EventFriendAdd, EventFriendDel, // [新增] 好友添加/删除
		EventC2CMsgReject, EventC2CMsgReceive, // [新增] C2C消息开关拒绝/接收
	},

	IntentGuildMessageReactions: {EventMessageReactionAdd, EventMessageReactionRemove},
	IntentGuildAtMessage:        {EventAtMessageCreate, EventPublicMessageDelete},
	IntentDirectMessages:        {EventDirectMessageCreate, EventDirectMessageDelete},
	IntentAudio:                 {EventAudioStart, EventAudioFinish, EventAudioOnMic, EventAudioOffMic},
	IntentAudit:                 {EventMessageAuditPass, EventMessageAuditReject},
	IntentForum: {
		EventForumThreadCreate, EventForumThreadUpdate, EventForumThreadDelete, EventForumPostCreate,
		EventForumPostDelete, EventForumReplyCreate, EventForumReplyDelete, EventForumAuditResult,
	},
	IntentInteraction: {EventInteractionCreate},
}

var eventIntentMap = transposeIntentEventMap(intentEventMap)

// transposeIntentEventMap 转置 intent 与 event 的关系，用于根据 event 找到 intent
func transposeIntentEventMap(input map[Intent][]EventType) map[EventType]Intent {
	result := make(map[EventType]Intent)
	for i, eventTypes := range input {
		for _, s := range eventTypes {
			result[s] = i
		}
	}
	return result
}

// EventToIntent 事件转换对应的Intent
func EventToIntent(events ...EventType) Intent {
	var i Intent
	for _, event := range events {
		i = i | eventIntentMap[event]
	}
	return i
}
