package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

type InteractionResponse struct {
	Data    string      `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

func init() {
	callapi.RegisterHandler("put_interaction", HandlePutInteraction)
}

func HandlePutInteraction(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {

	// 解析 ActionMessage 中的 Echo 字段获取 interactionID
	interactionID, ok := message.Echo.(string)
	if !ok {
		return "", fmt.Errorf("echo is not a string")
	}

	// 检查字符串是否仅包含数字 将数字形式的interactionID转换为真实的形式
	isNumeric := func(s string) bool {
		return regexp.MustCompile(`^\d+$`).MatchString(s)
	}

	if isNumeric(interactionID) && interactionID != "0" {
		// 当interactionID是字符串形式的数字时，执行转换
		var RealinteractionID string
		var err error
		if config.GetMemoryMsgid() {
			//从内存取
			RealinteractionID, _ = echo.GetCacheIDFromMemoryByRowID(interactionID)
		} else {
			RealinteractionID, err = idmap.RetrieveRowByCachev2(interactionID)
		}

		if err != nil {
			mylog.Printf("error retrieving real interactionID: %v", err)
		} else {
			// 重新赋值，RealinteractionID的类型与message.Params.interactionID兼容
			interactionID = RealinteractionID
		}
	}

	// 根据 PostType 解析出 code 的值
	var code int
	switch message.PostType {
	case "0":
		code = 0 // 成功
	case "1":
		code = 1 // 操作失败
	case "2":
		code = 2 // 操作频繁
	case "3":
		code = 3 // 重复操作
	case "4":
		code = 4 // 没有权限
	case "5":
		code = 5 // 仅管理员操作
	default:
		// 如果 PostType 不在预期范围内，可以设置一个默认值或返回错误
		return "", fmt.Errorf("invalid post type: %s", message.PostType)
	}

	// 构造请求体，包括 code
	requestBody := fmt.Sprintf(`{"code": %d}`, code)

	// 调用 PutInteraction API
	ctx := context.Background()
	err := api.PutInteraction(ctx, interactionID, requestBody)
	if err != nil {
		return "", err
	}

	var response InteractionResponse

	response.Data = ""
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	// Convert the members slice to a map
	outputMap := structToMap(response)

	mylog.Printf("put_interaction: %+v\n", outputMap)

	err = client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
	} else {
		mylog.Printf("响应put_interaction: %+v", outputMap)
	}
	//把结果从struct转换为json
	result, err := json.Marshal(response)
	if err != nil {
		mylog.Printf("Error marshaling data: %v", err)
		//todo 符合onebotv11 ws返回的错误码
		return "", nil
	}
	return string(result), nil
}
