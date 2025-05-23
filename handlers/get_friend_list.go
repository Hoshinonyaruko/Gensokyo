package handlers

import (
	"encoding/json"
	"regexp"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/hoshinonyaruko/gensokyo/structs"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_friend_list", HandleGetFriendList)
}

type APIOutput struct {
	Data    []structs.FriendData `json:"data"`
	Message string               `json:"message"`
	RetCode int                  `json:"retcode"`
	Status  string               `json:"status"`
	Echo    interface{}          `json:"echo"`
}

func HandleGetFriendList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	var output APIOutput

	// 检查字符串是否是数字
	isNumeric := func(s string) bool {
		return regexp.MustCompile(`^\d+$`).MatchString(s)
	}

	// 从数据库获取所有用户信息
	users, err := idmap.ListAllUsers()
	if err != nil {
		mylog.Errorf("Failed to list users: %v", err)
	}

	// 过滤用户ID是数字的用户
	filteredUsers := []structs.FriendData{} // 假设 User 是你用户结构体的类型
	for _, user := range users {
		if !isNumeric(user.UserID) {
			filteredUsers = append(filteredUsers, user)
		}
	}

	// 添加数据库中读取的用户数据到output.Data
	output.Data = append(output.Data, filteredUsers...)

	output.Message = ""
	output.RetCode = 0
	output.Status = "ok"

	output.Echo = message.Echo

	// Convert the APIOutput structure to a map[string]interface{}
	outputMap := structToMap(output)

	// Send the map
	err = client.SendMessage(outputMap) //发回去
	if err != nil {
		mylog.Printf("error sending friend list via wsclient: %v", err)
	}

	result, err := json.Marshal(output)
	if err != nil {
		mylog.Printf("Error marshaling data: %v", err)
		//todo 符合onebotv11 ws返回的错误码
		return "", nil
	}

	//mylog.Printf("get_friend_list: %s", result)
	return string(result), nil
}
