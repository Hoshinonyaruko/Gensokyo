package handlers

import (
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"
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
	callapi.RegisterHandler("get_guild_service_profile", getGuildServiceProfile)
}

func getGuildServiceProfile(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

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

	log.Printf("get_guild_service_profile: %+v/n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		log.Printf("Error sending message via client: %v", err)
	} else {
		log.Printf("响应get_guild_service_profile: %+v", outputMap)
	}
}
