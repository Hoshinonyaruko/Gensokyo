package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_list", getGroupList)
}

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

	// 初始化pager
	pager := &dto.GuildPager{
		Limit: "400",
	}

	guilds, err := api.MeGuilds(context.TODO(), pager)
	if err != nil {
		log.Println("Error fetching guild list:", err)
		// 创建虚拟的Group
		virtualGroup := Group{
			GroupCreateTime: time.Now().Format(time.RFC3339),
			GroupID:         "0000000", // 或其他虚拟值
			GroupLevel:      "0",
			GroupMemo:       "Error Fetching Guilds",
			GroupName:       "Error Guild",
			MaxMemberCount:  "0",
			MemberCount:     "0",
		}

		// 创建包含虚拟Group的GroupList
		groupList := GroupList{
			Data:    []Group{virtualGroup},
			Message: "Error fetching guilds",
			RetCode: -1, // 可以使用其他的错误代码
			Status:  "error",
			Echo:    "0",
		}

		if message.Echo == "" {
			groupList.Echo = "0"
		} else {
			groupList.Echo = message.Echo

			outputMap := structToMap(groupList)

			log.Printf("getGroupList(频道): %+v\n", outputMap)

			err = client.SendMessage(outputMap)
			if err != nil {
				log.Printf("error sending group info via wsclient: %v", err)
			}

			result, err := json.Marshal(groupList)
			if err != nil {
				log.Printf("Error marshaling data: %v", err)
				return
			}

			log.Printf("get_group_list: %s", result)
			return
		}
	}

	var groups []Group
	for _, guild := range guilds {
		joinedAtTime, err := guild.JoinedAt.Time()
		if err != nil {
			log.Println("Error parsing JoinedAt timestamp:", err)
			continue
		}
		joinedAtStr := joinedAtTime.Format(time.RFC3339) // or any other format you prefer

		group := Group{
			GroupCreateTime: joinedAtStr,
			GroupID:         guild.ID,
			GroupLevel:      guild.OwnerID,
			GroupMemo:       guild.Desc,
			GroupName:       guild.Name,
			MaxMemberCount:  strconv.FormatInt(guild.MaxMembers, 10),
			MemberCount:     strconv.Itoa(guild.MemberCount),
			// Add other fields if necessary
		}
		groups = append(groups, group)
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

		log.Printf("getGroupList(频道): %+v\n", outputMap)

		err = client.SendMessage(outputMap)
		if err != nil {
			log.Printf("error sending group info via wsclient: %v", err)
		}

		result, err := json.Marshal(groupList)
		if err != nil {
			log.Printf("Error marshaling data: %v", err)
			return
		}

		log.Printf("get_group_list: %s", result)
	}
}
