package multi

import (
	"sync"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/log"
	"github.com/tencent-connect/botgo/sessions/manager"
	"github.com/tencent-connect/botgo/token"
	"github.com/tencent-connect/botgo/websocket"
)

type ShardManager struct {
	Sessions      []dto.Session
	SessionChans  []chan dto.Session
	Clients       []websocket.WebSocket
	APInfo        *dto.WebsocketAP
	Token         *token.Token
	Intents       *dto.Intent
	StartInterval time.Duration
	wg            sync.WaitGroup
}

func NewShardManager(apInfo *dto.WebsocketAP, token *token.Token, intents *dto.Intent) *ShardManager {
	m := &ShardManager{
		APInfo:       apInfo,
		Token:        token,
		Intents:      intents,
		Sessions:     make([]dto.Session, apInfo.Shards),
		Clients:      make([]websocket.WebSocket, apInfo.Shards),
		SessionChans: make([]chan dto.Session, apInfo.Shards),
	}
	for i := range m.Sessions {
		m.SessionChans[i] = make(chan dto.Session, 1)
	}
	m.StartInterval = manager.CalcInterval(apInfo.SessionStartLimit.MaxConcurrency)
	return m
}

func (sm *ShardManager) StartAllShards() {
	for i := uint32(0); i < sm.APInfo.Shards; i++ {
		sm.StartShard(i)
	}
	sm.wg.Wait()
}

func (sm *ShardManager) StartShard(shardID uint32) {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		session := dto.Session{
			URL:     sm.APInfo.URL,
			Token:   *sm.Token,
			Intent:  *sm.Intents,
			LastSeq: 0,
			Shards: dto.ShardConfig{
				ShardID:    shardID,
				ShardCount: sm.APInfo.Shards,
			},
		}
		sm.Sessions[shardID] = session
		sm.SessionChans[shardID] <- session

		for session := range sm.SessionChans[shardID] {
			time.Sleep(sm.StartInterval)
			sm.newConnect(session, shardID)
		}
	}()
}

func (sm *ShardManager) newConnect(session dto.Session, shardID uint32) {
	wsClient := websocket.ClientImpl.New(session)
	sm.Clients[shardID] = wsClient
	if err := wsClient.Connect(); err != nil {
		log.Error(err)
		sm.SessionChans[shardID] <- session // Reconnect
		return
	}
	if session.ID != "" {
		err := wsClient.Resume()
		if err != nil {
			log.Errorf("[ws/session] Resume error: %+v", err)
			return
		}
	} else {
		err := wsClient.Identify()
		if err != nil {
			log.Errorf("[ws/session] Identify error: %+v", err)
			return
		}
	}
	if err := wsClient.Listening(); err != nil {
		log.Errorf("[ws/session] Listening error: %+v", err)
		sm.SessionChans[shardID] <- session // Reconnect
	}
}
