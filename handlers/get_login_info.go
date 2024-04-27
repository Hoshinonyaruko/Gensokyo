package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

type LoginInfoResponse struct {
	Data    LoginInfoData `json:"data"`
	Message string        `json:"message"`
	RetCode int           `json:"retcode"`
	Status  string        `json:"status"`
	Echo    interface{}   `json:"echo"`
}

type LoginInfoData struct {
	Nickname string `json:"nickname"`
	UserID   string `json:"user_id"` // Assuming UserID is a string type based on the pseudocode
}

func init() {
	callapi.RegisterHandler("get_login_info", GetLoginInfo)
}

func GetLoginInfo(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {

	var response LoginInfoResponse
	var botname string
	var globalBotID uint64

	// 获取机器人ID
	if config.GetUseUin() {
		globalBotID = uint64(config.GetUinint64())
	} else {
		globalBotID = config.GetAppID()
	}

	userIDStr := fmt.Sprintf("%d", globalBotID)
	botname = config.GetCustomBotName()

	response.Data = LoginInfoData{
		Nickname: botname,
		UserID:   userIDStr,
	}
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the members slice to a map
	outputMap := structToMap(response)

	mylog.Printf("get_login_info: %+v\n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
	} else {
		mylog.Printf("响应get_login_info: %+v", outputMap)
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
