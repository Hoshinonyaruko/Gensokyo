package handlers

import (
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/openapi"
)

type GuildChannelListResponse struct {
	Data    []interface{} `json:"data"`
	Message string        `json:"message"`
	RetCode int           `json:"retcode"`
	Status  string        `json:"status"`
	Echo    interface{}   `json:"echo"`
}

func init() {
	callapi.RegisterHandler("get_guild_channel_list", getGuildChannelList)
}

func getGuildChannelList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	var response GuildChannelListResponse

	response.Data = make([]interface{}, 0) // No data at the moment, but can be populated in the future
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the members slice to a map
	outputMap := structToMap(response)

	log.Printf("get_guild_channel_list: %s", outputMap)

	err := client.SendMessage(outputMap) //发回去
	if err != nil {
		log.Printf("Error sending message via client: %v", err)
	}
}
