package handlers

import (
	"context"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_whole_ban", SetGroupWholeBan)
}

func SetGroupWholeBan(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	// 从message中获取group_id
	groupID := message.Params.GroupID.(string)
	//读取ini 通过ChannelID取回之前储存的guild_id
	guildID, err := idmap.ReadConfigv2(groupID, "guild_id")
	if err != nil {
		mylog.Printf("Error reading config: %v", err)
		return "", nil
	}
	// 读取消息类型
	msgType, err := idmap.ReadConfigv2(groupID, "type")
	if err != nil {
		mylog.Printf("Error reading config for message type: %v", err)
		return "", nil
	}

	// 根据消息类型进行操作
	switch msgType {
	case "group":
		mylog.Printf("setGroupWholeBan(频道): 目前暂未开放该能力")
		return "", nil
	case "private":
		mylog.Printf("setGroupWholeBan(频道): 目前暂未适配私聊虚拟群场景的禁言能力")
		return "", nil
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
			mylog.Printf("Error setting whole guild mute: %v", err)
		}
		return "", nil
	}
	return "", nil
}
