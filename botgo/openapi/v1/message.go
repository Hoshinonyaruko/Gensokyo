package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
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

// PostFourm 发帖子
func (o *openAPI) PostFourm(ctx context.Context, channelID string, msg *dto.FourmToCreate) (*dto.Forum, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Forum{}).
		SetPathParam("channel_id", channelID).
		SetBody(msg).
		Put(o.getURL(fourmMessagesURI))
	if err != nil {
		return nil, err
	}

	return resp.Result().(*dto.Forum), nil
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
func (o *openAPI) PostGroupMessage(ctx context.Context, groupID string, msg dto.APIMessage) (*dto.GroupMessageResponse, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("group_id", groupID).
		SetBody(msg).
		Post(o.getURL(getGroupURLBySendType(msg.GetSendType())))
	if err != nil {
		return nil, err
	}
	msgType := msg.GetSendType()
	result := &dto.GroupMessageResponse{}
	switch msgType {
	case dto.RichMedia:
		result.MediaResponse = resp.Result().(*dto.MediaResponse)
	default:
		result.Message = resp.Result().(*dto.Message)
	}
	return result, nil
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
func (o *openAPI) PostC2CMessage(ctx context.Context, userID string, msg dto.APIMessage) (*dto.C2CMessageResponse, error) {
	var resp *resty.Response
	var err error

	msgType := msg.GetSendType()
	switch msgType {
	case dto.RichMedia:
		resp, err = o.request(ctx).
			SetResult(dto.MediaResponse{}). // 设置为媒体响应类型
			SetPathParam("user_id", userID).
			SetBody(msg).
			Post(o.getURL(getC2CURLBySendType(msgType)))
	default:
		resp, err = o.request(ctx).
			SetResult(dto.Message{}). // 设置为消息类型
			SetPathParam("user_id", userID).
			SetBody(msg).
			Post(o.getURL(getC2CURLBySendType(msgType)))
	}

	if err != nil {
		return nil, err
	}

	result := &dto.C2CMessageResponse{}
	switch msgType {
	case dto.RichMedia:
		result.MediaResponse = resp.Result().(*dto.MediaResponse)
	default:
		result.Message = resp.Result().(*dto.Message)
	}

	return result, nil
}
