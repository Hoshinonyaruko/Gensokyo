package dto

import (
	"fmt"

	"github.com/tencent-connect/botgo/token"
)

// WebsocketAP wss 接入点信息
type WebsocketAP struct {
	URL               string            `json:"url"`
	Shards            uint32            `json:"shards"`
	SessionStartLimit SessionStartLimit `json:"session_start_limit"`
}

// WebsocketAP wss 单个接入点信息
type WebsocketAPSingle struct {
	URL               string            `json:"url"`
	ShardCount        uint32            `json:"shards"`   //最大值 比如是4个分片就是4
	ShardID           uint32            `json:"shard_id"` //从0开始的 0 1 2 3 对应上面的
	SessionStartLimit SessionStartLimit `json:"session_start_limit"`
}

// SessionStartLimit 链接频控信息
type SessionStartLimit struct {
	Total          uint32 `json:"total"`
	Remaining      uint32 `json:"remaining"`
	ResetAfter     uint32 `json:"reset_after"`
	MaxConcurrency uint32 `json:"max_concurrency"`
}

// ShardConfig 连接的 shard 配置，ShardID 从 0 开始，ShardCount 最小为 1
type ShardConfig struct {
	ShardID    uint32
	ShardCount uint32
}

// Session 连接的 session 结构，包括链接的所有必要字段
type Session struct {
	ID      string
	URL     string
	Token   token.Token
	Intent  Intent
	LastSeq uint32
	Shards  ShardConfig
}

// String 输出session字符串
func (s *Session) String() string {
	return fmt.Sprintf("[ws][ID:%s][Shard:(%d/%d)][Intent:%d]",
		s.ID, s.Shards.ShardID, s.Shards.ShardCount, s.Intent)
}

// WSUser 当前连接的用户信息
type WSUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Bot      bool   `json:"bot"`
}
