package dto

import "encoding/json"

// ForumContentElement 接口定义了所有内容元素共有的方法，这里我们使用一个空接口，因为不同元素的方法会有所不同
type ForumContentElement interface{}

// ForumTextElement 文本元素结构体
type ForumTextElement struct {
	Type     int `json:"type"` // 类型标识，例如1代表文本
	TextInfo struct {
		Text string `json:"text"`
	} `json:"text"` //这里文档与实际情况不一致
}

// ForumChannelElement 频道元素结构体
type ForumChannelElement struct {
	Type        int `json:"type"` // 类型标识，例如5代表频道信息
	ChannelInfo struct {
		ChannelID   int    `json:"channel_id"`
		ChannelName string `json:"channel_name"`
	} `json:"channel_info"`
}

// ForumURLElement 链接元素结构体
type ForumURLElement struct {
	Type    int `json:"type"` // 类型标识，例如3代表URL
	URLInfo struct {
		URL         string `json:"url"`
		DisplayText string `json:"display_text"`
	} `json:"url_info"`
}

// 定义匹配JSON结构的新结构体
type ForumContentStructure struct {
	Paragraphs []struct {
		Elems []json.RawMessage `json:"elems"` // 使用json.RawMessage延迟解析
		Props json.RawMessage   `json:"props"`
	} `json:"paragraphs"`
}

// ThreadInfo 主题信息结构体更新，使用ContentElement接口切片来表示复杂的Content
type ThreadInfo struct {
	ThreadID string `json:"thread_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	DateTime string `json:"date_time"`
}

// Thread 主题事件主体内容
type Thread struct {
	GuildID    string     `json:"guild_id"`
	ChannelID  string     `json:"channel_id"`
	AuthorID   string     `json:"author_id"`
	ThreadInfo ThreadInfo `json:"thread_info"`
	ID         string     `json:"id,omitempty"` // 新增字段以存储ID
}

// Post 帖子事件主体内容
type Post struct {
	GuildID   string   `json:"guild_id"`
	ChannelID string   `json:"channel_id"`
	AuthorID  string   `json:"author_id"`
	PostInfo  PostInfo `json:"post_info"`
}

// PostInfo 帖子内容
type PostInfo struct {
	ThreadID string `json:"thread_id"`
	PostID   string `json:"post_id"`
	Content  string `json:"content"`
	DateTime string `json:"date_time"`
}

// Reply 回复事件主体内容
type Reply struct {
	GuildID   string    `json:"guild_id"`
	ChannelID string    `json:"channel_id"`
	AuthorID  string    `json:"author_id"`
	ReplyInfo ReplyInfo `json:"reply_info"`
}

// ReplyInfo 回复内容
type ReplyInfo struct {
	ThreadID string `json:"thread_id"`
	PostID   string `json:"post_id"`
	ReplyID  string `json:"reply_id"`
	Content  string `json:"content"`
	DateTime string `json:"date_time"`
}

// ForumAuditResult 帖子审核事件主体内容
type ForumAuditResult struct {
	TaskID      string `json:"task_id"`
	GuildID     string `json:"guild_id"`
	ChannelID   string `json:"channel_id"`
	AuthorID    string `json:"author_id"`
	ThreadID    string `json:"thread_id"`
	PostID      string `json:"post_id"`
	ReplyID     string `json:"reply_id"`
	PublishType uint32 `json:"type"`
	Result      uint32 `json:"result"`
	ErrMsg      string `json:"err_msg"`
	DateTime    string `json:"date_time"`
}
