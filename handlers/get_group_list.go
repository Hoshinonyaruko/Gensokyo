package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_list", GetGroupList)
}

// 全局的Pager实例，用于保存状态
var (
	globalPager *dto.GuildPager = &dto.GuildPager{
		Limit: "10",
	}
	lastCallTime time.Time // 保存上次调用API的时间
)

type Guild struct {
	JoinedAt    string `json:"joined_at"`
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	MaxMembers  string `json:"max_members"`
	MemberCount string `json:"member_count"`
}

type Group struct {
	GroupCreateTime int32  `json:"group_create_time"`
	GroupID         int64  `json:"group_id"`
	GroupLevel      int32  `json:"group_level"`
	GroupMemo       string `json:"group_memo"`
	GroupName       string `json:"group_name"`
	MaxMemberCount  int32  `json:"max_member_count"`
	MemberCount     int32  `json:"member_count"`
}

type GroupList struct {
	Data    []Group     `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

func GetGroupList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	//群还不支持,这里取得是频道的,如果后期支持了群,那都请求,一起返回
	var groupList GroupList

	// 初始化 groupList.Data 为一个空数组
	groupList.Data = []Group{}
	// 检查时间差异
	if time.Since(lastCallTime) > 5*time.Minute {
		// 如果超过5分钟，则重置分页状态
		globalPager = &dto.GuildPager{Limit: "10"}
	}
	// 全局pager
	guilds, err := api.MeGuilds(context.TODO(), globalPager)
	if err != nil {
		mylog.Println("Error fetching guild list:", err)
		return "", nil
	}
	if len(guilds) > 0 {
		// 更新Pager的After为最后一个元素的ID
		globalPager.After = guilds[len(guilds)-1].ID
	}
	lastCallTime = time.Now() // 更新上次调用API的时间
	//如果为空 则不使用分页
	if len(guilds) == 0 {
		Pager := &dto.GuildPager{Limit: "10"}
		guilds, err = api.MeGuilds(context.TODO(), Pager)
		if err != nil {
			mylog.Println("Error fetching guild list2:", err)
			return "", nil
		}
	}
	for _, guild := range guilds {
		joinedAtTime, err := guild.JoinedAt.Time()
		if err != nil {
			mylog.Println("Error parsing JoinedAt timestamp:", err)
			continue
		}
		groupID, _ := strconv.ParseInt(guild.ID, 10, 64)
		joinedAtTimestamp := int32(joinedAtTime.Unix()) // 获取 10 位时间戳并转换为 int32 类型
		group := Group{
			GroupCreateTime: joinedAtTimestamp,
			GroupID:         groupID,
			GroupLevel:      0,
			GroupMemo:       guild.Desc,
			GroupName:       "*" + guild.Name,
			MaxMemberCount:  int32(guild.MaxMembers),  // 确保这里也是 int32 类型
			MemberCount:     int32(guild.MemberCount), // 将这里也转换为 int32 类型
		}
		groupList.Data = append(groupList.Data, group)
		// 获取每个guild的channel信息
		channels, err := api.Channels(context.TODO(), guild.ID) // 使用guild.ID作为参数
		if err != nil {
			mylog.Println("Error fetching channels list:", err)
			continue
		}
		// 将channel信息转换为Group对象并添加到groups
		for _, channel := range channels {
			//转换ChannelID64
			ChannelID64, err := idmap.StoreIDv2(channel.ID)
			if err != nil {
				mylog.Printf("Error storing ID: %v", err)
			}
			channelGroup := Group{
				GroupCreateTime: 0, // 频道没有直接对应的创建时间字段
				GroupID:         ChannelID64,
				GroupLevel:      0,  // 频道没有直接对应的级别字段
				GroupMemo:       "", // 频道没有直接对应的描述字段
				GroupName:       channel.Name,
				MaxMemberCount:  0, // 频道没有直接对应的最大成员数字段
				MemberCount:     0, // 频道没有直接对应的成员数字段
			}
			groupList.Data = append(groupList.Data, channelGroup)
		}
	}
	//从idmaps数据库找群,组合成群列表需要的格式
	groupIDs, err := idmap.FindKeysBySubAndType("group", "type")
	if err != nil {
		mylog.Printf("Error FindKeysBySubAndType %s", err)
	}
	// 当前时间的 10 位 Unix 时间戳
	currentTimestamp := int32(time.Now().Unix())

	for _, idStr := range groupIDs {
		groupID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			mylog.Printf("Error converting group ID %s to int64: %v", idStr, err)
			continue
		}

		group := Group{
			GroupCreateTime: currentTimestamp, // 使用当前时间的时间戳
			GroupID:         groupID,
			GroupLevel:      0,
			GroupMemo:       "",
			GroupName:       "",
			MaxMemberCount:  0,
			MemberCount:     0,
		}

		groupList.Data = append(groupList.Data, group)
	}
	groupList.Message = ""
	groupList.RetCode = 0
	groupList.Status = "ok"

	if message.Echo == "" {
		groupList.Echo = "0"
	} else {
		groupList.Echo = message.Echo
	}
	outputMap := structToMap(groupList)

	mylog.Printf("getGroupList(频道): %+v\n", outputMap)

	err = client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("error sending group info via wsclient: %v", err)
	}

	result, err := json.Marshal(groupList)
	if err != nil {
		mylog.Printf("Error marshaling data: %v", err)
		return "", nil
	}

	mylog.Printf("get_group_list: %s", result)
	return string(result), nil
}
