package v2

import (
	"bytes"
	"context"
	"log"
	"strconv"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

// CreateDirectMessage 创建私信频道
func (o *openAPIv2) CreateDirectMessage(ctx context.Context, dm *dto.DirectMessageToCreate) (*dto.DirectMessage, error) {
	resp, err := o.request(ctx).
		SetResult(dto.DirectMessage{}).
		SetBody(dm).
		Post(o.getURL(userMeDMURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.DirectMessage), nil
}

// PostDirectMessage 在私信频道内发消息
func (o *openAPIv2) PostDirectMessage(ctx context.Context,
	dm *dto.DirectMessage, msg *dto.MessageToCreate) (*dto.Message, error) {
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("guild_id", dm.GuildID).
		SetBody(msg).
		Post(o.getURL(dmsURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.Message), nil
}

// PostDirectMessageMultipart 在私信频道内使用multipart/form-data发消息
func (o *openAPIv2) PostDirectMessageMultipart(ctx context.Context, dm *dto.DirectMessage, msg *dto.MessageToCreate, fileImageData []byte) (*dto.Message, error) {
	request := o.request(ctx).SetResult(dto.Message{}).SetPathParam("guild_id", dm.GuildID)

	// 直接的字符串或数值字段
	if msg.Content != "" {
		request = request.SetFormData(map[string]string{"content": msg.Content})
	}
	if msg.MsgType != 0 {
		request = request.SetFormData(map[string]string{"msg_type": strconv.Itoa(msg.MsgType)})
	}
	if msg.Image != "" {
		request = request.SetFormData(map[string]string{"image": msg.Image})
	}
	if msg.MsgID != "" {
		request = request.SetFormData(map[string]string{"msg_id": msg.MsgID})
	}
	if msg.EventID != "" {
		request = request.SetFormData(map[string]string{"event_id": msg.EventID})
	}

	// 将需要作为JSON发送的字段序列化为JSON字符串后添加到表单
	if msg.Embed != nil {
		request = request.SetFormData(map[string]string{"embed": toJSON(msg.Embed)})
	}
	if msg.Ark != nil {
		request = request.SetFormData(map[string]string{"ark": toJSON(msg.Ark)})
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

	// 如果提供了fileImageData，则设置file_image
	if len(fileImageData) > 0 {
		request = request.SetFileReader("file_image", "filename.jpg", bytes.NewReader(fileImageData))
	}

	resp, err := request.Post(o.getURL(dmsURI))
	if err != nil {
		// 打印msg内容
		log.Printf("Message being posted: %+v\n", *msg)
		return nil, err
	}

	// 打印msg内容
	log.Printf("Message being posted: %+v\n", *msg)
	return resp.Result().(*dto.Message), nil
}

// RetractDMMessage 撤回私信消息
func (o *openAPIv2) RetractDMMessage(ctx context.Context,
	guildID, msgID string, options ...openapi.RetractMessageOption) error {
	request := o.request(ctx).
		SetPathParam("guild_id", guildID).
		SetPathParam("message_id", string(msgID))
	for _, option := range options {
		if option == openapi.RetractMessageOptionHidetip {
			request = request.SetQueryParam("hidetip", "true")
		}
	}
	_, err := request.Delete(o.getURL(dmsMessageURI))
	return err
}

// PostDMSettingGuide 发送私信设置引导, jumpGuildID为设置引导要跳转的频道ID
func (o *openAPIv2) PostDMSettingGuide(ctx context.Context,
	dm *dto.DirectMessage, jumpGuildID string) (*dto.Message, error) {
	msg := &dto.SettingGuideToCreate{
		SettingGuide: &dto.SettingGuide{
			GuildID: jumpGuildID,
		},
	}
	resp, err := o.request(ctx).
		SetResult(dto.Message{}).
		SetPathParam("guild_id", dm.GuildID).
		SetBody(msg).
		Post(o.getURL(dmSettingGuideURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*dto.Message), nil
}
