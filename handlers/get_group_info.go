package handlers

import (
	"context"
	"encoding/json"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_info", handleGetGroupInfo)
}

type OnebotGroupInfo struct {
	GroupID         int64  `json:"group_id"`
	GroupName       string `json:"group_name"`
	GroupMemo       string `json:"group_memo"`
	GroupCreateTime uint32 `json:"group_create_time"`
	GroupLevel      uint32 `json:"group_level"`
	MemberCount     int32  `json:"member_count"`
	MaxMemberCount  int32  `json:"max_member_count"`
}

func ConvertGuildToGroupInfo(guild *dto.Guild, GroupId string) *OnebotGroupInfo {
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
	groupCreateTime := uint32(ts.Unix())

	return &OnebotGroupInfo{
		GroupID:         groupid64,
		GroupName:       guild.Name,
		GroupMemo:       guild.Desc,
		GroupCreateTime: groupCreateTime,
		GroupLevel:      0,
		MemberCount:     int32(guild.MemberCount),
		MaxMemberCount:  int32(guild.MaxMembers),
	}
}

func handleGetGroupInfo(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	params := message.Params

	//用GroupID给ChannelID赋值,因为我们是把频道虚拟成了群
	ChannelID := params.GroupID
	// 使用RetrieveRowByIDv2还原真实的ChannelID
	RChannelID, err := idmap.RetrieveRowByIDv2(ChannelID.(string))
	if err != nil {
		mylog.Printf("error retrieving real ChannelID: %v", err)
	}
	//读取ini 通过ChannelID取回之前储存的guild_id
	value, err := idmap.ReadConfigv2(RChannelID, "guild_id")
	if err != nil {
		mylog.Printf("handleGetGroupInfo:Error reading config: %v\n", err)
		return
	}
	//最后获取到guildID
	guildID := value
	mylog.Printf("调试,准备groupInfoMap(频道)guildID:%v", guildID)
	guild, err := api.Guild(context.TODO(), guildID)
	if err != nil {
		mylog.Printf("获取频道信息失败: %v", err)
		return
	}

	groupInfo := ConvertGuildToGroupInfo(guild, guildID)
	groupInfoMap := structToMap(groupInfo)

	// 打印groupInfoMap的内容
	mylog.Printf("groupInfoMap(频道): %+v\n", groupInfoMap)

	err = client.SendMessage(groupInfoMap) //发回去
	if err != nil {
		mylog.Printf("error sending group info via wsclient: %v", err)
	}
}

// 将结构体转换为 map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	json.Unmarshal(j, &out)
	return out
}
