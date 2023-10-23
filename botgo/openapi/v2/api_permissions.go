package v2

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
)

// GetAPIPermissions 获取频道可用权限列表
func (o *openAPIv2) GetAPIPermissions(ctx context.Context, guildID string) (*dto.APIPermissions, error) {
	resp, err := o.request(ctx).
		SetResult(dto.APIPermissions{}).
		SetPathParam("guild_id", guildID).
		Get(o.getURL(apiPermissionURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.APIPermissions), nil
}

// RequireAPIPermissions 创建频道 API 接口权限授权链接
func (o *openAPIv2) RequireAPIPermissions(ctx context.Context,
	guildID string, demand *dto.APIPermissionDemandToCreate) (*dto.APIPermissionDemand, error) {
	resp, err := o.request(ctx).
		SetResult(dto.APIPermissionDemand{}).
		SetPathParam("guild_id", guildID).
		SetBody(demand).
		Post(o.getURL(apiPermissionDemandURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.APIPermissionDemand), nil
}
