package handlers

import (
	"context"
	"log"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_whole_ban", setGroupBan)
}

func setGroupWholeBan(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {
	// 从message中获取group_id
	groupID := message.Params.GroupID.(string)
	// 使用RetrieveRowByIDv2还原真实的ChannelID
	RChannelID, err := idmap.RetrieveRowByIDv2(groupID)
	if err != nil {
		log.Printf("error retrieving real UserID: %v", err)
	}
	// 根据group_id读取guild_id
	guildID, err := idmap.ReadConfigv2(RChannelID, "guild_id")
	if err != nil {
		log.Printf("Error reading config for guild_id: %v", err)
		return
	}

	// 读取消息类型
	msgType, err := idmap.ReadConfigv2(groupID, "type")
	if err != nil {
		log.Printf("Error reading config for message type: %v", err)
		return
	}

	// 根据消息类型进行操作
	switch msgType {
	case "group":
		log.Printf("setGroupWholeBan(频道): 目前暂未开放该能力")
		return
	case "private":
		log.Printf("setGroupWholeBan(频道): 目前暂未适配私聊虚拟群场景的禁言能力")
		return
	case "guild":
		var duration string
		if message.Params.Enable {
			duration = "604800" // 7天: 60 * 60 * 24 * 7 onebot的全体禁言只有禁言和解开,先尝试7天
		} else {
			duration = "0"
		}

		mute := &dto.UpdateGuildMute{
			MuteSeconds: duration,
		}
		err := api.GuildMute(context.TODO(), guildID, mute)
		if err != nil {
			log.Printf("Error setting whole guild mute: %v", err)
		}
		return
	}
}
