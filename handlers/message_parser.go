package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hoshinonyaruko/gensokyo/callapi"
)

var BotID string
var AppID string

func parseMessageContent(paramsMessage callapi.ParamsContent) (string, map[string][]string) {
	messageText := ""

	switch message := paramsMessage.Message.(type) {
	case string:
		fmt.Printf("params.message is a string\n")
		messageText = message
	case []interface{}:
		//多个映射组成的切片
		fmt.Printf("params.message is a slice (segment_type_koishi)\n")
		for _, segment := range message {
			segmentMap, ok := segment.(map[string]interface{})
			if !ok {
				continue
			}

			segmentType, ok := segmentMap["type"].(string)
			if !ok {
				continue
			}

			segmentContent := ""
			switch segmentType {
			case "text":
				segmentContent, _ = segmentMap["data"].(map[string]interface{})["text"].(string)
			case "image":
				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
				segmentContent = "[CQ:image,file=" + fileContent + "]"
			case "voice":
				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
				segmentContent = "[CQ:record,file=" + fileContent + "]"
			case "at":
				qqNumber, _ := segmentMap["data"].(map[string]interface{})["qq"].(string)
				segmentContent = "[CQ:at,qq=" + qqNumber + "]"
			}

			messageText += segmentContent
		}
	case map[string]interface{}:
		//单个映射
		fmt.Printf("params.message is a map (segment_type_trss)\n")
		messageType, _ := message["type"].(string)
		switch messageType {
		case "text":
			messageText, _ = message["data"].(map[string]interface{})["text"].(string)
		case "image":
			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
			messageText = "[CQ:image,file=" + fileContent + "]"
		case "voice":
			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
			messageText = "[CQ:record,file=" + fileContent + "]"
		case "at":
			qqNumber, _ := message["data"].(map[string]interface{})["qq"].(string)
			messageText = "[CQ:at,qq=" + qqNumber + "]"
		}
	default:
		log.Println("Unsupported message format: params.message field is not a string, map or slice")
	}

	// 正则表达式部分
	localImagePattern := regexp.MustCompile(`\[CQ:image,file=file:///([^\]]+?)\]`)
	urlImagePattern := regexp.MustCompile(`\[CQ:image,file=http://(.+)\]`)
	base64ImagePattern := regexp.MustCompile(`\[CQ:image,file=base64://(.+)\]`)
	base64RecordPattern := regexp.MustCompile(`\[CQ:record,file=base64://(.+)\]`)

	patterns := []struct {
		key     string
		pattern *regexp.Regexp
	}{
		{"local_image", localImagePattern},
		{"url_image", urlImagePattern},
		{"base64_image", base64ImagePattern},
		{"base64_record", base64RecordPattern},
	}

	foundItems := make(map[string][]string)
	for _, pattern := range patterns {
		matches := pattern.pattern.FindAllStringSubmatch(messageText, -1)
		for _, match := range matches {
			if len(match) > 1 {
				foundItems[pattern.key] = append(foundItems[pattern.key], match[1])
				messageText = pattern.pattern.ReplaceAllString(messageText, "")
			}
		}
	}
	//处理at
	messageText = transformMessageText(messageText)

	return messageText, foundItems
}

func transformMessageText(messageText string) string {
	// 使用正则表达式来查找所有[CQ:at,qq=数字]的模式
	re := regexp.MustCompile(`\[CQ:at,qq=(\d+)\]`)
	// 使用正则表达式来替换找到的模式为<@!数字>
	return re.ReplaceAllStringFunc(messageText, func(m string) string {
		submatches := re.FindStringSubmatch(m)
		if len(submatches) > 1 {
			return "<@!" + submatches[1] + ">"
		}
		return m
	})
}

func RevertTransformedText(messageText string) string {
	// Trim leading and trailing spaces
	messageText = strings.TrimSpace(messageText)

	// 将messageText里的BotID替换成AppID
	messageText = strings.ReplaceAll(messageText, BotID, AppID)

	// 使用正则表达式来查找所有<@!数字>的模式
	re := regexp.MustCompile(`<@!(\d+)>`)
	// 使用正则表达式来替换找到的模式为[CQ:at,qq=数字]
	return re.ReplaceAllStringFunc(messageText, func(m string) string {
		submatches := re.FindStringSubmatch(m)
		if len(submatches) > 1 {
			return "[CQ:at,qq=" + submatches[1] + "]"
		}
		return m
	})
}
