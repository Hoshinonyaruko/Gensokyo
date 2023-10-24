package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"

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

func ConvertGuildToGroupInfo(guild *dto.Guild) *OnebotGroupInfo {
	groupID, err := strconv.ParseInt(guild.ID, 10, 64)
	if err != nil {
		log.Printf("转换ID失败: %v", err)
		return nil
	}

	ts, err := guild.JoinedAt.Time()
	if err != nil {
		log.Printf("转换JoinedAt失败: %v", err)
		return nil
	}
	groupCreateTime := uint32(ts.Unix())

	return &OnebotGroupInfo{
		GroupID:         groupID,
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
	//读取ini 通过ChannelID取回之前储存的guild_id
	value, err := idmap.ReadConfig(ChannelID.(string), "guild_id")
	if err != nil {
		log.Printf("handleGetGroupInfo:Error reading config: %v\n", err)
		return
	}
	//最后获取到guildID
	guildID := value
	log.Printf("调试,准备groupInfoMap(频道)guildID:%v", guildID)
	guild, err := api.Guild(context.TODO(), guildID)
	if err != nil {
		log.Printf("获取频道信息失败: %v", err)
		return
	}

	groupInfo := ConvertGuildToGroupInfo(guild)

	groupInfoMap := structToMap(groupInfo)

	// 打印groupInfoMap的内容
	log.Printf("groupInfoMap(频道): %+v/n", groupInfoMap)

	err = client.SendMessage(groupInfoMap) //发回去
	if err != nil {
		log.Printf("error sending group info via wsclient: %v", err)
	}
}

// 将结构体转换为 map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	json.Unmarshal(j, &out)
	return out
}
