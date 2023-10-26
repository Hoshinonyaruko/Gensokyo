package handlers

import (
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/tencent-connect/botgo/openapi"
)

type VersionInfoResponse struct {
	Data    VersionData `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

type VersionData struct {
	AppFullName              string `json:"app_full_name"`
	AppName                  string `json:"app_name"`
	AppVersion               string `json:"app_version"`
	CoolQDirectory           string `json:"coolq_directory"`
	CoolQEdition             string `json:"coolq_edition"`
	GoCQHTTP                 bool   `json:"go-cqhttp"`
	PluginBuildConfiguration string `json:"plugin_build_configuration"`
	PluginBuildNumber        int    `json:"plugin_build_number"`
	PluginVersion            string `json:"plugin_version"`
	ProtocolName             int    `json:"protocol_name"`
	ProtocolVersion          string `json:"protocol_version"`
	RuntimeOS                string `json:"runtime_os"`
	RuntimeVersion           string `json:"runtime_version"`
	Version                  string `json:"version"`
}

func init() {
	callapi.RegisterHandler("get_version_info", getVersionInfo)
}

func getVersionInfo(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	var response VersionInfoResponse

	response.Data = VersionData{
		AppFullName:              "gensokyo",
		AppName:                  "gensokyo",
		AppVersion:               "v1.0.0",
		CoolQDirectory:           "",
		CoolQEdition:             "pro",
		GoCQHTTP:                 true,
		PluginBuildConfiguration: "release",
		PluginBuildNumber:        99,
		PluginVersion:            "4.15.0",
		ProtocolName:             4,
		ProtocolVersion:          "v11",
		RuntimeOS:                "windows",
		RuntimeVersion:           "go1.20.2",
		Version:                  "v1.0.0",
	}
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = string(message.Echo) // Directly assign the string value

	// Convert the members slice to a map
	outputMap := structToMap(response)

	log.Printf("get_version_info: %+v/n", outputMap)

	err := client.SendMessage(outputMap)
	if err != nil {
		log.Printf("Error sending message via client: %v", err)
	}
}
