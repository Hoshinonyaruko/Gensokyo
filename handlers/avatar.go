package handlers

import (
	"fmt"
	"regexp"

	"github.com/hoshinonyaruko/gensokyo/config"
	"github.com/hoshinonyaruko/gensokyo/idmap"
	"github.com/hoshinonyaruko/gensokyo/mylog"
)

func ProcessCQAvatar(groupID string, text string) string {
	// 断言并获取 groupID 和 qq 号
	qqRegex := regexp.MustCompile(`\[CQ:avatar,qq=(\d+)\]`)
	qqMatches := qqRegex.FindAllStringSubmatch(text, -1)

	for _, match := range qqMatches {
		qqStr := match[1] // 提取 qq 号

		var originalUserID string
		var err error
		if config.GetIdmapPro() {
			// 如果UserID不是nil且配置为使用Pro版本，则调用RetrieveRowByIDv2Pro
			_, originalUserID, err = idmap.RetrieveRowByIDv2Pro(groupID, qqStr)
			if err != nil {
				mylog.Printf("Error1 retrieving original GroupID: %v", err)
				_, originalUserID, err = idmap.RetrieveRowByIDv2Pro("690426430", qqStr)
				if err != nil {
					mylog.Printf("Error reading private originalUserID: %v", err)
					return ""
				}
			}
		} else {
			originalUserID, err = idmap.RetrieveRowByIDv2(qqStr)
			if err != nil {
				mylog.Printf("Error retrieving original UserID: %v", err)
			}
		}

		// 生成头像URL
		avatarURL, _ := GenerateAvatarURLV2(originalUserID)

		// 替换文本中的 [CQ:avatar,qq=12345678] 为 [CQ:image,file=avatarurl]
		replacement := fmt.Sprintf("[CQ:image,file=%s]", avatarURL)
		text = qqRegex.ReplaceAllString(text, replacement)
	}

	return text
}

func ProcessCQAvatarNoGroupID(text string) string {
	// 断言并获取 groupID 和 qq 号
	qqRegex := regexp.MustCompile(`\[CQ:avatar,qq=(\d+)\]`)
	qqMatches := qqRegex.FindAllStringSubmatch(text, -1)

	for _, match := range qqMatches {
		qqStr := match[1] // 提取 qq 号

		var originalUserID string
		var err error
		if config.GetIdmapPro() {
			_, originalUserID, err = idmap.RetrieveRowByIDv2Pro("690426430", qqStr)
			if err != nil {
				mylog.Printf("Error reading private originalUserID: %v", err)
			}
		} else {
			originalUserID, err = idmap.RetrieveRowByIDv2(qqStr)
			if err != nil {
				mylog.Printf("Error retrieving original UserID: %v", err)
			}
		}

		// 生成头像URL
		avatarURL, _ := GenerateAvatarURLV2(originalUserID)

		// 替换文本中的 [CQ:avatar,qq=12345678] 为 [CQ:image,file=avatarurl]
		replacement := fmt.Sprintf("[CQ:image,file=%s]", avatarURL)
		text = qqRegex.ReplaceAllString(text, replacement)
	}

	return text
}

func GetAvatarCQCodeNoGroupID(qqNumber string) (string, error) {
	var originalUserID string
	var err error

	if config.GetIdmapPro() {
		// 如果配置为使用Pro版本，则调用RetrieveRowByIDv2Pro
		_, originalUserID, err = idmap.RetrieveRowByIDv2Pro("690426430", qqNumber)
		if err != nil {
			mylog.Printf("Error reading private originalUserID: %v", err)
			return "", err
		}
	} else {
		// 否则调用RetrieveRowByIDv2
		originalUserID, err = idmap.RetrieveRowByIDv2(qqNumber)
		if err != nil {
			mylog.Printf("Error retrieving original UserID: %v", err)
			return "", err
		}
	}

	// 生成头像URL
	avatarURL, err := GenerateAvatarURLV2(originalUserID)
	if err != nil {
		mylog.Printf("Error generating avatar URL: %v", err)
		return "", err
	}

	// 返回格式化后的字符串
	return fmt.Sprintf("[CQ:image,file=%s]", avatarURL), nil
}

func GetAvatarCQCode(groupID, qqNumber string) (string, error) {
	var originalUserID string
	var err error

	if config.GetIdmapPro() {
		// 如果配置为使用Pro版本，则调用RetrieveRowByIDv2Pro
		_, originalUserID, err = idmap.RetrieveRowByIDv2Pro(groupID, qqNumber)
		if err != nil {
			mylog.Printf("Error retrieving original GroupID: %v", err)
			_, originalUserID, err = idmap.RetrieveRowByIDv2Pro("690426430", qqNumber)
			if err != nil {
				mylog.Printf("Error reading private originalUserID: %v", err)
				return "", err
			}
		}
	} else {
		// 否则调用RetrieveRowByIDv2
		originalUserID, err = idmap.RetrieveRowByIDv2(qqNumber)
		if err != nil {
			mylog.Printf("Error retrieving original UserID: %v", err)
			return "", err
		}
	}

	// 生成头像URL
	avatarURL, err := GenerateAvatarURLV2(originalUserID)
	if err != nil {
		mylog.Printf("Error generating avatar URL: %v", err)
		return "", err
	}

	// 返回格式化后的字符串
	return fmt.Sprintf("[CQ:image,file=%s]", avatarURL), nil
}
