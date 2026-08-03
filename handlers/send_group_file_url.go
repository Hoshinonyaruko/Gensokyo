package handlers

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo/callapi"
	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/echo"
	"github.com/hoshinonyaruko/gensokyo/mylog"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

const (
	sendGroupFileURLAction     = "send_group_file_url"
	sendGroupFileURLNoticeType = "group_file_url_resp"

	qqOpenAPIBaseURL = "https://api.sgroup.qq.com"

	// QQ 群文件硬限制。
	maxGroupFileSize int64 = 200 * 1024 * 1024

	// 腾讯要求计算文件前 10002432 字节的 MD5。
	groupFileMD5PrefixSize int64 = 10002432

	// 客户端当前等待回执 20 秒。
	// 服务端需要提前结束，以便仍有时间发送失败 notice。
	groupFileWorkflowTimeout = 18 * time.Second
)

// groupUploadPrepareRequest 对应群文件预上传请求。
type groupUploadPrepareRequest struct {
	FileType int    `json:"file_type"`
	FileSize string `json:"file_size"`
	FileName string `json:"file_name"`
	MD5      string `json:"md5"`
	SHA1     string `json:"sha1"`
	MD510M   string `json:"md5_10m"`
}

// groupUploadPrepareResponse 对应群文件预上传响应。
type groupUploadPrepareResponse struct {
	UploadID     string            `json:"upload_id"`
	BlockSize    string            `json:"block_size"`
	Parts        []groupUploadPart `json:"parts"`
	UploadConfig struct {
		Concurrency  int `json:"concurrency"`
		RetryTimeout int `json:"retry_timeout"`
		RetryDelay   int `json:"retry_delay"`
	} `json:"upload_config"`
}

// groupUploadPart 是腾讯返回的单个分片上传任务。
type groupUploadPart struct {
	Index        int    `json:"index"`
	PresignedURL string `json:"presigned_url"`
	BlockSize    string `json:"block_size"`
}

// groupUploadPartFinishRequest 通知腾讯单个分片已经上传完成。
type groupUploadPartFinishRequest struct {
	UploadID  string `json:"upload_id"`
	PartIndex int    `json:"part_index"`
	BlockSize string `json:"block_size"`
	MD5       string `json:"md5"`
}

// groupFileMergeMessage 只用于当前 Handler 的分片合并请求。
//
// 独立定义是为了：
//  1. 不修改全局 dto.RichMediaMessage；
//  2. 明确发送 srv_send_msg=false；
//  3. 继续复用 PostGroupMessage 对 /files 的路由。
type groupFileMergeMessage struct {
	FileType   uint64 `json:"file_type"`
	SrvSendMsg bool   `json:"srv_send_msg"`
	FileName   string `json:"file_name"`
	UploadID   string `json:"upload_id"`
}

func (groupFileMergeMessage) GetEventID() string {
	return ""
}

func (groupFileMergeMessage) GetSendType() dto.SendType {
	return dto.RichMedia
}

func init() {
	callapi.RegisterHandler(
		sendGroupFileURLAction,
		HandleSendGroupFileURL,
	)
}

