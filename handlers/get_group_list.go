package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_list", getGroupList)
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
	GroupCreateTime string `json:"group_create_time"`
	GroupID         string `json:"group_id"`
	GroupLevel      string `json:"group_level"`
	GroupMemo       string `json:"group_memo"`
	GroupName       string `json:"group_name"`
	MaxMemberCount  string `json:"max_member_count"`
	MemberCount     string `json:"member_count"`
}

type GroupList struct {
	Data    []Group     `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

func getGroupList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	//群还不支持,这里取得是频道的,如果后期支持了群,那都请求,一起返回
	actiontype := "guild"
	if actiontype == "guild" {
		// 检查时间差异
		if time.Since(lastCallTime) > 5*time.Minute {
			// 如果超过5分钟，则重置分页状态
			globalPager = &dto.GuildPager{Limit: "10"}
		}
		// 全局pager
		guilds, err := api.MeGuilds(context.TODO(), globalPager)
		if err != nil {
			mylog.Println("Error fetching guild list:", err)
			return
		}
		if len(guilds) > 0 {
			// 更新Pager的After为最后一个元素的ID
			globalPager.After = guilds[len(guilds)-1].ID
		}
		lastCallTime = time.Now() // 更新上次调用API的时间

		if len(guilds) == 0 {
			return
		}
		var groups []Group
		for _, guild := range guilds {
			joinedAtTime, err := guild.JoinedAt.Time()
			if err != nil {
				mylog.Println("Error parsing JoinedAt timestamp:", err)
				continue
			}
			joinedAtStr := joinedAtTime.Format(time.RFC3339) // or any other format you prefer
			group := Group{
				GroupCreateTime: joinedAtStr,
				GroupID:         guild.ID,
				GroupLevel:      guild.OwnerID,
				GroupMemo:       guild.Desc,
				GroupName:       "*" + guild.Name,
				MaxMemberCount:  strconv.FormatInt(guild.MaxMembers, 10),
				MemberCount:     strconv.Itoa(guild.MemberCount),
			}
			groups = append(groups, group)
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
					GroupCreateTime: "", // 频道没有直接对应的创建时间字段
					GroupID:         fmt.Sprint(ChannelID64),
					GroupLevel:      "", // 频道没有直接对应的级别字段
					GroupMemo:       "", // 频道没有直接对应的描述字段
					GroupName:       channel.Name,
					MaxMemberCount:  "", // 频道没有直接对应的最大成员数字段
					MemberCount:     "", // 频道没有直接对应的成员数字段
				}
				groups = append(groups, channelGroup)
			}
		}
		groupList := GroupList{
			Data:    groups,
			Message: "",
			RetCode: 0,
			Status:  "ok",
		}
		if message.Echo == "" {
			groupList.Echo = "0"
		} else {
			groupList.Echo = message.Echo

			outputMap := structToMap(groupList)

			mylog.Printf("getGroupList(频道): %+v\n", outputMap)

			err = client.SendMessage(outputMap)
			if err != nil {
				mylog.Printf("error sending group info via wsclient: %v", err)
			}

			result, err := json.Marshal(groupList)
			if err != nil {
				mylog.Printf("Error marshaling data: %v", err)
				return
			}

			mylog.Printf("get_group_list: %s", result)
		}

	} else {
		// todo 等待群 群列表api
	}

}
