package handlers

import (
	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tidwall/gjson"
)

type MarkThisMessageAsReadAPIResponse struct {
	Status  string       `json:"status"`
	Data    gjson.Result `json:"data"`
	Msg     string       `json:"msg"`
	Wording string       `json:"wording"`
	RetCode int64        `json:"retcode"`
	Echo    interface{}  `json:"echo"`
}

func init() {
	callapi.RegisterHandler("mark_msg_as_read", MarkThisMessageAsRead)
}

func MarkThisMessageAsRead(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	var response MarkThisMessageAsReadAPIResponse

	response.Data.Str = "123"
	response.Msg = "123"
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the members slice to a map
	outputMap := structToMap(response)

	mylog.Printf("mark_msg_as_read: %+v\n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
	} else {
		mylog.Printf("响应mark_msg_as_read: %+v", outputMap)
	}

}