// HandleSendGroupFileURL 执行完整的群文件上传和被动发送流程：
//
//  1. Gensokyo 下载源文件；
//  2. 请求 upload_prepare；
//  3. 分片 PUT 到腾讯预签名地址；
//  4. 每片调用 upload_part_finish；
//  5. 使用 upload_id 调 /files 合并；
//  6. 从 lazymsgid 获取用户本次指令对应的 msg_id；
//  7. 使用 file_info + msg_id 被动发送群文件；
//  8. 发送 group_file_url_resp notice；
//  9. 删除临时文件。
func HandleSendGroupFileURL(
	client callapi.Client,
	_ openapi.OpenAPI,
	apiv2 openapi.OpenAPI,
	message callapi.ActionMessage,
) (string, error) {
	groupID, _ := message.Params.GroupID.(string)
	userID, _ := message.Params.UserID.(string)

	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	fileURL := strings.TrimSpace(message.Params.URL)

	// 统一失败出口。
	// 所有失败都会立即向客户端发送 failed notice，
	// 避免客户端一直等待 20 秒后才超时。
	fail := func(
		media *dto.MediaResponse,
		workflowErr error,
	) (string, error) {
		if workflowErr == nil {
			workflowErr = fmt.Errorf(
				"unknown group file error",
			)
		}

		mylog.Printf(
			"send_group_file_url failed: group_id=%s user_id=%s url=%s err=%v",
			groupID,
			userID,
			fileURL,
			workflowErr,
		)

		noticeErr := sendGroupFileURLNotice(
			client,
			message,
			groupID,
			fileURL,
			media,
			"",
			workflowErr.Error(),
		)
		if noticeErr != nil {
			return "", noticeErr
		}

		return "{}", nil
	}

	if !config.GetStringOb11() {
		return fail(
			nil,
			fmt.Errorf(
				"string_ob11 is disabled",
			),
		)
	}

	if len(groupID) != 32 {
		return fail(
			nil,
			fmt.Errorf(
				"group_id is invalid",
			),
		)
	}

	// 必须携带触发上传指令的用户 OpenID，
	// 用于获取 groupID + userID 精确 lazymsgid。
	if len(userID) != 32 {
		return fail(
			nil,
			fmt.Errorf(
				"user_id is invalid",
			),
		)
	}

	if !validGroupFileURL(fileURL) {
		return fail(
			nil,
			fmt.Errorf(
				"url is invalid",
			),
		)
	}

	if apiv2 == nil {
		return fail(
			nil,
			fmt.Errorf(
				"QQ OpenAPI v2 client is unavailable",
			),
		)
	}

	if !config.GetLazyMessageId() {
		return fail(
			nil,
			fmt.Errorf(
				"lazy_message_id is disabled",
			),
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		groupFileWorkflowTimeout,
	)
	defer cancel()

	media, sentMessageID, err :=
		sendGroupFileByChunkUpload(
			ctx,
			apiv2,
			groupID,
			userID,
			fileURL,
		)
	if err != nil {
		return fail(media, err)
	}

	// 只有最终被动消息明确发送成功，
	// 才回调 status=ok。
	if err := sendGroupFileURLNotice(
		client,
		message,
		groupID,
		fileURL,
		media,
		sentMessageID,
		"",
	); err != nil {
		return "", err
	}

	return "{}", nil
}

func sendGroupFileByChunkUpload(
	ctx context.Context,
	apiv2 openapi.OpenAPI,
	groupID string,
	userID string,
	fileURL string,
) (*dto.MediaResponse, string, error) {
	// 第一步：Gensokyo 自己下载源文件。
	tempFile, fileName, fileSize, err :=
		downloadGroupFile(
			ctx,
			fileURL,
		)
	if err != nil {
		return nil, "", fmt.Errorf(
			"download source file: %w",
			err,
		)
	}

	tempPath := tempFile.Name()

	// Windows 必须先 Close，再 Remove。
	defer func() {
		if err := tempFile.Close(); err != nil {
			mylog.Printf(
				"close group file temp failed: path=%s err=%v",
				tempPath,
				err,
			)
		}

		if err := os.Remove(tempPath); err != nil {
			if !os.IsNotExist(err) {
				mylog.Printf(
					"remove group file temp failed: path=%s err=%v",
					tempPath,
					err,
				)
			}
		} else {
			mylog.Printf(
				"group file temp removed: path=%s",
				tempPath,
			)
		}
	}()

	mylog.Printf(
		"group file downloaded: name=%s size=%d path=%s",
		fileName,
		fileSize,
		tempPath,
	)

	// 第二步：计算预上传要求的三个哈希。
	fileMD5, fileSHA1, fileMD510M, err :=
		calculateGroupFileHashes(
			ctx,
			tempFile,
			fileSize,
		)
	if err != nil {
		return nil, "", fmt.Errorf(
			"calculate file hashes: %w",
			err,
		)
	}

	// 第三步：请求 upload_prepare。
	prepare, err := prepareGroupFileUpload(
		ctx,
		apiv2,
		groupID,
		fileName,
		fileSize,
		fileMD5,
		fileSHA1,
		fileMD510M,
	)
	if err != nil {
		return nil, "", fmt.Errorf(
			"prepare chunk upload: %w",
			err,
		)
	}

	firstPartIndex := -1
	if len(prepare.Parts) > 0 {
		firstPartIndex = prepare.Parts[0].Index
	}

	mylog.Printf(
		"group file upload prepared: upload_id=%s block_size=%s parts=%d first_index=%d concurrency=%d",
		prepare.UploadID,
		prepare.BlockSize,
		len(prepare.Parts),
		firstPartIndex,
		prepare.UploadConfig.Concurrency,
	)

	// 第四步：逐片 PUT，并调用 upload_part_finish。
	if err := uploadGroupFileParts(
		ctx,
		apiv2,
		groupID,
		tempFile,
		fileSize,
		prepare,
	); err != nil {
		return nil, "", fmt.Errorf(
			"upload file parts: %w",
			err,
		)
	}

	// 第五步：携带 upload_id 调 /files 合并。
	//
	// 使用局部消息类型，保证 JSON 中明确存在：
	// "srv_send_msg": false
	mergeRequest := &groupFileMergeMessage{
		FileType:   4,
		SrvSendMsg: false,
		FileName:   fileName,
		UploadID:   prepare.UploadID,
	}

	mergeResult, err := apiv2.PostGroupMessage(
		ctx,
		groupID,
		mergeRequest,
	)
	if err != nil {
		return nil, "", fmt.Errorf(
			"merge uploaded file: %w",
			err,
		)
	}

	if mergeResult == nil ||
		mergeResult.MediaResponse == nil {
		return nil, "", fmt.Errorf(
			"merge returned empty media response",
		)
	}

	media := mergeResult.MediaResponse

	if strings.TrimSpace(media.FileInfo) == "" {
		return media, "", fmt.Errorf(
			"merge returned empty file_info",
		)
	}

	mylog.Printf(
		"group file merged: upload_id=%s file_uuid=%s ttl=%d",
		prepare.UploadID,
		media.FileUUID,
		media.TTL,
	)

	// 第六步：文件上传及合并成功后，才消费 lazymsgid。
	//
	// 不在下载、哈希、分片阶段提前取出，
	// 避免文件上传失败时浪费一次被动消息机会。
	replyMessageID := echo.GetLazyMessagesIdv2(
		groupID,
		userID,
	)

	mylog.Printf(
		"send_group_file_url lazy msg_id: group_id=%s user_id=%s msg_id=%s",
		groupID,
		userID,
		replyMessageID,
	)

	if replyMessageID == "" {
		return media, "", fmt.Errorf(
			"no passive message ID is available",
		)
	}

	// 2000 是主动消息标记。
	// 当前功能禁止回退为主动消息。
	if replyMessageID == "2000" {
		return media, "", fmt.Errorf(
			"lazy message ID is active-message marker",
		)
	}

	// 第七步：使用 file_info + lazymsgid 发送被动文件消息。
	//
	// 这里沿用 Gensokyo 现有 MsgSeq 管理方式：
	// 先取当前值，发送使用当前值，再保存下一值。
	msgSeq := echo.GetMappingSeq(
		replyMessageID,
	)
	echo.AddMappingSeq(
		replyMessageID,
		msgSeq+1,
	)

	sendRequest := &dto.MessageToCreate{
		MsgType: 7,
		Media: dto.Media{
			FileInfo: media.FileInfo,
		},
		MsgID:     replyMessageID,
		MsgSeq:    msgSeq,
		Timestamp: time.Now().Unix(),
	}

	sendResult, err := apiv2.PostGroupMessage(
		ctx,
		groupID,
		sendRequest,
	)
	if err != nil {
		return media, "", fmt.Errorf(
			"send passive group file: %w",
			err,
		)
	}

	if sendResult == nil ||
		sendResult.Message == nil {
		return media, "", fmt.Errorf(
			"passive send returned empty response",
		)
	}

	sentMessageID := strings.TrimSpace(
		sendResult.Message.ID,
	)
	if sentMessageID == "" {
		return media, "", fmt.Errorf(
			"passive send returned empty message ID",
		)
	}

	mylog.Printf(
		"send_group_file_url success: group_id=%s user_id=%s reply_msg_id=%s sent_msg_id=%s file_uuid=%s",
		groupID,
		userID,
		replyMessageID,
		sentMessageID,
		media.FileUUID,
	)

	return media, sentMessageID, nil
}

// downloadGroupFile 下载源文件到操作系统临时目录。
func downloadGroupFile(
	ctx context.Context,
	rawURL string,
) (*os.File, string, int64, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		return nil, "", 0, err
	}

	client := &http.Client{
		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {
			if len(via) >= 5 {
				return fmt.Errorf(
					"too many redirects",
				)
			}

			if !validGroupFileURL(
				req.URL.String(),
			) {
				return fmt.Errorf(
					"redirected to invalid URL",
				)
			}

			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", 0, fmt.Errorf(
			"source returned status %s",
			resp.Status,
		)
	}

	if resp.ContentLength > maxGroupFileSize {
		return nil, "", 0, fmt.Errorf(
			"source file exceeds 200MB",
		)
	}

	fileName := groupFileNameFromURL(
		rawURL,
	)

	tempFile, err := os.CreateTemp(
		"",
		"gensokyo-group-file-*",
	)
	if err != nil {
		return nil, "", 0, err
	}

	success := false

	// downloadGroupFile 自身失败时负责清理。
	// 返回成功后由上层统一清理。
	defer func() {
		if success {
			return
		}

		tempPath := tempFile.Name()
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	written, err := io.Copy(
		tempFile,
		io.LimitReader(
			resp.Body,
			maxGroupFileSize+1,
		),
	)
	if err != nil {
		return nil, "", 0, err
	}

	if written > maxGroupFileSize {
		return nil, "", 0, fmt.Errorf(
			"source file exceeds 200MB",
		)
	}

	if written <= 0 {
		return nil, "", 0, fmt.Errorf(
			"source file is empty",
		)
	}

	if _, err := tempFile.Seek(
		0,
		io.SeekStart,
	); err != nil {
		return nil, "", 0, err
	}

	success = true

	return tempFile, fileName, written, nil
}

// groupFileNameFromURL 从 URL 提取最终文件名。
func groupFileNameFromURL(
	rawURL string,
) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "group_file"
	}

	fileName := pathpkg.Base(
		parsed.Path,
	)

	if decoded, err := url.PathUnescape(
		fileName,
	); err == nil {
		fileName = decoded
	}

	return sanitizeGroupFileName(
		fileName,
	)
}

