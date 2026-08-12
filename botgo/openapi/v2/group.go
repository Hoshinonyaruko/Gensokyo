package v2

import (
	"context"
	"strings"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/log"
)

func (o *openAPIv2) GetGroupInfo(
	ctx context.Context,
	groupOpenID string,
) (*dto.GroupInfo, error) {

	// 记录本次请求真正使用之前的Token。
	// 如果腾讯返回100016，需要知道到底是哪一个Token被拒绝。
	staleToken := o.token.GetAccessToken()

	// 第一次正常请求。
	resp, err := o.request(ctx).
		SetResult(dto.GroupInfo{}).
		SetPathParam("group_openid", groupOpenID).
		Get(o.getURL(groupInfoURI))

	if err == nil {
		return resp.Result().(*dto.GroupInfo), nil
	}

	// 只处理腾讯明确返回的AccessToken过期错误。
	// 其他错误保持原有行为，不刷新、不重试。
	if !isAccessTokenExpiredError(err) {
		return nil, err
	}

	log.Warnf(
		"GetGroupInfo收到100016，立即刷新AccessToken并重试一次, group_openid=%s",
		groupOpenID,
	)

	// 同步刷新。
	//
	// 如果并发情况下其他goroutine已经把staleToken换掉，
	// RefreshAccessTokenIfStale会直接使用别人取得的新Token，
	// 不会再次重复刷新。
	if refreshErr := o.token.RefreshAccessTokenIfStale(
		ctx,
		staleToken,
	); refreshErr != nil {

		log.Errorf(
			"GetGroupInfo收到100016后刷新AccessToken失败, group_openid=%s, err=%v",
			groupOpenID,
			refreshErr,
		)

		return nil, err
	}

	// 只重试一次。
	resp, err = o.request(ctx).
		SetResult(dto.GroupInfo{}).
		SetPathParam("group_openid", groupOpenID).
		Get(o.getURL(groupInfoURI))

	if err != nil {
		return nil, err
	}

	log.Infof(
		"GetGroupInfo收到100016后刷新AccessToken重试成功, group_openid=%s",
		groupOpenID,
	)

	return resp.Result().(*dto.GroupInfo), nil
}

// isAccessTokenExpiredError 判断腾讯是否明确返回AccessToken过期。
// 当前腾讯实际返回：
// {"message":"connectapp biz failed: access_token is expired","code":100016,"err_code":100016}
func isAccessTokenExpiredError(err error) bool {
	if err == nil {
		return false
	}

	errText := err.Error()

	return strings.Contains(
		errText,
		`"code":100016`,
	) ||
		strings.Contains(
			errText,
			`"err_code":100016`,
		) ||
		strings.Contains(
			errText,
			"access_token is expired",
		)
}
