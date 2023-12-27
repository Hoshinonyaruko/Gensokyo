// 处理收到的信息事件
package Processor

import (
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
)

// ProcessGroupDelBot 处理机器人减少
func (p *Processors) ProcessGroupDelBot(data *dto.GroupAddBotEvent) error {
	var userid64 int64
	var GroupID64 int64
	var err error
	var Notice GroupNoticeEvent
	if config.GetIdmapPro() {
		GroupID64, userid64, err = idmap.StoreIDv2Pro(data.GroupOpenID, data.OpMemberOpenID)
		if err != nil {
			mylog.Fatalf("Error storing ID: %v", err)
		}
	} else {
		GroupID64, err = idmap.StoreIDv2(data.GroupOpenID)
		if err != nil {
			mylog.Errorf("failed to convert ChannelID to int: %v", err)
			return nil
		}
		userid64, err = idmap.StoreIDv2(data.OpMemberOpenID)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
	}
	mylog.Printf("Bot被[%v]从群[%v]移出", userid64, GroupID64)
	Notice = GroupNoticeEvent{
		GroupID:    GroupID64,
		NoticeType: "group_decrease",
		OperatorID: 0,
		PostType:   "notice",
		SelfID:     int64(config.GetAppID()),
		SubType:    "kick_me",
		Time:       data.Timestamp,
		UserID:     userid64,
	}
	groupMsgMap := structToMap(Notice)
	//上报信息到onebotv11应用端(正反ws)
	p.BroadcastMessageToAll(groupMsgMap)
	return nil
}
