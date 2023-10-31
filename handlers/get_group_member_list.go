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

type Response struct {
	Retcode int          `json:"retcode"`
	Status  string       `json:"status"`
	Data    []MemberList `json:"data"`
	Echo    interface{}  `json:"echo"` // 使用 interface{} 类型以容纳整数或文本
}

// Member Onebot 群成员
type MemberList struct {
	UserID       string `json:"user_id"`
	GroupID      string `json:"group_id"`
	Nickname     string `json:"nickname"`
	Role         string `json:"role"`
	JoinTime     string `json:"join_time"`
	LastSentTime string `json:"last_sent_time"`
	Level        string `json:"level,omitempty"` // 我添加了 Level 字段，因为你的示例中有它，但你可以删除它，如果不需要
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
		//要把group_id还原成guild_id
		//用group_id还原出channelid 这是虚拟成群的私聊信息
		message.Params.ChannelID = message.Params.GroupID.(string)
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(message.Params.ChannelID, "guild_id")
		if err != nil {
			log.Printf("Error reading config: %v", err)
			return
		}
		pager := &dto.GuildMembersPager{
			Limit: "400",
		}
		membersFromAPI, err := api.GuildMembers(context.TODO(), value, pager)
		if err != nil {
			log.Printf("Failed to fetch group members for guild %s: %v", value, err)
			return
		}

		// log.Println("Number of members in membersFromAPI:", len(membersFromAPI))
		// for i, member := range membersFromAPI {
		// 	log.Printf("Member %d: %+v\n", i+1, *member)
		// }

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

		log.Printf("member message.Echors: %+v\n", message.Echo)

		responseJSON := buildResponse(members, message.Echo) // assume echoValue is your echo data
		log.Printf("getGroupMemberList(频道): %s\n", responseJSON)

		err = client.SendMessage(responseJSON) //发回去
		if err != nil {
			log.Printf("Error sending message via client: %v", err)
		}
	default:
		log.Printf("Unknown msgType: %s", msgType)
	}
}

func buildResponse(members []MemberList, echoValue interface{}) map[string]interface{} {
	data := make([]map[string]interface{}, len(members))

	for i, member := range members {
		memberMap := map[string]interface{}{
			"user_id":        member.UserID,
			"group_id":       member.GroupID,
			"nickname":       member.Nickname,
			"role":           member.Role,
			"join_time":      member.JoinTime,
			"last_sent_time": member.LastSentTime,
		}
		data[i] = memberMap
	}

	response := map[string]interface{}{
		"retcode": 0,
		"status":  "ok",
		"data":    data,
	}

	// Set echo based on the type of echoValue
	switch v := echoValue.(type) {
	case int:
		log.Printf("Setting echo as int: %d", v)
		response["echo"] = v
	case string:
		log.Printf("Setting echo as string: %s", v)
		response["echo"] = v
	case []interface{}:
		log.Printf("Setting echo as array: %v", v)
		response["echo"] = v
	case map[string]interface{}:
		log.Printf("Setting echo as object: %v", v)
		response["echo"] = v
	default:
		log.Printf("Unknown type for echo: %T", v)
	}

	return response
}
