package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_info", HandleGetGroupInfo)
}

type OnebotGroupInfo struct {
	Data    GroupInfo   `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

type GroupInfo struct {
	GroupID         int64  `json:"group_id"`
	GroupName       string `json:"group_name"`
	GroupMemo       string `json:"group_memo"`
	GroupCreateTime int32  `json:"group_create_time"`
	GroupLevel      int32  `json:"group_level"`
	MemberCount     int32  `json:"member_count"`
	MaxMemberCount  int32  `json:"max_member_count"`
}

type GroupInfoNotice struct {
	PostType   string `json:"post_type"`
	NoticeType string `json:"notice_type"`

	Status  string `json:"status"`
	RetCode int    `json:"retcode"`

	GroupID        int64    `json:"group_id,omitempty"`
	RealGroupID    string   `json:"real_group_id,omitempty"`
	GroupName      string   `json:"group_name"`
	GroupMemo      string   `json:"group_memo"`
	GroupClassText string   `json:"group_class_text"`
	GroupTags      []string `json:"group_tags"`
	MemberCount    int      `json:"member_count"`

	ErrorMsg string `json:"error_msg,omitempty"`

	Time   int64       `json:"time"`
	SelfID int64       `json:"self_id"`
	Echo   interface{} `json:"echo,omitempty"`
}

func ConvertGuildToGroupInfo(guild *dto.Guild, GroupId string, message callapi.ActionMessage) *OnebotGroupInfo {
	// 使用idmap.StoreIDv2映射GroupId到一个int64的值
	groupid64, err := idmap.StoreIDv2(GroupId)
	if err != nil {
		mylog.Printf("Error storing GroupID: %v", err)
		return nil
	}

	ts, err := guild.JoinedAt.Time()
	if err != nil {
		mylog.Printf("转换JoinedAt失败: %v", err)
		return nil
	}
	groupCreateTime := int32(ts.Unix())

	groupInfo := &GroupInfo{
		GroupID:         groupid64,
		GroupName:       guild.Name,
		GroupMemo:       guild.Desc,
		GroupCreateTime: groupCreateTime,
		GroupLevel:      0,
		MemberCount:     int32(guild.MemberCount),
		MaxMemberCount:  int32(guild.MaxMembers),
	}

	// 创建 OnebotGroupInfo 实例并填充数据
	onebotGroupInfo := &OnebotGroupInfo{
		Data:    *groupInfo,
		Message: "success",
		RetCode: 0,
		Status:  "ok",
	}
	if message.Echo == "" {
		onebotGroupInfo.Echo = "0"
	} else {
		onebotGroupInfo.Echo = message.Echo
	}

	return onebotGroupInfo
}

func HandleGetGroupInfo(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	params := message.Params

	// StringOB11 + QQ开放平台32字符群OpenID
	// 直接调用腾讯群基本信息API，并通过自定义Notice推送结果。
	// 不经过idmap，不转换int64，不影响原有频道get_group_info逻辑。
	if config.GetStringOb11() {
		groupID, ok := params.GroupID.(string)

		if ok && len(groupID) == 32 {
			resultData, err := apiv2.GetGroupInfo(
				context.Background(),
				groupID,
			)

			if err != nil {
				mylog.Printf(
					"GetGroupInfo failed, group_openid=%s, err=%v",
					groupID,
					err,
				)

				failedNotice := GroupInfoNotice{
					PostType:   "notice",
					NoticeType: "group_info_resp",

					Status:  "failed",
					RetCode: 100,

					RealGroupID: groupID,

					ErrorMsg: err.Error(),

					Time:   time.Now().Unix(),
					SelfID: int64(config.GetAppID()),
					Echo:   message.Echo,
				}

				outputMap := structToMap(failedNotice)

				mylog.Printf(
					"Pushing GroupInfo failed Notice: %+v\n",
					outputMap,
				)

				sendErr := client.SendMessage(outputMap)
				if sendErr != nil {
					mylog.Printf(
						"error sending group info failed notice via wsclient: %v",
						sendErr,
					)
				}

				return "{}", nil
			}

			fakeNotice := GroupInfoNotice{
				PostType:   "notice",
				NoticeType: "group_info_resp",

				Status:  "ok",
				RetCode: 0,

				RealGroupID:    resultData.GroupOpenID,
				GroupName:      resultData.GroupName,
				GroupMemo:      resultData.GroupFingerMemo,
				GroupClassText: resultData.GroupClassText,
				GroupTags:      resultData.GroupTags,
				MemberCount:    resultData.GroupMemberNum,

				Time:   time.Now().Unix(),
				SelfID: int64(config.GetAppID()),
				Echo:   message.Echo,
			}

			outputMap := structToMap(fakeNotice)

			mylog.Printf(
				"Pushing GroupInfo Notice: %+v\n",
				outputMap,
			)

			err = client.SendMessage(outputMap)
			if err != nil {
				mylog.Printf(
					"error sending group info notice via wsclient: %v",
					err,
				)
			}

			return "{}", nil
		}
	}
	// 使用 message.Echo 作为key来获取消息类型
	var msgType string
	var groupInfo *OnebotGroupInfo
	var err error
	if echoStr, ok := message.Echo.(string); ok {
		// 当 message.Echo 是字符串类型时执行此块
		msgType = echo.GetMsgTypeByKey(echoStr)
	}
	if msgType == "" {
		msgType = GetMessageTypeByGroupid(config.GetAppIDStr(), message.Params.GroupID)
	}
	if msgType == "" {
		msgType = GetMessageTypeByGroupidV2(message.Params.GroupID)
	}
	switch msgType {
	case "guild", "guild_private":
		//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
		ChannelID := params.GroupID
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		mylog.Printf("测试:%v", ChannelID.(string))
		RChannelID, err := idmap.RetrieveRowByIDv2(ChannelID.(string))
		if err != nil {
			mylog.Printf("error retrieving real ChannelID: %v", err)
		}
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(RChannelID, "guild_id")
		if err != nil {
			mylog.Printf("handleGetGroupInfo:Error reading config: %v\n", err)
			return "", nil
		}
		//最后获取到guildID
		guildID := value
		mylog.Printf("调试,准备groupInfoMap(频道)guildID:%v", guildID)
		guild, err := api.Guild(context.TODO(), guildID)
		if err != nil {
			mylog.Printf("获取频道信息失败: %v", err)
			return "", nil
		}
		groupInfo = ConvertGuildToGroupInfo(guild, guildID, message)
	default:
		var groupid int64
		groupid, _ = strconv.ParseInt(message.Params.GroupID.(string), 10, 64)
		groupCreateTime := time.Now().Unix()
		// 创建 GroupInfo 实例
		groupInfo1 := &GroupInfo{
			GroupID:         groupid,
			GroupName:       "测试群",
			GroupMemo:       "这是一个测试群",
			GroupCreateTime: int32(groupCreateTime),
			GroupLevel:      0,
			MemberCount:     500,
			MaxMemberCount:  1000,
		}
		// 创建 OnebotGroupInfo 实例并嵌入 GroupInfo
		groupInfo = &OnebotGroupInfo{
			Data:    *groupInfo1, // 将 groupInfo 添加到 Data 切片中
			Message: "success",
			RetCode: 0,
			Status:  "ok",
		}
		if message.Echo == "" {
			groupInfo.Echo = "0"
		} else {
			groupInfo.Echo = message.Echo
		}
	}
	groupInfoMap := structToMap(groupInfo)

	// 打印groupInfoMap的内容
	mylog.Printf("groupInfoMap(频道): %+v\n", groupInfoMap)

	err = client.SendMessage(groupInfoMap) //发回去
	if err != nil {
		mylog.Printf("error sending group info via wsclient: %v", err)
	}
	//把结果从struct转换为json
	result, err := json.Marshal(groupInfo)
	if err != nil {
		mylog.Printf("Error marshaling data: %v", err)
		//todo 符合onebotv11 ws返回的错误码
		return "", nil
	}
	return string(result), nil
}

// 将结构体转换为 map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	json.Unmarshal(j, &out)
	return out
}