// sanitizeGroupFileName 清理文件名中的路径和控制字符。
func sanitizeGroupFileName(
	fileName string,
) string {
	fileName = strings.NewReplacer(
		"\r", "_",
		"\n", "_",
		"/", "_",
		"\\", "_",
		"\x00", "",
	).Replace(fileName)

	fileName = strings.TrimSpace(
		fileName,
	)

	switch fileName {
	case "", ".", "..":
		return "group_file"
	default:
		return fileName
	}
}

// calculateGroupFileHashes 一次遍历文件，同时计算：
//
//   - 整体 MD5
//   - 整体 SHA1
//   - 前 10002432 字节 MD5
func calculateGroupFileHashes(
	ctx context.Context,
	file *os.File,
	fileSize int64,
) (
	fileMD5 string,
	fileSHA1 string,
	fileMD510M string,
	err error,
) {
	if _, err := file.Seek(
		0,
		io.SeekStart,
	); err != nil {
		return "", "", "", err
	}

	wholeMD5 := md5.New()
	wholeSHA1 := sha1.New()
	prefixMD5 := md5.New()

	prefixRemaining := fileSize
	if prefixRemaining > groupFileMD5PrefixSize {
		prefixRemaining = groupFileMD5PrefixSize
	}

	buffer := make(
		[]byte,
		256*1024,
	)

	var totalRead int64

	for {
		select {
		case <-ctx.Done():
			return "", "", "", ctx.Err()
		default:
		}

		n, readErr := file.Read(buffer)

		if n > 0 {
			chunk := buffer[:n]

			if _, err := wholeMD5.Write(
				chunk,
			); err != nil {
				return "", "", "", err
			}

			if _, err := wholeSHA1.Write(
				chunk,
			); err != nil {
				return "", "", "", err
			}

			if prefixRemaining > 0 {
				prefixBytes := int64(n)
				if prefixBytes > prefixRemaining {
					prefixBytes = prefixRemaining
				}

				if _, err := prefixMD5.Write(
					chunk[:int(prefixBytes)],
				); err != nil {
					return "", "", "", err
				}

				prefixRemaining -= prefixBytes
			}

			totalRead += int64(n)
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			return "", "", "", readErr
		}
	}

	if totalRead != fileSize {
		return "", "", "", fmt.Errorf(
			"unexpected file size while hashing: expected=%d actual=%d",
			fileSize,
			totalRead,
		)
	}

	if _, err := file.Seek(
		0,
		io.SeekStart,
	); err != nil {
		return "", "", "", err
	}

	return hex.EncodeToString(
			wholeMD5.Sum(nil),
		),
		hex.EncodeToString(
			wholeSHA1.Sum(nil),
		),
		hex.EncodeToString(
			prefixMD5.Sum(nil),
		),
		nil
}

