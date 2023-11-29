package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
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
		mylog.Printf("Error reading config: %v", err)
		return
	}

	switch msgType {
	case "group":
		mylog.Printf("getGroupMemberList(group): 开始从本地获取群成员列表")
		// 实现的功能
		var members []MemberList

		// 使用 message.Params.GroupID.(string) 作为 id 来调用 FindSubKeysById
		userIDs, err := idmap.FindSubKeysById(message.Params.GroupID.(string))
		if err != nil {
			mylog.Printf("Error retrieving user IDs: %v", err)
			return // 或者处理错误
		}

		// 获取当前时间的前一天，并转换为10位时间戳
		yesterday := time.Now().AddDate(0, 0, -1).Unix()

		for _, userID := range userIDs {
			member := MemberList{
				UserID:   userID,
				GroupID:  message.Params.GroupID.(string),
				Nickname: "", // 根据需要填充或保留为空
				JoinTime: strconv.FormatInt(yesterday, 10),
				Role:     "admin", // 默认权限，可以根据需要修改
			}

			members = append(members, member)
		}
		mylog.Printf("member message.Echors: %+v\n", message.Echo)

		responseJSON := buildResponse(members, message.Echo)
		mylog.Printf("getGroupMemberList(群): %s\n", responseJSON)

		err = client.SendMessage(responseJSON)
		if err != nil {
			mylog.Printf("Error sending message via client: %v", err)
		}
		return
	case "private":
		mylog.Printf("getGroupMemberList(private): 目前暂未适配私聊虚拟群场景获取虚拟群列表能力")
		return
	case "guild":
		//要把group_id还原成guild_id
		//用group_id还原出channelid 这是虚拟成群的私聊信息
		message.Params.ChannelID = message.Params.GroupID.(string)
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err := idmap.RetrieveRowByIDv2(message.Params.ChannelID)
		if err != nil {
			mylog.Printf("error retrieving real ChannelID: %v", err)
		}
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(RChannelID, "guild_id")
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return
		}
		pager := &dto.GuildMembersPager{
			Limit: "400",
		}
		membersFromAPI, err := api.GuildMembers(context.TODO(), value, pager)
		if err != nil {
			mylog.Printf("Failed to fetch group members for guild %s: %v", value, err)
			return
		}

		// mylog.Println("Number of members in membersFromAPI:", len(membersFromAPI))
		// for i, member := range membersFromAPI {
		// 	mylog.Printf("Member %d: %+v\n", i+1, *member)
		// }

		var members []MemberList
		for _, memberFromAPI := range membersFromAPI {
			joinedAtTime, err := memberFromAPI.JoinedAt.Time()
			if err != nil {
				mylog.Println("Error parsing JoinedAt timestamp:", err)
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

		mylog.Printf("member message.Echors: %+v\n", message.Echo)

		responseJSON := buildResponse(members, message.Echo) // assume echoValue is your echo data
		mylog.Printf("getGroupMemberList(频道): %s\n", responseJSON)

		err = client.SendMessage(responseJSON) //发回去
		if err != nil {
			mylog.Printf("Error sending message via client: %v", err)
		}
	default:
		mylog.Printf("Unknown msgType: %s", msgType)
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
		mylog.Printf("Setting echo as int: %d", v)
		response["echo"] = v
	case string:
		mylog.Printf("Setting echo as string: %s", v)
		response["echo"] = v
	case []interface{}:
		mylog.Printf("Setting echo as array: %v", v)
		response["echo"] = v
	case map[string]interface{}:
		mylog.Printf("Setting echo as object: %v", v)
		response["echo"] = v
	default:
		mylog.Printf("Unknown type for echo: %T", v)
	}

	return response
}
