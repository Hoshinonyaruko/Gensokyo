package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/openapi"
)

func init() {
	callapi.RegisterHandler("delete_msg", DeleteMsg)
}

func DeleteMsg(client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI, message callapi.ActionMessage) (string, error) {
	var RealMsgID string
	var err error

	// 如果从内存取
	if config.GetMemoryMsgid() {
		//还原msgid
		RealMsgID, _ = echo.GetCacheIDFromMemoryByRowID(message.Params.MessageID.(string))
	} else {
		//还原msgid
		RealMsgID, err = idmap.RetrieveRowByCachev2(message.Params.MessageID.(string))
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
	}

	//重新赋值
	message.Params.MessageID = RealMsgID
	//撤回频道信息
	if message.Params.ChannelID != nil && message.Params.ChannelID != "" {
		var RChannelID string
		var err error
		// 使用RetrieveRowByIDv2还原真实的ChannelID
		RChannelID, err = idmap.RetrieveRowByIDv2(message.Params.ChannelID.(string))
		if err != nil {
			mylog.Printf("error retrieving real RChannelID: %v", err)
		}
		message.Params.ChannelID = RChannelID
		err = api.RetractMessage(context.TODO(), message.Params.ChannelID.(string), message.Params.MessageID.(string), openapi.RetractMessageOptionHidetip)
		if err != nil {
			fmt.Println("Error retracting channel message:", err)
		}

	}

	//撤回频道私信
	if message.Params.GuildID != nil && message.Params.GuildID != "" {
		//这里很复杂 要取的话需要调用internal-api 根据情况还原，虚拟成群就用群（channel-id）还原完整channel-id，
		//然后internal-api读配置获取guild-id ，虚拟成私信就用userid还原完整userid，然后读channel-id然后读guild-id
		//因为GuildID本身不直接出现在ob11事件里。
		err := api.RetractDMMessage(context.TODO(), message.Params.GuildID.(string), message.Params.MessageID.(string), openapi.RetractMessageOptionHidetip)
		if err != nil {
			fmt.Println("Error retracting DM message:", err)
		}

	}

	//撤回群信息
	if message.Params.GroupID != nil && message.Params.GroupID != "" {
		var originalGroupID string
		originalGroupID, err := idmap.RetrieveRowByIDv2(message.Params.GroupID.(string))
		if err != nil {
			mylog.Printf("Error retrieving original GroupID: %v", err)
		}
		message.Params.GroupID = originalGroupID
		err = api.RetractGroupMessage(context.TODO(), message.Params.GroupID.(string), message.Params.MessageID.(string), openapi.RetractMessageOptionHidetip)
		if err != nil {
			fmt.Println("Error retracting group message:", err)
		}

	}

	//撤回C2C私信消息列表
	if message.Params.UserID != nil && message.Params.UserID != "" {
		var UserID string
		//还原真实的userid
		UserID, err := idmap.RetrieveRowByIDv2(message.Params.UserID.(string))
		if err != nil {
			mylog.Printf("Error reading config: %v", err)
			return "", nil
		}
		message.Params.UserID = UserID
		err = api.RetractC2CMessage(context.TODO(), message.Params.UserID.(string), message.Params.MessageID.(string), openapi.RetractMessageOptionHidetip)
		if err != nil {
			fmt.Println("Error retracting C2C message:", err)
		}

	}

	var response GetStatusResponse
	response.Message = ""
	response.RetCode = 0
	response.Status = "ok"
	response.Echo = message.Echo

	outputMap := structToMap(response)

	mylog.Printf("delete_msg: %+v\n", outputMap)

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
