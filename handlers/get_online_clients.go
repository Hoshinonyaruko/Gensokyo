package handlers

import (
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/openapi"
)

type OnlineClientsResponse struct {
	Data    OnlineClientsData `json:"data"`
	Message string            `json:"message"`
	RetCode int               `json:"retcode"`
	Status  string            `json:"status"`
	Echo    interface{}       `json:"echo"`
}

type OnlineClientsData struct {
	Clients []interface{} `json:"clients"` // It seems you want an empty array for clients
	TinyID  int64         `json:"tiny_id"`
}

func init() {
	callapi.RegisterHandler("get_online_clients", getOnlineClients)
}

func getOnlineClients(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	var response OnlineClientsResponse

	response.Data = OnlineClientsData{
		Clients: make([]interface{}, 0), // Empty array
		TinyID:  0,
	}
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the members slice to a map
	outputMap := structToMap(response)

	log.Printf("get_online_clients: %+v\n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		log.Printf("Error sending message via client: %v", err)
	} else {
		log.Printf("响应get_online_clients: %+v", outputMap)
	}
}
