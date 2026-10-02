package v1

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
)

// GetGroupInfo 获取群基本信息
func (o *openAPI) GetGroupInfo(
	ctx context.Context,
	groupOpenID string,
) (*dto.GroupInfo, error) {

	resp, err := o.request(ctx).
		SetResult(dto.GroupInfo{}).
		SetPathParam("group_openid", groupOpenID).
		Get(o.getURL(groupInfoURI))

	if err != nil {
		return nil, err
	}

	return resp.Result().(*dto.GroupInfo), nil
}

// GetGroupMuteState 获取QQ群当前禁言状态。
func (o *openAPI) GetGroupMuteState(
	ctx context.Context,
	groupOpenID string,
) (*dto.GroupMuteState, error) {

	resp, err := o.request(ctx).
		SetResult(dto.GroupMuteState{}).
		SetPathParam("group_openid", groupOpenID).
		Get(o.getURL(groupRestrictChatSettingURI))

	if err != nil {
		return nil, err
	}

	return resp.Result().(*dto.GroupMuteState), nil
}

// SetGroupMemberMute 设置QQ群单个成员的禁言状态。
func (o *openAPI) SetGroupMemberMute(
	ctx context.Context,
	groupOpenID string,
	mute *dto.SetGroupMemberMute,
) error {

	_, err := o.request(ctx).
		SetPathParam("group_openid", groupOpenID).
		SetBody(mute).
		Post(o.getURL(groupRestrictChatSettingURI))

	if err != nil {
		return err
	}

	return nil
}
