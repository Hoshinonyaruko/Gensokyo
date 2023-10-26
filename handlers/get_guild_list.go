package handlers

import (
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/openapi"
)

type GuildListResponse struct {
	Data    []GuildData `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

type GuildData struct {
	GuildID        string `json:"guild_id"`
	GuildName      string `json:"guild_name"`
	GuildDisplayID string `json:"guild_display_id"`
}

func init() {
	callapi.RegisterHandler("get_guild_list", getGuildList)
}

func getGuildList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	var response GuildListResponse

	// Assuming 'a' is some value you want to loop till.
	a := 1 // Replace with appropriate value

	for i := 1; i <= a; i++ {
		guildData := GuildData{
			GuildID:        "0",
			GuildName:      "868858989",
			GuildDisplayID: "868858989",
		}
		response.Data = append(response.Data, guildData)
	}

	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = string(message.Echo) // Directly assign the string value

	// Convert the members slice to a map
	outputMap := structToMap(response)

	log.Printf("getGuildList(频道): %+v\n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		log.Printf("Error sending message via client: %v", err)
	}
}
