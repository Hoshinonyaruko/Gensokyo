package handlers

import (
	"encoding/json"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

type GuildServiceProfileResponse struct {
	Data    GuildServiceProfileData `json:"data"`
	Message string                  `json:"message"`
	RetCode int                     `json:"retcode"`
	Status  string                  `json:"status"`
	Echo    interface{}             `json:"echo"`
}

type GuildServiceProfileData struct {
	Nickname string `json:"nickname"`
	TinyID   int64  `json:"tiny_id"`
}

func init() {
	callapi.RegisterHandler("get_guild_service_profile", GetGuildServiceProfile)
}

func GetGuildServiceProfile(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {

	var response GuildServiceProfileResponse

	response.Data = GuildServiceProfileData{
		Nickname: "",
		TinyID:   0,
	}
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the members slice to a map
	outputMap := structToMap(response)

	mylog.Printf("get_guild_service_profile: %+v/n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
	} else {
		mylog.Printf("响应get_guild_service_profile: %+v", outputMap)
	}
	//把结果从struct转换为json
	result, err := json.Marshal(response)
	if err != nil {
		mylog.Printf("Error marshaling data: %v", err)
		//todo 符合onebotv11 ws返回的错误码
		return "", nil
	}
	return string(result), nil
}
