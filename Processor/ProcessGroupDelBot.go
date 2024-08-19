// 处理收到的信息事件
package Processor

import (
	"fmt"
	"strconv"

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
			mylog.Errorf("Error storing ID: %v", err)
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
	var timestampInt64 int64
	switch v := data.Timestamp.(type) {
	case string:
		timestampInt64, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			mylog.Printf("Error converting timestamp string to int64: %v", err)
			return nil
		}
	case int64:
		timestampInt64 = v
	case float64:
		timestampInt64 = int64(v)
	default:
		mylog.Printf("Invalid type for timestamp: %T", v)
		return nil
	}
	mylog.Printf("Bot被[%v]从群[%v]移出", userid64, GroupID64)
	//从数据库删除群数据(仅删除类型缓存,再次加入会刷新)
	idmap.DeleteConfigv2(fmt.Sprint(GroupID64), "type")

	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}
	Notice = GroupNoticeEvent{
		GroupID:    GroupID64,
		NoticeType: "group_decrease",
		OperatorID: 0,
		PostType:   "notice",
		SelfID:     selfid64,
		SubType:    "kick_me",
		Time:       timestampInt64,
		UserID:     userid64,
	}
	//增强配置
	if !config.GetNativeOb11() {
		Notice.RealUserID = data.OpMemberOpenID
		Notice.RealGroupID = data.GroupOpenID
	}
	groupMsgMap := structToMap(Notice)
	//上报信息到onebotv11应用端(正反ws)
	go p.BroadcastMessageToAll(groupMsgMap, p.Apiv2, data)
	return nil
}
