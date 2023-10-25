package token

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

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
// func (atoken *AuthTokenInfo) StartRefreshAccessToken(ctx context.Context, tokenURL, appID, clientSecrent string) (err error) {
// 	// 首先，立即获取一次AccessToken
// 	tokenInfo, err := queryAccessToken(ctx, tokenURL, appID, clientSecrent)
// 	if err != nil {
// 		return err
// 	}
// 	atoken.setAuthToken(tokenInfo)
// 	fmt.Printf("获取到的token是: %s\n", tokenInfo.Token) // 输出获取到的token

// 	// 获取token的有效期（通常以秒为单位）
// 	tokenTTL := tokenInfo.ExpiresIn
// 	// 使用sync.Once保证仅启动一个goroutine进行定时刷新
// 	atoken.once.Do(func() {
// 		go func() { // 启动一个新的goroutine
// 			for {
// 				// 如果tokenTTL为0或负数，将其设置为1
// 				if tokenTTL <= 0 {
// 					tokenTTL = 1
// 				}
// 				select {
// 				case <-time.NewTimer(time.Duration(tokenTTL) * time.Second).C: // 当token过期时
// 				case upToken := <-atoken.forceUpToken: // 接收强制更新token的信号
// 					log.Warnf("recv uptoken info:%v", upToken)
// 				case <-ctx.Done(): // 当上下文结束时，退出goroutine
// 					log.Warnf("recv ctx:%v exit refreshAccessToken", ctx.Err())
// 					return
// 				}
// 				// 查询并获取新的AccessToken
// 				tokenInfo, err := queryAccessToken(ctx, tokenURL, appID, clientSecrent)
// 				if err == nil {
// 					atoken.setAuthToken(tokenInfo)
// 					fmt.Printf("获取到的token是: %s\n", tokenInfo.Token) // 输出获取到的token
// 					tokenTTL = tokenInfo.ExpiresIn
// 				} else {
// 					log.Errorf("queryAccessToken err:%v", err)
// 				}
// 			}
// 		}()
// 	})
// 	return
// }

// 测试用
func (atoken *AuthTokenInfo) StartRefreshAccessToken(ctx context.Context, tokenURL, appID, clientSecrent string) (err error) {
	// 创建一个固定的token信息
	fixedTokenInfo := AccessTokenInfo{
		Token:     "PpAPgoel0-gTeaxy-ydak0kUKxJrCSlbLcwtuPt99jCPVrahkqh3WSiIy9s63tCZnTEp4asw035u",
		ExpiresIn: 3600, // 这里假设token的有效时间是3600秒，你可以根据需要调整
	}
	atoken.setAuthToken(fixedTokenInfo)
	return nil
}

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

type queryTokenRsp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

func queryAccessToken(ctx context.Context, tokenURL, appID, clientSecrent string) (AccessTokenInfo, error) {
	method := "POST"

	defer func() {
		if err := recover(); err != nil {
			log.Errorf("queryAccessToken err:%v", err)
		}
	}()
	if tokenURL == "" {
		tokenURL = getAccessTokenURL
	}

	queryReq := queryTokenReq{
		AppID:        appID,
		ClientSecret: clientSecrent,
	}
	data, err := json.Marshal(queryReq)
	if err != nil {
		return AccessTokenInfo{}, err
	}
	payload := bytes.NewReader(data)
	log.Infof("reqdata:%v", string(data))
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	log.Infof("tokenURL:%v", tokenURL)
	req, err := http.NewRequest(method, tokenURL, payload)
	if err != nil {
		log.Errorf("NewRequest err:%v", err)
		return AccessTokenInfo{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		log.Errorf("http do err:%v", err)
		return AccessTokenInfo{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("ReadAll do err:%v", err)
		return AccessTokenInfo{}, err
	}
	log.Infof("accesstoken:%v", string(body))
	queryRsp := queryTokenRsp{}
	if err = json.Unmarshal(body, &queryRsp); err != nil {
		log.Errorf("Unmarshal err:%v", err)
		return AccessTokenInfo{}, err
	}

	rdata := AccessTokenInfo{
		Token:  queryRsp.AccessToken,
		UpTime: time.Now(),
	}
	rdata.ExpiresIn, _ = strconv.ParseInt(queryRsp.ExpiresIn, 10, 64)
	return rdata, err
}
