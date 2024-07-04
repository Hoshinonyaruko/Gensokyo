package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

type GetAvatarResponse struct {
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Echo    interface{} `json:"echo"`
	UserID  int64       `json:"user_id"`
}

func init() {
	callapi.RegisterHandler("get_avatar", GetAvatar)
}

func GetAvatar(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {

	var response GetAvatarResponse
	var originalUserID string
	var err error

	if config.GetIdmapPro() {
		// 如果UserID不是nil且配置为使用Pro版本，则调用RetrieveRowByIDv2Pro
		_, originalUserID, err = idmap.RetrieveRowByIDv2Pro(message.Params.GroupID.(string), message.Params.UserID.(string))
		if err != nil {
			mylog.Printf("Error1 retrieving original GroupID: %v", err)
			_, originalUserID, err = idmap.RetrieveRowByIDv2Pro("690426430", message.Params.UserID.(string))
			if err != nil {
				mylog.Printf("Error reading private originalUserID: %v", err)
			}
		}
	} else {
		originalUserID, err = idmap.RetrieveRowByIDv2(message.Params.UserID.(string))
		if err != nil {
			mylog.Printf("Error retrieving original UserID: %v", err)
		}
	}

	avatarurl, _ := GenerateAvatarURLV2(originalUserID)

	useridstr := message.Params.UserID.(string)

	response.Message = avatarurl
	response.RetCode = 0
	response.Echo = message.Echo
	response.UserID, _ = strconv.ParseInt(useridstr, 10, 64)

	outputMap := structToMap(response)

	mylog.Printf("get_avatar: %+v\n", outputMap)

	err = client.SendMessage(outputMap)
	if err != nil {
		mylog.Printf("Error sending message via client: %v", err)
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

// GenerateAvatarURLV2 生成根据32位ID 和 Appid 组合的 新QQ 头像 URL
func GenerateAvatarURLV2(openid string) (string, error) {

	appidstr := config.GetAppIDStr()
	// 构建并返回 URL
	return fmt.Sprintf("https://q.qlogo.cn/qqapp/%s/%s/640", appidstr, openid), nil
}
