package handlers

import (
	"context"
	"encoding/json"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/mylog"
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
	callapi.RegisterHandler("get_guild_channel_list", GetGuildChannelList)
}

func GetGuildChannelList(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	var response GuildChannelListResponse

	// 解析请求参数
	guildID := message.Params.GuildID

	// 根据请求参数调用API
	channels, err := api.Channels(context.TODO(), guildID.(string))
	if err != nil {
		// 如果发生错误，记录日志并返回null
		mylog.Printf("Error fetching channels: %v", err)
		client.SendMessage(map[string]interface{}{"data": nil})
		return "", nil
	}

	// 构建响应数据
	for _, channel := range channels {
		channelInfo := map[string]interface{}{
			"owner_guild_id":    guildID,
			"channel_id":        channel.ID,
			"channel_type":      channel.Type,
			"channel_name":      channel.Name,
			"create_time":       0, // Default value as actual value is not available
			"creator_tiny_id":   channel.OwnerID,
			"talk_permission":   channel.Permissions,
			"visible_type":      channel.Position,
			"current_slow_mode": 0, // Default value as actual value is not available
		}

		// Append the channel information to the response data
		response.Data = append(response.Data, channelInfo)
	}

	// Set other fields of the response
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the response to a map for sending
	outputMap := structToMap(response)

	mylog.Printf("get_guild_channel_list: %s", outputMap)

	// Send the response
	err = client.SendMessage(outputMap)
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