// prepareGroupFileUpload 请求腾讯生成 upload_id 和预签名分片 URL。
func prepareGroupFileUpload(
	ctx context.Context,
	apiv2 openapi.OpenAPI,
	groupID string,
	fileName string,
	fileSize int64,
	fileMD5 string,
	fileSHA1 string,
	fileMD510M string,
) (*groupUploadPrepareResponse, error) {
	request := &groupUploadPrepareRequest{
		FileType: 4,
		FileSize: strconv.FormatInt(
			fileSize,
			10,
		),
		FileName: fileName,
		MD5:      fileMD5,
		SHA1:     fileSHA1,
		MD510M:   fileMD510M,
	}

	requestURL := fmt.Sprintf(
		"%s/v2/groups/%s/upload_prepare",
		qqOpenAPIBaseURL,
		url.PathEscape(groupID),
	)

	responseBody, err := apiv2.Transport(
		ctx,
		http.MethodPost,
		requestURL,
		request,
	)
	if err != nil {
		return nil, err
	}

	var response groupUploadPrepareResponse

	if err := json.Unmarshal(
		responseBody,
		&response,
	); err != nil {
		return nil, fmt.Errorf(
			"decode prepare response: %w",
			err,
		)
	}

	response.UploadID = strings.TrimSpace(
		response.UploadID,
	)
	response.BlockSize = strings.TrimSpace(
		response.BlockSize,
	)

	if response.UploadID == "" {
		return nil, fmt.Errorf(
			"prepare response upload_id is empty",
		)
	}

	if _, err := parsePositiveInt64(
		response.BlockSize,
	); err != nil {
		return nil, fmt.Errorf(
			"prepare response block_size is invalid: %w",
			err,
		)
	}

	if len(response.Parts) == 0 {
		return nil, fmt.Errorf(
			"prepare response parts is empty",
		)
	}

	return &response, nil
}

