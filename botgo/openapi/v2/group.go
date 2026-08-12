package v2

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/tencent-connect/botgo/dto"
)

func (o *openAPIv2) GetGroupInfo(
	ctx context.Context,
	groupOpenID string,
) (*dto.GroupInfo, error) {

	// 请求开始前，管理器当前持有的Token。
	managerTokenBefore := o.token.GetAccessToken()

	resp, err := o.request(ctx).
		SetResult(dto.GroupInfo{}).
		SetPathParam("group_openid", groupOpenID).
		Get(o.getURL(groupInfoURI))

	// ============================================================
	// 获取这一次HTTP真正发送出去的Authorization Token。
	//
	// 不能只看 managerTokenBefore，
	// 因为 OnBeforeRequest 阶段仍然可能更新Authorization。
	// RawRequest才代表真正发给腾讯的HTTP请求。
	// ============================================================
	requestToken := managerTokenBefore

	if resp != nil &&
		resp.Request != nil &&
		resp.Request.RawRequest != nil {

		authHeader := resp.Request.RawRequest.Header.Get(
			"Authorization",
		)

		if strings.HasPrefix(authHeader, "QQBot ") {
			requestToken = strings.TrimSpace(
				strings.TrimPrefix(
					authHeader,
					"QQBot ",
				),
			)
		}
	}

	// 请求结束后，再读取一次当前管理器Token。
	managerTokenAfter := o.token.GetAccessToken()

	// ============================================================
	// 成功请求也记录：
	// PID
	// 管理器Token
	// HTTP实际Token
	//
	// 这样之后第一次100016出现时，
	// 可以直接比较“之前成功”和“现在失败”是不是同一枚Token。
	// ============================================================
	if err == nil {

		fmt.Printf(
			"[TOKEN-DIAG-SUCCESS] pid=%d group=%s manager_before=%s manager_after=%s request=%s before_eq_request=%t after_eq_request=%t\n",
			os.Getpid(),
			groupOpenID,
			shortToken(managerTokenBefore),
			shortToken(managerTokenAfter),
			shortToken(requestToken),
			managerTokenBefore != "" &&
				requestToken != "" &&
				managerTokenBefore == requestToken,
			managerTokenAfter != "" &&
				requestToken != "" &&
				managerTokenAfter == requestToken,
		)

		return resp.Result().(*dto.GroupInfo), nil
	}

	// 普通失败也先输出实际请求Token。
	fmt.Printf(
		"[TOKEN-DIAG-FAILED] pid=%d group=%s manager_before=%s manager_after=%s request=%s err=%v\n",
		os.Getpid(),
		groupOpenID,
		shortToken(managerTokenBefore),
		shortToken(managerTokenAfter),
		shortToken(requestToken),
		err,
	)

	// 不是100016，保持原行为。
	if !isAccessTokenExpiredError(err) {
		return nil, err
	}

	// ============================================================
	// 到这里就是冷曦说的：
	// “100016 那一刻”
	//
	// 此时我们已经知道：
	// requestToken = 这个失败HTTP真正使用的Token
	//
	// 接下来主动请求一次腾讯Token接口，
	// 只观察：
	//   1. 腾讯现在返回哪个Token
	//   2. 腾讯原样返回的expires_in
	//
	// DebugQueryAccessToken不会写入本地Token。
	// ============================================================
	probeInfo, probeErr := o.token.DebugQueryAccessToken(ctx)

	managerTokenNow := o.token.GetAccessToken()

	if probeErr != nil {

		fmt.Printf(
			"[TOKEN-DIAG] pid=%d group=%s manager=%s request=%s tencent=<probe_failed> expires_in=? manager_eq_request=%t err=%v\n",
			os.Getpid(),
			groupOpenID,
			shortToken(managerTokenNow),
			shortToken(requestToken),
			managerTokenNow != "" &&
				requestToken != "" &&
				managerTokenNow == requestToken,
			probeErr,
		)

	} else {

		fmt.Printf(
			"[TOKEN-DIAG] pid=%d group=%s manager=%s request=%s tencent=%s expires_in=%d manager_eq_request=%t request_eq_tencent=%t manager_eq_tencent=%t\n",
			os.Getpid(),
			groupOpenID,
			shortToken(managerTokenNow),
			shortToken(requestToken),
			shortToken(probeInfo.Token),
			probeInfo.ExpiresIn,
			managerTokenNow != "" &&
				requestToken != "" &&
				managerTokenNow == requestToken,
			requestToken != "" &&
				probeInfo.Token != "" &&
				requestToken == probeInfo.Token,
			managerTokenNow != "" &&
				probeInfo.Token != "" &&
				managerTokenNow == probeInfo.Token,
		)
	}

	fmt.Printf(
		"[GetGroupInfo] 收到100016，立即刷新AccessToken并重试一次, group_openid=%s, err=%v\n",
		groupOpenID,
		err,
	)

	// ============================================================
	// 注意这里：
	// 用“这次HTTP真正被腾讯拒绝的requestToken”作为staleToken。
	// 不再使用请求开始前记录的managerTokenBefore。
	// ============================================================
	if refreshErr := o.token.RefreshAccessTokenIfStale(
		ctx,
		requestToken,
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

		retryToken := o.token.GetAccessToken()

		if resp != nil &&
			resp.Request != nil &&
			resp.Request.RawRequest != nil {

			authHeader := resp.Request.RawRequest.Header.Get(
				"Authorization",
			)

			if strings.HasPrefix(authHeader, "QQBot ") {
				retryToken = strings.TrimSpace(
					strings.TrimPrefix(
						authHeader,
						"QQBot ",
					),
				)
			}
		}

		fmt.Printf(
			"[TOKEN-DIAG-RETRY] pid=%d group=%s first_request=%s retry_request=%s same=%t err=%v\n",
			os.Getpid(),
			groupOpenID,
			shortToken(requestToken),
			shortToken(retryToken),
			requestToken != "" &&
				retryToken != "" &&
				requestToken == retryToken,
			err,
		)

		fmt.Printf(
			"[GetGroupInfo] 100016刷新后重试仍失败, group_openid=%s, err=%v\n",
			groupOpenID,
			err,
		)

		return nil, err
	}

	retryToken := o.token.GetAccessToken()

	if resp != nil &&
		resp.Request != nil &&
		resp.Request.RawRequest != nil {

		authHeader := resp.Request.RawRequest.Header.Get(
			"Authorization",
		)

		if strings.HasPrefix(authHeader, "QQBot ") {
			retryToken = strings.TrimSpace(
				strings.TrimPrefix(
					authHeader,
					"QQBot ",
				),
			)
		}
	}

	fmt.Printf(
		"[TOKEN-DIAG-RETRY] pid=%d group=%s first_request=%s retry_request=%s same=%t result=success\n",
		os.Getpid(),
		groupOpenID,
		shortToken(requestToken),
		shortToken(retryToken),
		requestToken != "" &&
			retryToken != "" &&
			requestToken == retryToken,
	)

	fmt.Printf(
		"[GetGroupInfo] 100016刷新后重试成功, group_openid=%s\n",
		groupOpenID,
	)

	return resp.Result().(*dto.GroupInfo), nil
}

// shortToken 对Token做SHA256，只显示前12位十六进制。
// 不泄露Token原文，但足够判断两个Token是不是同一个。
func shortToken(token string) string {
	if token == "" {
		return "<empty>"
	}

	sum := sha256.Sum256([]byte(token))

	return fmt.Sprintf(
		"%x",
		sum[:6],
	)
}

// isAccessTokenExpiredError 判断腾讯是否明确返回AccessToken过期。
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
