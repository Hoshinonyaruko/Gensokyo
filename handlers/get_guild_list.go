package handlers

import (
	"context"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
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

	// 创建一个 GuildPager 实例，设置 limit 为 10
	pager := dto.GuildPager{Limit: "50", After: "0"} // 默认从0开始，取50个

	// 调用 API 获取群组列表
	guilds, err := api.MeGuilds(context.Background(), &pager)
	if err != nil {
		mylog.Printf("Error fetching guilds: %v", err)
		return
	}

	// 将获取的群组数据添加到 response 中
	for _, guild := range guilds {
		guildData := GuildData{
			GuildID:        guild.ID,
			GuildName:      guild.Name,
			GuildDisplayID: guild.ID, // 或其他合适的字段
			// ... 其他需要的字段
		}
		response.Data = append(response.Data, guildData)
	}

	// 设置 response 的其他属性
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the response to a map (if needed)
	outputMap := structToMap(response)

	mylog.Printf("getGuildList(频道): %+v\n", outputMap)

	// 发送消息
	err = client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
	}
}
