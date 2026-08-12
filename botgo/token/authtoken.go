package token

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/tencent-connect/botgo/log"
)

// getAccessTokenURL 取得AccessToken的地址
// https://bots.qq.com/app/getAppAccessToken
var getAccessTokenURL = "https://api.bot.qq.com/app/getAppAccessToken"

// AuthTokenInfo 动态鉴权Token信息
type AuthTokenInfo struct {
	accessToken  AccessTokenInfo
	lock         *sync.RWMutex
	refreshLock  *sync.Mutex
	forceUpToken chan interface{}
	once         sync.Once
}

// AccessTokenInfo 鉴权Token信息
type AccessTokenInfo struct {
	Token     string
	ExpiresIn int64
	UpTime    time.Time
}

// NewAuthTokenInfo 初始化动态鉴权Token
func NewAuthTokenInfo() *AuthTokenInfo {
	return &AuthTokenInfo{
		lock:         &sync.RWMutex{},
		refreshLock:  &sync.Mutex{},
		forceUpToken: make(chan interface{}, 10),
	}
}

// ForceUpToken 强制刷新Token
func (atoken *AuthTokenInfo) ForceUpToken(ctx context.Context, reason string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("uptoken err:%v", ctx.Err())
	case atoken.forceUpToken <- reason:
	}
	return nil
}

// isTokenValid 判断当前 AccessToken 在预留 buffer 后是否仍然有效。
func (atoken *AuthTokenInfo) isTokenValid(buffer time.Duration) bool {
	tokenInfo := atoken.getAuthToken()

	if tokenInfo.Token == "" ||
		tokenInfo.ExpiresIn <= 0 ||
		tokenInfo.UpTime.IsZero() {
		return false
	}

	expireAt := tokenInfo.UpTime.Add(
		time.Duration(tokenInfo.ExpiresIn) * time.Second,
	)

	return time.Now().Before(
		expireAt.Add(-buffer),
	)
}

// refreshAccessToken 真正执行 Token 刷新。
// force=false 时，在拿到刷新锁后会再次检查，避免并发重复刷新。
func (atoken *AuthTokenInfo) refreshAccessToken(
	ctx context.Context,
	tokenURL string,
	appID string,
	clientSecret string,
	force bool,
) (AccessTokenInfo, error) {

	atoken.refreshLock.Lock()
	defer atoken.refreshLock.Unlock()

	// 其他 goroutine 可能已经完成刷新。
	if !force && atoken.isTokenValid(60*time.Second) {
		return atoken.getAuthToken(), nil
	}

	// 与 Elaina 的节奏一致：
	// 最多尝试3次，第一次失败等3秒，第二次失败等6秒。
	retryDelays := []time.Duration{
		3 * time.Second,
		6 * time.Second,
	}

	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		tokenInfo, err := queryAccessToken(
			ctx,
			tokenURL,
			appID,
			clientSecret,
		)

		if err == nil &&
			tokenInfo.Token != "" &&
			tokenInfo.ExpiresIn > 0 {

			atoken.setAuthToken(tokenInfo)

			// 不输出真正的 Token。
			log.Infof(
				"AccessToken刷新成功, expires_in=%d",
				tokenInfo.ExpiresIn,
			)

			return tokenInfo, nil
		}

		if err == nil {
			err = fmt.Errorf(
				"invalid AccessToken response: token_empty=%t expires_in=%d",
				tokenInfo.Token == "",
				tokenInfo.ExpiresIn,
			)
		}

		lastErr = err

		log.Errorf(
			"AccessToken刷新失败 attempt=%d/3 err=%v",
			attempt+1,
			err,
		)

		if attempt < len(retryDelays) {
			select {
			case <-time.After(retryDelays[attempt]):

			case <-ctx.Done():
				return AccessTokenInfo{}, ctx.Err()
			}
		}
	}

	return AccessTokenInfo{}, lastErr
}

// EnsureAccessToken 用于 OpenAPI 请求前检查 Token。
// 有可用旧 Token 时，不阻塞业务请求：
// 即使 Token 已进入刷新窗口，也继续使用当前 Token，
// 同时通知后台刷新协程尽快刷新。
// 只有完全没有 Token 时，才同步获取一次。
func (atoken *AuthTokenInfo) EnsureAccessToken(
	ctx context.Context,
	tokenURL string,
	appID string,
	clientSecret string,
) error {

	tokenInfo := atoken.getAuthToken()

	// 完全没有旧 Token 时无法降级，只能同步获取。
	if tokenInfo.Token == "" {
		_, err := atoken.refreshAccessToken(
			ctx,
			tokenURL,
			appID,
			clientSecret,
			false,
		)
		return err
	}

	// 当前 Token 还有至少60秒有效期，直接使用。
	if atoken.isTokenValid(60 * time.Second) {
		return nil
	}

	// 已进入刷新窗口。
	// 不阻塞当前 OpenAPI 请求，继续使用已有 Token。
	// 仅通知后台刷新协程尽快刷新。
	select {
	case atoken.forceUpToken <- "OpenAPI请求发现AccessToken进入刷新窗口":
	default:
		// 刷新通知队列已有任务时直接忽略，
		// 避免业务请求因为通知刷新而阻塞。
	}

	return nil
}

