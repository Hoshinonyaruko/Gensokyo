package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2" // resty 是一个优秀的 rest api 客户端，可以极大的减少开发基于 rest 标准接口求请求的封装工作量

	"github.com/tencent-connect/botgo/errs"
	"github.com/tencent-connect/botgo/log"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"github.com/tencent-connect/botgo/version"
)

// MaxIdleConns 默认指定空闲连接池大小
const MaxIdleConns = 3000

type openAPIv2 struct {
	token   *token.Token
	timeout time.Duration

	sandbox     bool   // 请求沙箱环境
	debug       bool   // debug 模式，调试sdk时候使用
	lastTraceID string // lastTraceID id

	restyClient *resty.Client // resty client 复用
}

// Setup 注册
func Setup() {
	openapi.Register(openapi.APIv2, &openAPIv2{})
}

// Version 创建当前版本
func (o *openAPIv2) Version() openapi.APIVersion {
	return openapi.APIv2
}

// TraceID 获取 lastTraceID id
func (o *openAPIv2) TraceID() string {
	return o.lastTraceID
}

// Setup 生成一个实例
func (o *openAPIv2) Setup(token *token.Token, inSandbox bool) openapi.OpenAPI {
	// 打印传入的令牌以进行调试
	//log.Printf("Setup called with token: %+v\n", token)

	api := &openAPIv2{
		token:   token,
		timeout: 3 * time.Second,
		sandbox: inSandbox,
	}
	api.setupClient() // 初始化可复用的 client
	return api
}

// WithTimeout 设置请求接口超时时间
func (o *openAPIv2) WithTimeout(duration time.Duration) openapi.OpenAPI {
	o.restyClient.SetTimeout(duration)
	return o
}

// Transport 透传请求
func (o *openAPIv2) Transport(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	resp, err := o.request(ctx).SetBody(body).Execute(method, url)
	return resp.Body(), err
}

// 初始化 client
func (o *openAPIv2) setupClient() {
	o.restyClient = resty.New().
		SetTransport(createTransport(nil, MaxIdleConns)). // 自定义 transport
		SetLogger(log.DefaultLogger).
		SetDebug(o.debug).
		SetTimeout(o.timeout).
		SetAuthScheme(string(o.token.Type)).
		SetHeader("User-Agent", version.String()).
		SetHeader("X-Union-Appid", fmt.Sprint(o.token.GetAppID())).
		SetPreRequestHook(
			func(client *resty.Client, request *http.Request) error {
				// 执行请求前过滤器
				// 由于在 `OnBeforeRequest` 的时候，request 还没生成，所以 filter 不能使用，所以放到 `PreRequestHook`
				return openapi.DoReqFilterChains(request, nil)
			},
		).
		OnBeforeRequest(
			func(c *resty.Client, r *resty.Request) error {
				// 设置授权方案为 "QQBot"
				c.SetAuthScheme("QQBot")
				c.SetAuthToken(o.token.GetAccessToken())
				return nil
			},
		).
		// 设置请求之后的钩子，打印日志，判断状态码
		OnAfterResponse(
			func(client *resty.Client, resp *resty.Response) error {
				log.Infof("%v", respInfo(resp))
				// 执行请求后过滤器
				if err := openapi.DoRespFilterChains(resp.Request.RawRequest, resp.RawResponse); err != nil {
					return err
				}
				traceID := resp.Header().Get(openapi.TraceIDKey)
				o.lastTraceID = traceID
				// 非成功含义的状态码，需要返回 error 供调用方识别
				if !openapi.IsSuccessStatus(resp.StatusCode()) {
					return errs.New(resp.StatusCode(), string(resp.Body()), traceID)
				}
				return nil
			},
		)
}

// request 每个请求，都需要创建一个 request
func (o *openAPIv2) request(ctx context.Context) *resty.Request {
	return o.restyClient.R().SetContext(ctx)
}

// respInfo 用于输出日志的时候格式化数据
func respInfo(resp *resty.Response) string {
	bodyJSON, _ := json.Marshal(resp.Request.Body)
	return fmt.Sprintf(
		"[OPENAPI]%v %v, traceID:%v, status:%v, elapsed:%v req: %v, resp: %v",
		resp.Request.Method,
		resp.Request.URL,
		resp.Header().Get(openapi.TraceIDKey),
		resp.Status(),
		resp.Time(),
		string(bodyJSON),
		string(resp.Body()),
	)
}

// reqInfo 用于请求日志格式化
func reqInfo(req *resty.Request) string {
	bodyJSON, _ := json.Marshal(req)
	return fmt.Sprintf("[OPENAPI]%v %v data:%v", req.Method, req.URL, string(bodyJSON))
}

func createTransport(localAddr net.Addr, idleConns int) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}
	if localAddr != nil {
		dialer.LocalAddr = localAddr
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          idleConns,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   idleConns,
		MaxConnsPerHost:       idleConns,
	}
}