// uploadGroupFileParts 上传所有分片。
//
// 注意：
// 腾讯文档声明 index 从 0 开始，
// 但生产接口已经实际返回过从 1 开始。
//
// 本函数不再假设固定基数：
//   - 将 parts 按 index 排序；
//   - 只要求 index 连续；
//   - 文件 offset 按排序后位置累计；
//   - finish 时原样使用腾讯返回的 index。
func uploadGroupFileParts(
	ctx context.Context,
	apiv2 openapi.OpenAPI,
	groupID string,
	file *os.File,
	fileSize int64,
	prepare *groupUploadPrepareResponse,
) error {
	if prepare == nil {
		return fmt.Errorf(
			"prepare response is nil",
		)
	}

	defaultBlockSize, err := parsePositiveInt64(
		prepare.BlockSize,
	)
	if err != nil {
		return fmt.Errorf(
			"invalid default block size: %w",
			err,
		)
	}

	parts := append(
		[]groupUploadPart(nil),
		prepare.Parts...,
	)

	sort.Slice(
		parts,
		func(i int, j int) bool {
			return parts[i].Index <
				parts[j].Index
		},
	)

	if len(parts) == 0 {
		return fmt.Errorf(
			"parts is empty",
		)
	}

	firstIndex := parts[0].Index
	if firstIndex < 0 {
		return fmt.Errorf(
			"invalid first part index: %d",
			firstIndex,
		)
	}

	// 同时兼容：
	//   0,1,2...
	//   1,2,3...
	for position, part := range parts {
		expectedIndex := firstIndex + position

		if part.Index != expectedIndex {
			return fmt.Errorf(
				"part index is not continuous: expected=%d actual=%d",
				expectedIndex,
				part.Index,
			)
		}
	}

	var offset int64

	for position, part := range parts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if strings.TrimSpace(
			part.PresignedURL,
		) == "" {
			return fmt.Errorf(
				"part %d presigned_url is empty",
				part.Index,
			)
		}

		remaining := fileSize - offset
		if remaining <= 0 {
			return fmt.Errorf(
				"part %d exceeds file size",
				part.Index,
			)
		}

		// 优先使用腾讯针对该片返回的实际 block_size。
		actualSize := int64(0)

		if strings.TrimSpace(
			part.BlockSize,
		) != "" {
			actualSize, err = parsePositiveInt64(
				part.BlockSize,
			)
			if err != nil {
				return fmt.Errorf(
					"part %d block_size is invalid: %w",
					part.Index,
					err,
				)
			}
		} else {
			actualSize = defaultBlockSize
		}

		if actualSize > remaining {
			return fmt.Errorf(
				"part %d size exceeds remaining file: part_size=%d remaining=%d",
				part.Index,
				actualSize,
				remaining,
			)
		}

		// 非最后一片如果小于默认块大小，
		// 只记录日志，不擅自修改腾讯下发的实际值。
		if position < len(parts)-1 &&
			actualSize != defaultBlockSize {
			mylog.Printf(
				"group file non-final part uses custom size: index=%d size=%d default=%d",
				part.Index,
				actualSize,
				defaultBlockSize,
			)
		}

		partMD5, err := calculateGroupFilePartMD5(
			file,
			offset,
			actualSize,
		)
		if err != nil {
			return fmt.Errorf(
				"calculate part %d md5: %w",
				part.Index,
				err,
			)
		}

		if err := putGroupFilePart(
			ctx,
			part.PresignedURL,
			file,
			offset,
			actualSize,
		); err != nil {
			return fmt.Errorf(
				"put part %d: %w",
				part.Index,
				err,
			)
		}

		if err := finishGroupFilePart(
			ctx,
			apiv2,
			groupID,
			prepare.UploadID,
			part.Index,
			actualSize,
			partMD5,
		); err != nil {
			return fmt.Errorf(
				"finish part %d: %w",
				part.Index,
				err,
			)
		}

		mylog.Printf(
			"group file part uploaded: upload_id=%s index=%d position=%d offset=%d size=%d",
			prepare.UploadID,
			part.Index,
			position,
			offset,
			actualSize,
		)

		offset += actualSize
	}

	if offset != fileSize {
		return fmt.Errorf(
			"uploaded size does not match file size: expected=%d actual=%d",
			fileSize,
			offset,
		)
	}

	return nil
}