// StartRefreshAccessToken 启动 AccessToken 管理。
// 1. 启动时立即获取。
// 2. 后台提前60秒刷新。
// 3. 刷新失败后最多立即尝试3次。
// 4. 仍失败则30秒后继续尝试。
// 5. OpenAPI 请求前还会通过 EnsureAccessToken 再做一次保险。
func (atoken *AuthTokenInfo) StartRefreshAccessToken(
	ctx context.Context,
	tokenURL string,
	appID string,
	clientSecret string,
) error {

	// 启动时立即获取一次。
	_, initialErr := atoken.refreshAccessToken(
		ctx,
		tokenURL,
		appID,
		clientSecret,
		true,
	)

	if initialErr != nil {
		log.Errorf(
			"初始化AccessToken失败: %v",
			initialErr,
		)
	}

	atoken.once.Do(func() {
		go func() {
			for {
				tokenInfo := atoken.getAuthToken()

				// 默认30秒后再检查。
				// 如果当前有有效 Token，则计算提前60秒刷新的时间点。
				waitDuration := 30 * time.Second

				if tokenInfo.Token != "" &&
					tokenInfo.ExpiresIn > 0 &&
					!tokenInfo.UpTime.IsZero() {

					expireAt := tokenInfo.UpTime.Add(
						time.Duration(tokenInfo.ExpiresIn) *
							time.Second,
					)

					refreshAt := expireAt.Add(
						-60 * time.Second,
					)

					waitDuration = time.Until(refreshAt)

					// 与 Elaina 思路一致：
					// 即使已经进入刷新窗口，也最多等30秒重新刷新，
					// 避免高频空转。
					if waitDuration < 30*time.Second {
						waitDuration = 30 * time.Second
					}
				}

				timer := time.NewTimer(waitDuration)

				select {
				case <-timer.C:
					// 到达自动刷新时间。

				case upToken := <-atoken.forceUpToken:
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}

					log.Warnf(
						"recv uptoken info:%v",
						upToken,
					)

				case <-ctx.Done():
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}

					log.Warnf(
						"recv ctx:%v exit refreshAccessToken",
						ctx.Err(),
					)

					return
				}

				_, err := atoken.refreshAccessToken(
					ctx,
					tokenURL,
					appID,
					clientSecret,
					true,
				)

				if err != nil {
					log.Errorf(
						"后台AccessToken刷新失败: %v",
						err,
					)

					// 不需要额外 sleep。
					// 因为旧 Token 的刷新时间已经过去，
					// 下一轮 waitDuration 自动会变成30秒。
				}
			}
		}()
	})

	return initialErr
}

// 测试用
// func (atoken *AuthTokenInfo) StartRefreshAccessToken(ctx context.Context, tokenURL, appID, clientSecrent string) (err error) {
// 	// 创建一个固定的token信息
// 	fixedTokenInfo := AccessTokenInfo{
// 		Token:     "PyR4PL9_eRfAkIIlWE4nAawocFMlPfQCySgASB5vJRduWgKh0mSOp4zm4AOzDKpweV9iu5zq-OWm",
// 		ExpiresIn: 3600, // 这里假设token的有效时间是3600秒，你可以根据需要调整
// 	}
// 	atoken.setAuthToken(fixedTokenInfo)
// 	return nil
// }

func (atoken *AuthTokenInfo) getAuthToken() AccessTokenInfo {
	atoken.lock.RLock()
	defer atoken.lock.RUnlock()
	return atoken.accessToken

}

func (atoken *AuthTokenInfo) setAuthToken(accessToken AccessTokenInfo) {
	atoken.lock.Lock()
	defer atoken.lock.Unlock()
	atoken.accessToken = accessToken
}

type queryTokenReq struct {
	AppID        string `json:"appId"`
	ClientSecret string `json:"clientSecret"`
}

// 自定义地址返回需满足这个格式
type queryTokenRsp struct {
	AccessToken string      `json:"access_token"`
	ExpiresIn   interface{} `json:"expires_in"` // 允许任何类型
}

// func queryAccessToken(ctx context.Context, tokenURL, appID, clientSecrent string) (AccessTokenInfo, error) {
// 	method := "POST"

// 	defer func() {
// 		if err := recover(); err != nil {
// 			log.Errorf("queryAccessToken err:%v", err)
// 		}
// 	}()
// 	if tokenURL == "" {
// 		tokenURL = getAccessTokenURL
// 	}

// 	queryReq := queryTokenReq{
// 		AppID:        appID,
// 		ClientSecret: clientSecrent,
// 	}
// 	data, err := json.Marshal(queryReq)
// 	if err != nil {
// 		return AccessTokenInfo{}, err
// 	}
// 	payload := bytes.NewReader(data)
// 	log.Infof("reqdata:%v", string(data))
// 	client := &http.Client{
// 		Timeout: 10 * time.Second,
// 	}
// 	log.Infof("tokenURL:%v", tokenURL)
// 	req, err := http.NewRequest(method, tokenURL, payload)
// 	if err != nil {
// 		log.Errorf("NewRequest err:%v", err)
// 		return AccessTokenInfo{}, err
// 	}
// 	req.Header.Add("Content-Type", "application/json")
// 	res, err := client.Do(req)
// 	if err != nil {
// 		log.Errorf("http do err:%v", err)
// 		return AccessTokenInfo{}, err
// 	}
// 	defer res.Body.Close()

