package handlers

import (
	"context"
	"log"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

// Member Onebot 群成员
type MemberList struct {
	UserID       string `json:"user_id"`
	GroupID      string `json:"group_id"`
	Nickname     string `json:"nickname"`
	Role         string `json:"role"`
	JoinTime     string `json:"join_time"`
	LastSentTime string `json:"last_sent_time"`
}

func init() {
	callapi.RegisterHandler("get_group_member_list", getGroupMemberList)
}

func getGroupMemberList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	msgType, err := idmap.ReadConfigv2(message.Params.GroupID.(string), "type")
	if err != nil {
		log.Printf("Error reading config: %v", err)
		return
	}

	switch msgType {
	case "group":
		log.Printf("getGroupMemberList(频道): 目前暂未开放该能力")
		return
	case "private":
		log.Printf("getGroupMemberList(频道): 目前暂未适配私聊虚拟群场景获取虚拟群列表能力")
		return
	case "guild":
		pager := &dto.GuildMembersPager{
			Limit: "400",
		}
		membersFromAPI, err := api.GuildMembers(context.TODO(), message.Params.GroupID.(string), pager)
		if err != nil {
			log.Printf("Failed to fetch group members for guild %s: %v", message.Params.GroupID.(string), err)
			return
		}

		var members []MemberList
		for _, memberFromAPI := range membersFromAPI {
			joinedAtTime, err := memberFromAPI.JoinedAt.Time()
			if err != nil {
				log.Println("Error parsing JoinedAt timestamp:", err)
				continue
			}
			joinedAtStr := joinedAtTime.Format(time.RFC3339) // or any other format

			member := MemberList{
				UserID:   memberFromAPI.User.ID,
				GroupID:  message.Params.GroupID.(string),
				Nickname: memberFromAPI.Nick,
				JoinTime: joinedAtStr,
			}
			for _, role := range memberFromAPI.Roles {
				switch role {
				case "4":
					member.Role = "owner"
				case "2":
					member.Role = "admin"
				case "11", "default":
					member.Role = "member"
				}
				if member.Role == "owner" || member.Role == "admin" {
					break
				}
			}
			members = append(members, member)
		}

		// Convert the APIOutput structure to a map[string]interface{}
		outputMap := structToMap(members)

		log.Printf("getGroupMemberList(频道): %+v\n", outputMap)

		err = client.SendMessage(outputMap) //发回去
		if err != nil {
			log.Printf("Error sending message via client: %v", err)
		}
	default:
		log.Printf("Unknown msgType: %s", msgType)
	}
}
