package handlers

import (
	"fmt"
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
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
	callapi.RegisterHandler("get_login_info", getLoginInfo)
}

func getLoginInfo(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	var response LoginInfoResponse

	// Assuming 全局_botid is a global or environment variable
	globalBotID := config.GetAppID() // Replace with the actual global variable or value
	userIDStr := fmt.Sprintf("%d", globalBotID)

	response.Data = LoginInfoData{
		Nickname: "gensokyo全域机器人",
		UserID:   userIDStr,
	}
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the members slice to a map
	outputMap := structToMap(response)

	log.Printf("get_login_info: %+v\n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		log.Printf("Error sending message via client: %v", err)
	} else {
		log.Printf("响应get_login_info: %+v", outputMap)
	}

	return
}
