// 处理收到的回调事件
package Processor

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/handlers"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket/client"
)

// ProcessGroupMsgReject 处理群关闭机器人推送
func (p *Processors) ProcessGroupMsgReject(data *dto.GroupMsgRejectEvent) error {
	// 转换appid
	var userid64 int64
	var GroupID64 int64
	var LongGroupID64 int64
	var err error
	var fromgid, fromuid string
	if data.GroupOpenID != "" {
		fromgid = data.GroupOpenID
		fromuid = data.OpMemberOpenID
	}

	// 获取s
	s := client.GetGlobalS()
	// 转换appid
	AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

	// 获取当前时间的13位毫秒级时间戳
	currentTimeMillis := time.Now().UnixNano() / 1e6

	// 构造echostr，包括AppID，原始的s变量和当前时间戳
	echostr := fmt.Sprintf("%s_%d_%d", AppIDString, s, currentTimeMillis)

	fmt.Printf("快乐测试[%v][%v]\n", fromgid, fromuid)

	if config.GetIdmapPro() {
		//将真实id转为int userid64
		GroupID64, userid64, err = idmap.StoreIDv2Pro(fromgid, fromuid)
		if err != nil {
			mylog.Fatalf("Error storing ID: %v", err)
		}
		// 当哈希碰撞 因为获取时候是用的非idmap的get函数
		LongGroupID64, _ = idmap.StoreIDv2(fromgid)
		_, _ = idmap.StoreIDv2(fromuid)
		if !config.GetHashIDValue() {
			mylog.Fatalf("避坑日志:你开启了高级id转换,请设置hash_id为true,并且删除idmaps并重启")
		}
	} else {
		// 映射str的GroupID到int
		GroupID64, err = idmap.StoreIDv2(fromgid)
		if err != nil {
			mylog.Errorf("failed to convert ChannelID to int: %v", err)
			return nil
		}
		// 映射str的userid到int
		userid64, err = idmap.StoreIDv2(fromuid)
		if err != nil {
			mylog.Printf("Error storing ID: %v", err)
			return nil
		}
		// 修复不开启idmap-pro问题
		LongGroupID64 = GroupID64
	}
	var selfid64 int64
	if config.GetUseUin() {
		selfid64 = config.GetUinint64()
	} else {
		selfid64 = int64(p.Settings.AppID)
	}

	if !config.GetGlobalGroupMsgRejectReciveEventToMessage() {
		notice := &OnebotGroupRejectNotice{
			GroupID:    GroupID64,
			NoticeType: "group_reject",
			PostType:   "notice",
			SelfID:     selfid64,
			SubType:    "create",
			Time:       time.Now().Unix(),
			UserID:     userid64,
			Data:       data,
		}
		//调试
		PrintStructWithFieldNames(notice)

		// Convert OnebotGroupMessage to map and send
		noticeMap := structToMap(notice)

		//上报信息到onebotv11应用端(正反ws)
		go p.BroadcastMessageToAll(noticeMap, p.Apiv2, data)

		// 转换appid
		AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

		// 储存和群号相关的eventid
		// idmap-pro的设计其实是有问题的,和idmap冲突,并且也还是会哈希碰撞 需要用一个不会碰撞的id去存
		echo.AddEvnetID(AppIDString, LongGroupID64, data.EventID)
	} else {
		if data.GroupOpenID != "" {
			//群回调
			newdata := ConvertRejectToMessage(data)
			//mylog.Printf("回调测试111-newdata:%v\n", newdata)

			// 如果在Array模式下, 则处理Message为Segment格式
			var segmentedMessages interface{} = newdata.Content
			if config.GetArrayValue() {
				segmentedMessages = handlers.ConvertToSegmentedMessage(newdata)
			}

			var IsBindedUserId, IsBindedGroupId bool
			if config.GetHashIDValue() {
				IsBindedUserId = idmap.CheckValue(data.OpMemberOpenID, userid64)
				IsBindedGroupId = idmap.CheckValue(data.GroupOpenID, GroupID64)
			} else {
				IsBindedUserId = idmap.CheckValuev2(userid64)
				IsBindedGroupId = idmap.CheckValuev2(GroupID64)
			}
			//平台事件,不是真实信息,无需messageID
			messageID64 := 123
			messageID := int(messageID64)
			var selfid64 int64
			if config.GetUseUin() {
				selfid64 = config.GetUinint64()
			} else {
				selfid64 = int64(p.Settings.AppID)
			}
			//mylog.Printf("回调测试-interaction:%v\n", segmentedMessages)
			groupMsg := OnebotGroupMessage{
				RawMessage:  newdata.Content,
				Message:     segmentedMessages,
				MessageID:   messageID,
				GroupID:     GroupID64,
				MessageType: "group",
				PostType:    "message",
				SelfID:      selfid64,
				UserID:      userid64,
				Sender: Sender{
					UserID: userid64,
					Sex:    "0",
					Age:    0,
					Area:   "0",
					Level:  "0",
				},
				SubType: "normal",
				Time:    time.Now().Unix(),
			}
			//增强配置
			if !config.GetNativeOb11() {
				groupMsg.RealMessageType = "group_reject"
				groupMsg.IsBindedUserId = IsBindedUserId
				groupMsg.IsBindedGroupId = IsBindedGroupId
				groupMsg.RealGroupID = data.GroupOpenID
				groupMsg.RealUserID = data.OpMemberOpenID
				groupMsg.Avatar, _ = GenerateAvatarURLV2(data.OpMemberOpenID)
			}
			//根据条件判断是否增加nick和card
			var CaN = config.GetCardAndNick()
			if CaN != "" {
				groupMsg.Sender.Nickname = CaN
				groupMsg.Sender.Card = CaN
			}
			// 根据条件判断是否添加Echo字段
			if config.GetTwoWayEcho() {
				groupMsg.Echo = echostr
				//用向应用端(如果支持)发送echo,来确定客户端的send_msg对应的触发词原文
				echo.AddMsgIDv3(AppIDString, echostr, newdata.Content)
			}
			// 获取MasterID数组
			masterIDs := config.GetMasterID()

			// 判断userid64是否在masterIDs数组里
			isMaster := false
			for _, id := range masterIDs {
				if strconv.FormatInt(userid64, 10) == id {
					isMaster = true
					break
				}
			}

			// 根据isMaster的值为groupMsg的Sender赋值role字段
			if isMaster {
				groupMsg.Sender.Role = "owner"
			} else {
				groupMsg.Sender.Role = "member"
			}

			// 映射消息类型
			echo.AddMsgType(AppIDString, s, "group")

			//储存当前群或频道号的类型
			idmap.WriteConfigv2(fmt.Sprint(GroupID64), "type", "group")

			//映射类型
			echo.AddMsgType(AppIDString, GroupID64, "group")

			// 调试
			PrintStructWithFieldNames(groupMsg)

			// Convert OnebotGroupMessage to map and send
			groupMsgMap := structToMap(groupMsg)
			//上报信息到onebotv11应用端(正反ws)
			go p.BroadcastMessageToAll(groupMsgMap, p.Apiv2, data)

			// 转换appid
			AppIDString := strconv.FormatUint(p.Settings.AppID, 10)

			// 储存和群号相关的eventid
			fmt.Printf("测试:储存eventid:[%v]LongGroupID64[%v]\n", data.EventID, LongGroupID64)
			echo.AddEvnetID(AppIDString, LongGroupID64, data.EventID)
		}
	}

	return nil
}

// ConvertRejectToMessage 转换 Reject 到 Message
func ConvertRejectToMessage(r *dto.GroupMsgRejectEvent) *dto.Message {
	var message dto.Message

	// 直接映射的字段
	message.GroupID = r.GroupOpenID

	// 特殊处理的字段
	message.Content = config.GetGlobalGroupMsgRejectMessage()
	message.DirectMessage = false

	return &message
}
