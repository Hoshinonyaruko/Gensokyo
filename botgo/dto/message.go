package dto

// Message 消息结构体定义
type Message struct {
	// 消息ID
	ID string `json:"id"`
	// 子频道ID
	ChannelID string `json:"channel_id"`
	// 频道ID
	GuildID string `json:"guild_id"`
	// 群ID
	GroupID string `json:"group_id"`

	// 内容
	Content string `json:"content"`
	// 发送时间
	Timestamp Timestamp `json:"timestamp"`
	// 消息编辑时间
	EditedTimestamp Timestamp `json:"edited_timestamp"`
	// 是否@all
	MentionEveryone bool `json:"mention_everyone"`
	// 消息发送方
	Author *User `json:"author"`
	// 消息发送方Author的member属性，只是部分属性
	Member *Member `json:"member"`
	// 附件
	Attachments []*MessageAttachment `json:"attachments"`
	// 结构化消息-embeds
	Embeds []*Embed `json:"embeds"`
	// 消息中的提醒信息(@)列表
	Mentions []*User `json:"mentions"`
	// ark 消息
	Ark *Ark `json:"ark"`
	// 私信消息
	DirectMessage bool `json:"direct_message"`
	// 子频道 seq，用于消息间的排序，seq 在同一子频道中按从先到后的顺序递增，不同的子频道之前消息无法排序
	SeqInChannel string `json:"seq_in_channel"`
	// 引用的消息
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	// 私信场景下，该字段用来标识从哪个频道发起的私信
	SrcGuildID string `json:"src_guild_id"`
	//返回的ret 超过主动限制会返回22009
	Ret int `json:"ret,omitempty"`
}

// Forum 消息结构体定义
type Forum struct {
	// 消息ID
	TaskId string `json:"task_id"`
	// 发送时间 秒级时间戳
	CreateTime string `json:"create_time"`
}

// GroupAddBotEvent 表示群添加机器人事件的数据结构
type GroupAddBotEvent struct {
	GroupOpenID    string      `json:"group_openid"`
	OpMemberOpenID string      `json:"op_member_openid"`
	Timestamp      interface{} `json:"timestamp"`
}

type MediaResponse struct {
	//UUID
	FileUUID string `json:"file_uuid"`
	//file_info
	FileInfo string `json:"file_info"`
	TTL      int    `json:"ttl"`
	//返回的ret 超过主动限制会返回22009
	Ret int `json:"ret,omitempty"`
}

//群信息结构
type GroupMessageResponse struct {
	MediaResponse *MediaResponse
	Message       *Message
}

// C2CMessageResponse 用于包装 C2C 消息的响应
type C2CMessageResponse struct {
	Message       *Message       `json:"message,omitempty"`
	MediaResponse *MediaResponse `json:"media_response,omitempty"`
}

// Embed 结构
type Embed struct {
	Title       string                `json:"title,omitempty"`
	Description string                `json:"description,omitempty"`
	Prompt      string                `json:"prompt"` // 消息弹窗内容，消息列表摘要
	Thumbnail   MessageEmbedThumbnail `json:"thumbnail,omitempty"`
	Fields      []*EmbedField         `json:"fields,omitempty"`
}

// MessageEmbedThumbnail embed 消息的缩略图对象
type MessageEmbedThumbnail struct {
	URL string `json:"url"`
}

// EmbedField Embed字段描述
type EmbedField struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// MessageAttachment 附件定义
type MessageAttachment struct {
	URL         string `json:"url,omitempty"`
	FileName    string `json:"filename,omitempty"`
	Height      int    `json:"height,omitempty"`
	Size        int    `json:"size,omitempty"`
	Width       int    `json:"width,omitempty"`
	ContentType string `json:"content_type,omitempty"` // voice:语音, image/xxx: 图片 video/xxx: 视频
}

// MessageReactionUsers 消息表情表态用户列表
type MessageReactionUsers struct {
	Users  []*User `json:"users,omitempty"`
	Cookie string  `json:"cookie,omitempty"`
	IsEnd  bool    `json:"is_end,omitempty"`
}