func parsePositiveInt64(
	value string,
) (int64, error) {
	number, err := strconv.ParseInt(
		strings.TrimSpace(value),
		10,
		64,
	)
	if err != nil {
		return 0, err
	}

	if number <= 0 {
		return 0, fmt.Errorf(
			"value must be positive",
		)
	}

	return number, nil
}

func calculateGroupFilePartMD5(
	file *os.File,
	offset int64,
	size int64,
) (string, error) {
	if file == nil {
		return "", fmt.Errorf(
			"file is nil",
		)
	}

	if offset < 0 || size <= 0 {
		return "", fmt.Errorf(
			"invalid part range",
		)
	}

	hash := md5.New()

	copied, err := io.Copy(
		hash,
		io.NewSectionReader(
			file,
			offset,
			size,
		),
	)
	if err != nil {
		return "", err
	}

	if copied != size {
		return "", fmt.Errorf(
			"unexpected part size: expected=%d actual=%d",
			size,
			copied,
		)
	}

	return hex.EncodeToString(
		hash.Sum(nil),
	), nil
}

// putGroupFilePart 将单个分片 PUT 到腾讯返回的预签名 URL。
//
// 预签名 URL 不使用 QQBot Authorization，
// 因此必须使用普通 http.Client，不能使用 apiv2.Transport。
func putGroupFilePart(
	ctx context.Context,
	presignedURL string,
	file *os.File,
	offset int64,
	size int64,
) error {
	if !validGroupFileURL(
		presignedURL,
	) {
		return fmt.Errorf(
			"presigned URL is invalid",
		)
	}

	body := io.NewSectionReader(
		file,
		offset,
		size,
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		presignedURL,
		body,
	)
	if err != nil {
		return err
	}

	request.ContentLength = size

	response, err := http.DefaultClient.Do(
		request,
	)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// 消费少量响应体，便于连接复用，
	// 同时避免异常服务端返回超大响应。
	_, _ = io.Copy(
		io.Discard,
		io.LimitReader(
			response.Body,
			4096,
		),
	)

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"presigned PUT returned %s",
			response.Status,
		)
	}

	return nil
}