// 	body, err := ioutil.ReadAll(res.Body)
// 	if err != nil {
// 		log.Errorf("ReadAll do err:%v", err)
// 		return AccessTokenInfo{}, err
// 	}
// 	log.Infof("accesstoken:%v", string(body))
// 	queryRsp := queryTokenRsp{}
// 	if err = json.Unmarshal(body, &queryRsp); err != nil {
// 		log.Errorf("Unmarshal err:%v", err)
// 		return AccessTokenInfo{}, err
// 	}

// 	rdata := AccessTokenInfo{
// 		Token:  queryRsp.AccessToken,
// 		UpTime: time.Now(),
// 	}
// 	rdata.ExpiresIn, _ = strconv.ParseInt(queryRsp.ExpiresIn, 10, 64)
// 	return rdata, err
// }

func queryAccessToken(ctx context.Context, tokenURL, appID, clientSecret string) (AccessTokenInfo, error) {
	// 是否使用自定义ac地址
	configURL := config.GetDevelop_Acdir()
	if tokenURL == "" {
		if configURL != "" {
			tokenURL = configURL
		} else {
			tokenURL = getAccessTokenURL // 默认值
		}
	}

	var req *http.Request
	var err error

	// 自定义地址使用GET，无需参数。
	if configURL != "" {
		req, err = http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			configURL,
			nil,
		)
		if err != nil {
			return AccessTokenInfo{}, err
		}
	} else {
		// 默认AccessToken地址使用POST。
		reqBody := queryTokenReq{
			AppID:        appID,
			ClientSecret: clientSecret,
		}

		reqData, err := json.Marshal(reqBody)
		if err != nil {
			return AccessTokenInfo{}, err
		}

		req, err = http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			tokenURL,
			bytes.NewReader(reqData),
		)
		if err != nil {
			return AccessTokenInfo{}, err
		}

		req.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	// Perform the HTTP request.
	// Token 获取必须有明确超时，否则同步 EnsureAccessToken 时
	// 可能导致所有 OpenAPI 请求一起等待。
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return AccessTokenInfo{}, err
	}
	defer resp.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AccessTokenInfo{}, err
	}

	var respData queryTokenRsp
	if err = json.Unmarshal(body, &respData); err != nil {
		return AccessTokenInfo{}, err
	}

	// 自定义地址返回可能是string或者int
	expiresInInt, err := parseExpiresIn(respData.ExpiresIn)
	if err != nil {
		return AccessTokenInfo{}, fmt.Errorf("expires_in is not a valid int or string: %v", err)
	}

	return AccessTokenInfo{
		Token:     respData.AccessToken,
		ExpiresIn: expiresInInt,
		UpTime:    time.Now(),
	}, nil
}

func parseExpiresIn(expiresIn interface{}) (int64, error) {
	switch v := expiresIn.(type) {
	case float64:
		// JSON numbers come as floats by default
		return int64(v), nil
	case int64:
		// In case the JSON decoder was set to use integers
		return v, nil
	case json.Number:
		// Convert json.Number to int64
		return v.Int64()
	case string:
		// If the value is a string, try converting it to an integer
		return strconv.ParseInt(v, 10, 64)
	default:
		// If none of the above types matched, return an error
		return 0, fmt.Errorf("expires_in is not a valid type, got %T", expiresIn)
	}
}

//原函数
// func queryAccessToken(ctx context.Context, tokenURL, appID, clientSecret string) (AccessTokenInfo, error) {
// 	if tokenURL == "" {
// 		tokenURL = getAccessTokenURL // Assumes getAccessTokenURL is declared elsewhere
// 	}

// 	reqBody := queryTokenReq{
// 		AppID:        appID,
// 		ClientSecret: clientSecret,
// 	}
// 	reqData, err := json.Marshal(reqBody)
// 	if err != nil {
// 		return AccessTokenInfo{}, err
// 	}

// 	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(reqData))
// 	if err != nil {
// 		return AccessTokenInfo{}, err
// 	}
// 	req.Header.Add("Content-Type", "application/json")

// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return AccessTokenInfo{}, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return AccessTokenInfo{}, err
// 	}

// 	var respData queryTokenRsp
// 	if err := json.Unmarshal(body, &respData); err != nil {
// 		return AccessTokenInfo{}, err
// 	}

// 	expiresIn, _ := strconv.ParseInt(respData.ExpiresIn, 10, 64) // Ignoring error can be dangerous, handle it as needed

// 	return AccessTokenInfo{
// 		Token:     respData.AccessToken,
// 		ExpiresIn: expiresIn,
// 		UpTime:    time.Now(),
// 	}, nil
// }
