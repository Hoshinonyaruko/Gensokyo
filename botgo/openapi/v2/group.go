package v2

import (
	"context"
	"fmt"
	"strings"

	"github.com/tencent-connect/botgo/dto"
)

func (o *openAPIv2) GetGroupInfo(
	ctx context.Context,
	groupOpenID string,
) (*dto.GroupInfo, error) {

	// 记录调用前的Token。
	// 如果腾讯返回100016，用于判断刷新后Token是否已经被其他goroutine替换。
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

	fmt.Printf(
		"[GetGroupInfo] 收到100016，立即刷新AccessToken并重试一次, group_openid=%s, err=%v\n",
		groupOpenID,
		err,
	)

	// 同步刷新。
	// 如果等待refreshLock期间已有其他goroutine完成刷新，
	// RefreshAccessTokenIfStale会直接使用已经刷新的Token，
	// 不会再次重复刷新。
	if refreshErr := o.token.RefreshAccessTokenIfStale(
		ctx,
		staleToken,
	); refreshErr != nil {

		fmt.Printf(
			"[GetGroupInfo] 收到100016后刷新AccessToken失败, group_openid=%s, err=%v\n",
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
		fmt.Printf(
			"[GetGroupInfo] 100016刷新后重试仍失败, group_openid=%s, err=%v\n",
			groupOpenID,
			err,
		)

		return nil, err
	}

	fmt.Printf(
		"[GetGroupInfo] 100016刷新后重试成功, group_openid=%s\n",
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
