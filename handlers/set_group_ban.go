package handlers

import (
	"context"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("set_group_ban", SetGroupBan)
}

// SetGroupBan QQ群单成员禁言。
//
// 仅适配 StringOB11 模式下腾讯开放平台原始 32 字符 OpenID：
//
//	group_id = group_openid
//	user_id  = member_openid
//
// 不进行任何 idmap 转换，也不兼容旧的虚拟整数 ID。
func SetGroupBan(
	client callapi.Client,
	api openapi.OpenAPI,
	apiv2 openapi.OpenAPI,
	message callapi.ActionMessage,
) (string, error) {

	// 仅支持 StringOB11 原始 OpenID 模式。
	if !config.GetStringOb11() {
		mylog.Printf(
			"set_group_ban: 当前仅支持 StringOB11 32位OpenID模式",
		)
		return "", nil
	}

	// group_id 在该模式下直接就是腾讯 group_openid。
	groupID, ok := message.Params.GroupID.(string)
	if !ok || len(groupID) != 32 {
		mylog.Printf(
			"set_group_ban: group_id 必须是32位字符串, group_id=%v",
			message.Params.GroupID,
		)
		return "", nil
	}

	// user_id 在该模式下直接就是腾讯 member_openid。
	userID, ok := message.Params.UserID.(string)
	if !ok || len(userID) != 32 {
		mylog.Printf(
			"set_group_ban: user_id 必须是32位字符串, user_id=%v",
			message.Params.UserID,
		)
		return "", nil
	}

	duration := message.Params.Duration

	if duration < 0 {
		mylog.Printf(
			"set_group_ban: duration不能小于0, duration=%d",
			duration,
		)
		return "", nil
	}

	ctx := context.Background()

	// 先查询当前禁言状态。
	//
	// QQ群禁言接口区分：
	// add    新增禁言
	// update 修改已有禁言
	// del    解除禁言
	//
	// 因此先GET一次，判断目标成员当前是否已经处于禁言状态。
	state, err := apiv2.GetGroupMuteState(
		ctx,
		groupID,
	)
	if err != nil {
		mylog.Printf(
			"set_group_ban: 获取群禁言状态失败, group_openid=%s user_openid=%s err=%v",
			groupID,
			userID,
			err,
		)
		return "", nil
	}

	// 判断目标成员当前是否已经被禁言。
	isMuted := false

	if state != nil {
		for _, member := range state.Members {
			if member.MemberOpenID == userID {
				isMuted = true
				break
			}
		}
	}

	// OneBot:
	//
	// duration == 0
	// 表示解除禁言。
	if duration == 0 {

		// 本来就没有被禁言，直接视为成功。
		// 不需要向QQ重复发送del。
		if !isMuted {
			mylog.Printf(
				"set_group_ban: 用户当前未被禁言，无需解除, group_openid=%s user_openid=%s",
				groupID,
				userID,
			)
			return "", nil
		}

		mute := &dto.SetGroupMemberMute{
			Members: []dto.SetGroupMemberMuteState{
				{
					Op:           "del",
					MemberOpenID: userID,
				},
			},
		}

		err = apiv2.SetGroupMemberMute(
			ctx,
			groupID,
			mute,
		)
		if err != nil {
			mylog.Printf(
				"set_group_ban: 解除群成员禁言失败, group_openid=%s user_openid=%s err=%v",
				groupID,
				userID,
				err,
			)
			return "", nil
		}

		mylog.Printf(
			"set_group_ban: 解除群成员禁言成功, group_openid=%s user_openid=%s",
			groupID,
			userID,
		)

		return "", nil
	}

	// duration > 0
	//
	// QQ接口要求的是RFC3339格式的禁言截止时间，
	// OneBot传入的是禁言秒数。
	muteExpireAt := time.Now().
		UTC().
		Add(time.Duration(duration) * time.Second).
		Format(time.RFC3339)

	op := "add"

	if isMuted {
		op = "update"
	}

	mute := &dto.SetGroupMemberMute{
		Members: []dto.SetGroupMemberMuteState{
			{
				Op:           op,
				MemberOpenID: userID,
				MuteExpireAt: muteExpireAt,
			},
		},
	}

	err = apiv2.SetGroupMemberMute(
		ctx,
		groupID,
		mute,
	)
	if err != nil {
		mylog.Printf(
			"set_group_ban: 设置群成员禁言失败, group_openid=%s user_openid=%s duration=%d op=%s err=%v",
			groupID,
			userID,
			duration,
			op,
			err,
		)
		return "", nil
	}

	mylog.Printf(
		"set_group_ban: 设置群成员禁言成功, group_openid=%s user_openid=%s duration=%d op=%s mute_expire_at=%s",
		groupID,
		userID,
		duration,
		op,
		muteExpireAt,
	)

	return "", nil
}