// finishGroupFilePart 通知腾讯该分片已经上传完成。
func finishGroupFilePart(
	ctx context.Context,
	apiv2 openapi.OpenAPI,
	groupID string,
	uploadID string,
	partIndex int,
	blockSize int64,
	partMD5 string,
) error {
	request := &groupUploadPartFinishRequest{
		UploadID:  uploadID,
		PartIndex: partIndex,
		BlockSize: strconv.FormatInt(
			blockSize,
			10,
		),
		MD5: partMD5,
	}

	requestURL := fmt.Sprintf(
		"%s/v2/groups/%s/upload_part_finish",
		qqOpenAPIBaseURL,
		url.PathEscape(groupID),
	)

	_, err := apiv2.Transport(
		ctx,
		http.MethodPost,
		requestURL,
		request,
	)

	return err
}

func validGroupFileURL(
	rawURL string,
) bool {
	rawURL = strings.TrimSpace(
		rawURL,
	)
	if rawURL == "" {
		return false
	}

	parsed, err := url.Parse(
		rawURL,
	)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(
		parsed.Scheme,
	)

	return (scheme == "http" ||
		scheme == "https") &&
		parsed.Host != ""
}
func sendGroupFileURLNotice(
	client callapi.Client,
	message callapi.ActionMessage,
	groupID string,
	fileURL string,
	media *dto.MediaResponse,
	sentMessageID string,
	errorMessage string,
) error {
	if client == nil {
		return fmt.Errorf(
			"group file notice client is nil",
		)
	}

	// 客户端使用 string echo 查找等待 Channel。
	// 在发送前直接确认类型，避免回执发出去却无法匹配。
	echoValue, ok := message.Echo.(string)
	if !ok {
		return fmt.Errorf(
			"group file notice echo is not string: %T",
			message.Echo,
		)
	}

	echoValue = strings.TrimSpace(echoValue)
	if echoValue == "" {
		return fmt.Errorf(
			"group file notice echo is empty",
		)
	}

	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return fmt.Errorf(
			"group file notice group_id is empty",
		)
	}

	status := "ok"
	retCode := 0

	if errorMessage != "" {
		status = "failed"
		retCode = 100
	}

	notice := map[string]interface{}{
		"time":        time.Now().Unix(),
		"self_id":     config.GetAppID(),
		"post_type":   "notice",
		"notice_type": "group_file_url_resp",

		"status":    status,
		"retcode":   retCode,
		"error_msg": errorMessage,
		"echo":      echoValue,

		// 客户端在进入 notice 分支之前，
		// 会使用 real_group_id 执行测试群和例外群过滤。
		"real_group_id": groupID,

		// 保持事件语义完整。
		"real_message_type": "group",

		"url": fileURL,
	}

	if sentMessageID != "" {
		notice["message_id"] = sentMessageID
	}

	if media != nil {
		notice["file_uuid"] = media.FileUUID
		notice["file_info"] = media.FileInfo
		notice["ttl"] = media.TTL
	}

	mylog.Printf(
		"group_file_url_resp enqueue: echo=%s status=%s real_group_id=%s message_id=%s",
		echoValue,
		status,
		groupID,
		sentMessageID,
	)

	if err := client.SendMessage(notice); err != nil {
		mylog.Printf(
			"group_file_url_resp enqueue failed: echo=%s status=%s err=%v",
			echoValue,
			status,
			err,
		)

		return fmt.Errorf(
			"send group_file_url_resp notice: %w",
			err,
		)
	}

	mylog.Printf(
		"group_file_url_resp queued: echo=%s status=%s real_group_id=%s",
		echoValue,
		status,
		groupID,
	)

	return nil
}
