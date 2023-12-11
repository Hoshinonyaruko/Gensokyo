package handlers

import (
	"encoding/json"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_friend_list", HandleGetFriendList)
}

type APIOutput struct {
	Data    []FriendData `json:"data"`
	Message string       `json:"message"`
	RetCode int          `json:"retcode"`
	Status  string       `json:"status"`
	Echo    interface{}  `json:"echo"`
}

type FriendData struct {
	Nickname string `json:"nickname"`
	Remark   string `json:"remark"`
	UserID   string `json:"user_id"`
}

func HandleGetFriendList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	var output APIOutput

	for i := 0; i < 10; i++ { // Assume we want to loop 10 times to create friend data
		data := FriendData{
			Nickname: "小狐狸",
			Remark:   "",
			UserID:   "2022717137",
		}
		output.Data = append(output.Data, data)
	}

	output.Message = ""
	output.RetCode = 0
	output.Status = "ok"

	output.Echo = message.Echo

	// Convert the APIOutput structure to a map[string]interface{}
	outputMap := structToMap(output)

	// Send the map
	err := client.SendMessage(outputMap) //发回去
	if err != nil {
		mylog.Printf("error sending friend list via wsclient: %v", err)
	}

	result, err := json.Marshal(output)
	if err != nil {
		mylog.Printf("Error marshaling data: %v", err)
		//todo 符合onebotv11 ws返回的错误码
		return "", nil
	}

	mylog.Printf("get_friend_list: %s", result)
	return string(result), nil
}
