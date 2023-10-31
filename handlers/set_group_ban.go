package handlers

import (
	"context"
	"log"
	"strconv"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("get_group_ban", setGroupBan)
}

func setGroupBan(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) {

	// 从message中获取group_id和UserID
	groupID := message.Params.GroupID.(string)
	receivedUserID := message.Params.UserID.(string)
	// 使用RetrieveRowByIDv2还原真实的ChannelID
	RChannelID, err := idmap.RetrieveRowByIDv2(groupID)
	if err != nil {
		log.Printf("error retrieving real UserID: %v", err)
	}
	// 根据RChannelID读取guild_id
	guildID, err := idmap.ReadConfigv2(RChannelID, "guild_id")
	if err != nil {
		log.Printf("Error reading config for guild_id: %v", err)
		return
	}

	// 根据UserID读取真实的userid
	realUserID, err := idmap.RetrieveRowByIDv2(receivedUserID)
	if err != nil {
		log.Printf("Error reading real userID: %v", err)
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
		log.Printf("setGroupBan(频道): 目前暂未开放该能力")
		return
	case "private":
		log.Printf("setGroupBan(频道): 目前暂未适配私聊虚拟群场景的禁言能力")
		return
	case "guild":
		duration := strconv.Itoa(message.Params.Duration)
		mute := &dto.UpdateGuildMute{
			MuteSeconds: duration,
			UserIDs:     []string{realUserID},
		}
		err := api.MemberMute(context.TODO(), guildID, realUserID, mute)
		if err != nil {
			log.Printf("Error muting member: %v", err)
		}
		return
	}
}
