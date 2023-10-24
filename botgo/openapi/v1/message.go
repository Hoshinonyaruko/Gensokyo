package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tidwall/gjson"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/errs"
	"github.com/tencent-connect/botgo/openapi"
)

// Message 拉取单条消息
func (o *openAPI) Message(ctx context.Context, channelID string, messageID string) (*dto.Message, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("channel_id", channelID).
		SetPathParam("message_id", messageID).
		Get(o.getURL(messageURI))
	if err != nil {
		return nil, err
	}

	// 兼容处理
	result := resp.Result().(*dto.Message)
	if result.ID == "" {
		body := gjson.Get(resp.String(), "message")
		if err := json.Unmarshal([]byte(body.String()), result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Messages 拉取消息列表
func (o *openAPI) Messages(ctx context.Context, channelID string, pager *dto.MessagesPager) ([]*dto.Message, error) {
	if pager == nil {
		return nil, errs.ErrPagerIsNil
	}
	resp, err := o.request(ctx).
		SetPathParam("channel_id", channelID).
		SetQueryParams(pager.QueryParams()).
		Get(o.getURL(messagesURI))
	if err != nil {
		return nil, err
	}

	messages := make([]*dto.Message, 0)
	if err := json.Unmarshal(resp.Body(), &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// PostMessage 发消息
func (o *openAPI) PostMessage(ctx context.Context, channelID string, msg *dto.MessageToCreate) (*dto.Message, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("channel_id", channelID).
		SetBody(msg).
		Post(o.getURL(messagesURI))
	if err != nil {
		return nil, err
	}

	return resp.Result().(*dto.Message), nil
}

// PostMessageMultipart 发送消息使用multipart/form-data
func (o *openAPI) PostMessageMultipart(ctx context.Context, channelID string, msg *dto.MessageToCreate, fileImageData []byte) (*dto.Message, error) {
	request := o.request(ctx).SetResult(dto.Message{}).SetPathParam("channel_id", channelID)

	// 将MessageToCreate的字段转为multipart form data
	if msg.Content != "" {
		request = request.SetFormData(map[string]string{"content": msg.Content})
	}
	if msg.MsgType != 0 {
		request = request.SetFormData(map[string]string{"msg_type": strconv.Itoa(msg.MsgType)})
	}
	if msg.Embed != nil {
		request = request.SetFormData(map[string]string{"embed": toJSON(msg.Embed)})
	}
	if msg.Ark != nil {
		request = request.SetFormData(map[string]string{"ark": toJSON(msg.Ark)})
	}
	if msg.Image != "" {
		request = request.SetFormData(map[string]string{"image": msg.Image})
	}
	if msg.MsgID != "" {
		request = request.SetFormData(map[string]string{"msg_id": msg.MsgID})
	}
	if msg.MessageReference != nil {
		request = request.SetFormData(map[string]string{"message_reference": toJSON(msg.MessageReference)})
	}
	if msg.Markdown != nil {
		request = request.SetFormData(map[string]string{"markdown": toJSON(msg.Markdown)})
	}
	if msg.Keyboard != nil {
		request = request.SetFormData(map[string]string{"keyboard": toJSON(msg.Keyboard)})
	}
	if msg.EventID != "" {
		request = request.SetFormData(map[string]string{"event_id": msg.EventID})
	}

	// 如果提供了fileImageData，则设置file_image
	if len(fileImageData) > 0 {
		request = request.SetFileReader("file_image", "filename.jpg", bytes.NewReader(fileImageData))
	}

	resp, err := request.Post(o.getURL(messagesURI))
	if err != nil {
		return nil, err
	}

	return resp.Result().(*dto.Message), nil
}

// 辅助函数：序列化为JSON
func toJSON(obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// PatchMessage 编辑消息
func (o *openAPI) PatchMessage(ctx context.Context,
	channelID string, messageID string, msg *dto.MessageToCreate) (*dto.Message, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("channel_id", channelID).
		SetPathParam("message_id", messageID).
		SetBody(msg).
		Patch(o.getURL(messageURI))
	if err != nil {
		return nil, err
	}

	return resp.Result().(*dto.Message), nil
}

// RetractMessage 撤回消息
func (o *openAPI) RetractMessage(ctx context.Context,
	channelID, msgID string, options ...openapi.RetractMessageOption) error {
	request := o.request(ctx).
		SetPathParam("channel_id", channelID).
		SetPathParam("message_id", string(msgID))
	for _, option := range options {
		if option == openapi.RetractMessageOptionHidetip {
			request = request.SetQueryParam("hidetip", "true")
		}
	}
	_, err := request.Delete(o.getURL(messageURI))
	return err
}

// PostSettingGuide 发送设置引导消息, atUserID为要at的用户
func (o *openAPI) PostSettingGuide(ctx context.Context,
	channelID string, atUserIDs []string) (*dto.Message, error) {
	var content string
	for _, userID := range atUserIDs {
		content += fmt.Sprintf("<@%s>", userID)
	}
	msg := &dto.SettingGuideToCreate{
		Content: content,
	}
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("channel_id", channelID).
		SetBody(msg).
		Post(o.getURL(settingGuideURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.Message), nil
}

func getGroupURLBySendType(msgType dto.SendType) uri {
	switch msgType {
	case dto.RichMedia:
		return groupRichMediaURI
	default:
		return groupMessagesURI
	}
}

// PostGroupMessage 回复群消息
func (o *openAPI) PostGroupMessage(ctx context.Context, groupID string, msg dto.APIMessage) (*dto.Message, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("group_id", groupID).
		SetBody(msg).
		Post(o.getURL(getGroupURLBySendType(msg.GetSendType())))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.Message), nil
}

func getC2CURLBySendType(msgType dto.SendType) uri {
	switch msgType {
	case dto.RichMedia:
		return c2cRichMediaURI
	default:
		return c2cMessagesURI
	}
}

// PostC2CMessage 回复C2C消息
func (o *openAPI) PostC2CMessage(ctx context.Context, userID string, msg dto.APIMessage) (*dto.Message, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("user_id", userID).
		SetBody(msg).
		Post(o.getURL(getC2CURLBySendType(msg.GetSendType())))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.Message), nil
}
