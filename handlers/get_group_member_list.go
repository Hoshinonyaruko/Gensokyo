package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
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
	GroupID         uint64 `json:"group_id"`
	UserID          uint64 `json:"user_id"`
	Nickname        string `json:"nickname"`
	Card            string `json:"card"`
	Sex             string `json:"sex"`
	Age             int32  `json:"age"`
	Area            string `json:"area"`
	JoinTime        int32  `json:"join_time"`
	LastSentTime    int32  `json:"last_sent_time"`
	Level           string `json:"level"`
	Role            string `json:"role"`
	Unfriendly      bool   `json:"unfriendly"`
	Title           string `json:"title"`
	TitleExpireTime int64  `json:"title_expire_time"`
	CardChangeable  bool   `json:"card_changeable"`
	ShutUpTimestamp int64  `json:"shut_up_timestamp"`
}

func init() {
	callapi.RegisterHandler("get_group_member_list", GetGroupMemberList)
}

func GetGroupMemberList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {

	msgType, err := idmap.ReadConfigv2(message.Params.GroupID.(string), "type")
	if err != nil {
		mylog.Printf("Error reading config: %v", err)
		return "", nil
	}

	switch msgType {
	case "group":
		mylog.Printf("getGroupMemberList(group): 开始从本地获取群成员列表(请在config打开idmap-pro以缓存群成员列表)")
		// 实现的功能
		var members []MemberList

		// 使用 message.Params.GroupID.(string) 作为 id 来调用 FindSubKeysById
		userIDs, err := idmap.FindSubKeysByIdPro(message.Params.GroupID.(string))
		if err != nil {
			mylog.Printf("Error retrieving user IDs: %v", err)
			return "", nil // 或者处理错误
		}

		// 获取当前时间的前一天，并转换为10位时间戳
		yesterday := time.Now().AddDate(0, 0, -1).Unix()

		for _, userID := range userIDs {
			userIDInt, err := strconv.ParseUint(userID, 10, 64)
			if err != nil {
				mylog.Printf("Error ParseInt73: %v", err)
			}
			groupIDInt, err := strconv.ParseUint(message.Params.GroupID.(string), 10, 64)
			if err != nil {
				mylog.Printf("Error ParseInt76: %v", err)
			}
			joinTimeInt := int32(yesterday)
			member := MemberList{
				UserID:          userIDInt,
				GroupID:         groupIDInt,
				Nickname:        "主人",
				Card:            "主人",
				Sex:             "0",
				Age:             0,
				Area:            "0",
				JoinTime:        joinTimeInt,
				LastSentTime:    0,
				Level:           "0",
				Role:            "member",
				Unfriendly:      false,
				Title:           "0",
				TitleExpireTime: 0,
				CardChangeable:  false,
				ShutUpTimestamp: 0,
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
		result, err := ConvertMapToJSONString(responseJSON)
		if err != nil {
			mylog.Printf("Error marshaling data: %v", err)
			//todo 符合onebotv11 ws返回的错误码
			return "", nil
		}
		return string(result), nil
	case "private":
		mylog.Printf("getGroupMemberList(private): 目前暂未适配私聊虚拟群场景获取虚拟群列表能力")
		return "", nil
	case "guild":
		//要把group_id还原成guild_id
		//用group_id还原出channelid 这是虚拟成群的私聊信息
		message.Params.ChannelID = message.Params.GroupID.(string)
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err := idmap.RetrieveRowByIDv2(message.Params.ChannelID.(string))
		if err != nil {
			mylog.Printf("error retrieving real ChannelID: %v", err)
		}
		//读取ini 通过ChannelID取回之前储存的guild_id
		value, err := idmap.ReadConfigv2(RChannelID, "guild_id")
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return "", nil
		}
		pager := &dto.GuildMembersPager{
			Limit: "400",
		}
		membersFromAPI, err := api.GuildMembers(context.TODO(), value, pager)
		if err != nil {
			mylog.Printf("Failed to fetch group members for guild %s: %v", value, err)
		}
		// 检查是否是 11253 错误
		if err != nil && strings.Contains(err.Error(), `"code":11253`) {
			mylog.Printf("getGroupMemberList(guild): 开始从本地获取频道成员列表(请在config打开idmap-pro以缓存频道成员列表)")
			// 实现的功能
			var members []MemberList
			var userIDInt, groupIDInt uint64
			// 使用 message.Params.ChannelID  作为 id 来调用 FindSubKeysById
			userIDs, err := idmap.FindSubKeysByIdPro(message.Params.ChannelID.(string))
			if err != nil {
				mylog.Printf("Error retrieving user IDs: %v", err)
				return "", nil // 或者处理错误
			}
			mylog.Printf("返回的userIDs:%v", userIDs)
			// 获取当前时间的前一天，并转换为10位时间戳
			yesterday := time.Now().AddDate(0, 0, -1).Unix()

			for _, userID := range userIDs {
				if config.GetTransFormApiIds() {
					userIDInt, err = strconv.ParseUint(userID, 10, 64)
					if err != nil {
						mylog.Printf("Error ParseInt162: %v", err)
					}
					groupIDInt, err = strconv.ParseUint(message.Params.GroupID.(string), 10, 64)
					if err != nil {
						mylog.Printf("Error ParseInt166: %v", err)
					}
				} else {
					// 使用RetrieveRowByIDv2还原真实的Userid
					RuserIDStr, err := idmap.RetrieveRowByIDv2(userID)
					if err != nil {
						mylog.Printf("测试,通过idmap.RetrieveRowByIDv2获取RuserIDStr出错173:%v", err)
					}
					userIDInt, err = strconv.ParseUint(RuserIDStr, 10, 64)
					if err != nil {
						mylog.Printf("测试,通过idmap.RetrieveRowByIDv2获取的RChannelID出错177:%v", err)
					}
					RGroupidStr, err := idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
					if err != nil {
						mylog.Printf("测试,通过idmap.RetrieveRowByIDv2获取的RGroupidStr出错181:%v", err)
					}
					// 使用RetrieveRowByIDv2还原真实的ChannelID
					groupIDInt, err = strconv.ParseUint(RGroupidStr, 10, 64)
					if err != nil {
						mylog.Printf("测试,通过idmap.RetrieveRowByIDv2获取的RChannelID出错241:%v", err)
					}
				}

				joinTimeInt := int32(yesterday)
				member := MemberList{
					UserID:          userIDInt,
					GroupID:         groupIDInt,
					Nickname:        "主人",
					Card:            "主人",
					Sex:             "0",
					Age:             0,
					Area:            "0",
					JoinTime:        joinTimeInt,
					LastSentTime:    0,
					Level:           "0",
					Role:            "member",
					Unfriendly:      false,
					Title:           "0",
					TitleExpireTime: 0,
					CardChangeable:  false,
					ShutUpTimestamp: 0,
				}
				members = append(members, member)
			}
			mylog.Printf("member message.Echors: %+v\n", message.Echo)

			responseJSON := buildResponse(members, message.Echo)
			mylog.Printf("getGroupMemberList(频道): %s\n", responseJSON)

			err = client.SendMessage(responseJSON)
			if err != nil {
				mylog.Printf("Error sending message via client: %v", err)
			}
			result, err := ConvertMapToJSONString(responseJSON)
			if err != nil {
				mylog.Printf("Error marshaling data: %v", err)
				//todo 符合onebotv11 ws返回的错误码
				return "", nil
			}
			return string(result), nil
		}

		// mylog.Println("Number of members in membersFromAPI:", len(membersFromAPI))
		// for i, member := range membersFromAPI {
		// 	mylog.Printf("Member %d: %+v\n", i+1, *member)
		// }

		var members []MemberList
		var userIDUInt uint64
		var userIDInt64 int64
		var groupIDInt int64
		for _, memberFromAPI := range membersFromAPI {
			joinedAtTime, err := memberFromAPI.JoinedAt.Time()
			if err != nil {
				mylog.Println("Error parsing JoinedAt timestamp:", err)
				continue
			}
			userIDUInt = 0
			//判断是否转换api返回值的id
			if config.GetTransFormApiIds() {
				//调用的群号本身就是转换后的,不需要重复转换
				groupIDInt, err = strconv.ParseInt(message.Params.GroupID.(string), 10, 64)
				if err != nil {
					mylog.Printf("Error ParseUInt152: %v", err)
				}
				//判断是否开启idmap-pro转换
				if config.GetIdmapPro() {
					//用GroupID给ChannelID赋值,因为是把频道虚拟成了群
					message.Params.ChannelID = message.Params.GroupID.(string)
					//将真实id转为int userid64
					_, userIDInt64, err = idmap.StoreIDv2Pro(message.Params.ChannelID.(string), memberFromAPI.User.ID)
					if err != nil {
						mylog.Errorf("Error storing ID: %v", err)
					}
				} else {
					//映射str的userid到int
					userIDInt64, err = idmap.StoreIDv2(memberFromAPI.User.ID)
					if err != nil {
						mylog.Printf("Error storing ID 2400: %v", err)
						return "", nil
					}
				}
			} else {
				//原始值
				userIDUInt, err = strconv.ParseUint(memberFromAPI.User.ID, 10, 64)
				if err != nil {
					mylog.Printf("Error ParseUInt152: %v", err)
				}
				//用GroupID给ChannelID赋值,因为是把频道虚拟成了群
				message.Params.ChannelID = message.Params.GroupID.(string)
				var RChannelID string
				//根据api调用中的参数,还原真实的频道号
				if memberFromAPI.User.ID != "" && config.GetIdmapPro() {
					RChannelID, _, err = idmap.RetrieveRowByIDv2Pro(message.Params.ChannelID.(string), memberFromAPI.User.ID)
					if err != nil {
						mylog.Printf("测试,通过Proid获取的RChannelID出错232:%v", err)
					}
				}
				if RChannelID == "" {
					// 使用RetrieveRowByIDv2还原真实的ChannelID
					RChannelID, err = idmap.RetrieveRowByIDv2(message.Params.ChannelID.(string))
					if err != nil {
						mylog.Printf("测试,通过idmap.RetrieveRowByIDv2获取的RChannelID出错241:%v", err)
					}
				}
				groupIDInt, err = strconv.ParseInt(RChannelID, 10, 64)
				if err != nil {
					mylog.Printf("Error ParseInt156: %v", err)
				}
			}
			if userIDUInt == 0 {
				userIDUInt = uint64(userIDInt64)
			}
			joinTimeInt := int32(joinedAtTime.Unix())
			member := MemberList{
				UserID:          userIDUInt,
				GroupID:         uint64(groupIDInt),
				Nickname:        memberFromAPI.Nick,
				Card:            memberFromAPI.Nick, // 使用昵称作为默认值(TODO: 将来可能发生变更)
				Sex:             "0",                // 使用默认值
				Age:             0,                  // 使用默认值
				Area:            "0",                // 使用默认值
				JoinTime:        joinTimeInt,
				LastSentTime:    0,        // 使用默认值
				Level:           "0",      // 0
				Role:            "member", //
				Unfriendly:      false,
				Title:           "0", // 使用默认值
				TitleExpireTime: 0,   // 使用默认值
				CardChangeable:  false,
				ShutUpTimestamp: 0, // 使用默认值
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
		result, err := ConvertMapToJSONString(responseJSON)
		if err != nil {
			mylog.Printf("Error marshaling data: %v", err)
			//todo 符合onebotv11 ws返回的错误码
			return "", nil
		}
		return string(result), nil
	default:
		mylog.Printf("Unknown msgType: %s", msgType)
	}
	return "", nil
}

func buildResponse(members []MemberList, echoValue interface{}) map[string]interface{} {
	data := make([]map[string]interface{}, len(members))

	for i, member := range members {
		memberMap := map[string]interface{}{
			"user_id":           member.UserID,
			"group_id":          member.GroupID,
			"nickname":          member.Nickname,
			"card":              member.Card,
			"sex":               member.Sex,
			"age":               member.Age,
			"area":              member.Area,
			"join_time":         member.JoinTime,
			"last_sent_time":    member.LastSentTime,
			"level":             member.Level,
			"role":              member.Role,
			"unfriendly":        member.Unfriendly,
			"title":             member.Title,
			"title_expire_time": member.TitleExpireTime,
			"card_changeable":   member.CardChangeable,
			"shut_up_timestamp": member.ShutUpTimestamp,
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
