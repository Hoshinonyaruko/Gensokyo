package handlers

import (
	"encoding/json"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

type GetStatusResponse struct {
	Data    StatusData  `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

type StatusData struct {
	AppInitialized bool       `json:"app_initialized"`
	AppEnabled     bool       `json:"app_enabled"`
	PluginsGood    bool       `json:"plugins_good"`
	AppGood        bool       `json:"app_good"`
	Online         bool       `json:"online"`
	Good           bool       `json:"good"`
	Stat           Statistics `json:"stat"`
}

type Statistics struct {
	PacketReceived  uint64 `json:"packet_received"`
	PacketSent      uint64 `json:"packet_sent"`
	PacketLost      uint32 `json:"packet_lost"`
	MessageReceived uint64 `json:"message_received"`
	MessageSent     uint64 `json:"message_sent"`
	DisconnectTimes uint32 `json:"disconnect_times"`
	LostTimes       uint32 `json:"lost_times"`
	LastMessageTime int64  `json:"last_message_time"`
}

func init() {
	callapi.RegisterHandler("get_status", GetStatus)
}

func GetStatus(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {

	var response GetStatusResponse

	response.Data = StatusData{
		AppInitialized: true,
		AppEnabled:     true,
		PluginsGood:    true,
		AppGood:        true,
		Online:         true, //测试数据
		Good:           true, //测试数据
		Stat: Statistics{
			PacketReceived:  1000,       //测试数据
			PacketSent:      950,        //测试数据
			PacketLost:      50,         //测试数据
			MessageReceived: 500,        //测试数据
			MessageSent:     490,        //测试数据
			DisconnectTimes: 5,          //测试数据
			LostTimes:       2,          //测试数据
			LastMessageTime: 1677721600, //测试数据
		},
	}
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	outputMap := structToMap(response)

	mylog.Printf("get_status: %+v\n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
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
