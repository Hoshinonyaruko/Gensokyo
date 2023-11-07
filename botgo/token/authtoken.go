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
var getAccessTokenURL = "https://bots.qq.com/app/getAppAccessToken"

// AuthTokenInfo 动态鉴权Token信息
type AuthTokenInfo struct {
	accessToken  AccessTokenInfo
	lock         *sync.RWMutex
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

// StartRefreshAccessToken 启动获取AccessToken的后台刷新
// 该函数首先会立即查询一次AccessToken，并保存。
// 然后它将在后台启动一个goroutine，定期（根据token的有效期）刷新AccessToken。
func (atoken *AuthTokenInfo) StartRefreshAccessToken(ctx context.Context, tokenURL, appID, clientSecrent string) (err error) {
	// 首先，立即获取一次AccessToken
	tokenInfo, err := queryAccessToken(ctx, tokenURL, appID, clientSecrent)
	if err != nil {
		log.Errorf("无法获取AccessToken: %v", err)
		//return err
	}
	atoken.setAuthToken(tokenInfo)
	log.Info("获取到的token是: %s\n", tokenInfo.Token) // 输出获取到的token

	// 获取token的有效期（通常以秒为单位）
	tokenTTL := tokenInfo.ExpiresIn
	// 使用sync.Once保证仅启动一个goroutine进行定时刷新
	atoken.once.Do(func() {
		go func() { // 启动一个新的goroutine
			for {
				// 如果tokenTTL为0或负数，将其设置为15
				if tokenTTL <= 0 {
					tokenTTL = 15
				}
				select {
				case <-time.NewTimer(time.Duration(tokenTTL) * time.Second).C: // 当token过期时
				case upToken := <-atoken.forceUpToken: // 接收强制更新token的信号
					log.Warnf("recv uptoken info:%v", upToken)
				case <-ctx.Done(): // 当上下文结束时，退出goroutine
					log.Warnf("recv ctx:%v exit refreshAccessToken", ctx.Err())
					return
				}
				// 查询并获取新的AccessToken
				tokenInfo, err := queryAccessToken(ctx, tokenURL, appID, clientSecrent)
				if err == nil {
					atoken.setAuthToken(tokenInfo)
					log.Info("获取到的token是: %s\n", tokenInfo.Token) // 输出获取到的token
					tokenTTL = tokenInfo.ExpiresIn
				} else {
					log.Errorf("queryAccessToken err:%v", err)
					log.Errorf("请在config.yml或网页控制台的默认机器人中设置正确的appid和密钥信息")
				}
			}
		}()
	})
	return
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

	//自定义地址使用get,无需参数
	if configURL != "" {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, configURL, nil)
	} else {
		// 默认ac地址使用post
		reqBody := queryTokenReq{
			AppID:        appID,
			ClientSecret: clientSecret,
		}
		reqData, err := json.Marshal(reqBody)
		if err != nil {
			return AccessTokenInfo{}, err
		}

		req, err = http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(reqData))
		if err != nil {
			return AccessTokenInfo{}, err
		}
		req.Header.Set("Content-Type", "application/json")
	}

	// Perform the HTTP request.
	resp, err := http.DefaultClient.Do(req)
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
