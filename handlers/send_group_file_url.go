package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

const (
	sendGroupFileURLAction     = "send_group_file_url"
	sendGroupFileURLNoticeType = "group_file_url_resp"
)

func init() {
	callapi.RegisterHandler(
		sendGroupFileURLAction,
		HandleSendGroupFileURL,
	)
}

// HandleSendGroupFileURL 通过 URL 向 QQ 群发送普通文件。
//
// 限制：
//   - 必须开启 string_ob11
//   - group_id 必须直接传入 32 位群 OpenID
//   - 仅支持 URL
//   - 不支持分片、本地文件、base64
func HandleSendGroupFileURL(
	client callapi.Client,
	_ openapi.OpenAPI,
	apiv2 openapi.OpenAPI,
	message callapi.ActionMessage,
) (string, error) {
	groupID, _ := message.Params.GroupID.(string)
	groupID = strings.TrimSpace(groupID)

	fileURL := strings.TrimSpace(message.Params.URL)

	if !config.GetStringOb11() {
		return pushGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			nil,
			"string_ob11 is disabled; send_group_file_url is unsupported",
		)
	}

	if len(groupID) != 32 {
		return pushGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			nil,
			"group_id must be a 32-character group OpenID",
		)
	}

	if !validGroupFileURL(fileURL) {
		return pushGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			nil,
			"url must be a valid http or https URL",
		)
	}

	if apiv2 == nil {
		return pushGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			nil,
			"QQ OpenAPI v2 client is unavailable",
		)
	}

	request := &dto.RichMediaMessage{
		FileType:   4,
		URL:        fileURL,
		SrvSendMsg: true,
	}

	result, err := apiv2.PostGroupMessage(
		context.Background(),
		groupID,
		request,
	)
	if err != nil {
		mylog.Printf(
			"send_group_file_url failed: group_id=%s, url=%s, err=%v",
			groupID,
			fileURL,
			err,
		)

		return pushGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			nil,
			fmt.Sprintf("QQ API file upload failed: %v", err),
		)
	}

	if result == nil || result.MediaResponse == nil {
		mylog.Printf(
			"send_group_file_url returned empty media response: group_id=%s",
			groupID,
		)

		return pushGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			nil,
			"QQ API returned an empty media response",
		)
	}

	media := result.MediaResponse

	if media.Ret != 0 {
		return pushGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			nil,
			fmt.Sprintf("QQ API returned ret=%d", media.Ret),
		)
	}

	mylog.Printf(
		"send_group_file_url success: group_id=%s, message_id=%s, file_uuid=%s",
		groupID,
		media.ID,
		media.FileUUID,
	)

	return pushGroupFileURLNotice(
		client,
		message,
		groupID,
		fileURL,
		media,
		"",
	)
}

func validGroupFileURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)

	return (scheme == "http" || scheme == "https") &&
		parsed.Host != ""
}

func pushGroupFileURLNotice(
	client callapi.Client,
	message callapi.ActionMessage,
	groupID string,
	fileURL string,
	media *dto.MediaResponse,
	errorMessage string,
) (string, error) {
	if client == nil {
		return "", fmt.Errorf(
			"send_group_file_url: websocket client is nil",
		)
	}

	status := "ok"
	retCode := 0

	if errorMessage != "" {
		status = "failed"
		retCode = 100
	}

	notice := map[string]interface{}{
		"post_type":   "notice",
		"notice_type": sendGroupFileURLNoticeType,
		"status":      status,
		"retcode":     retCode,
		"group_id":    groupID,
		"url":         fileURL,
		"error_msg":   errorMessage,
		"echo":        message.Echo,
		"time":        time.Now().Unix(),
		"self_id":     config.GetAppID(),
	}

	if media != nil {
		notice["message_id"] = media.ID
		notice["file_uuid"] = media.FileUUID
		notice["file_info"] = media.FileInfo
		notice["ttl"] = media.TTL
	}

	if err := client.SendMessage(notice); err != nil {
		return "", fmt.Errorf(
			"send group_file_url_resp notice: %w",
			err,
		)
	}

	return "{}", nil
}
