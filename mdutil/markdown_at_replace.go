package mdutil

import (
	"strings"

	"github.com/hoshinonyaruko/gensokyo/unioncache"
)

// ReplaceQQBotAtUserIDUnionToRaw scans markdown and replaces
// <qqbot-at-user id="UNION" />  ->  <qqbot-at-user id="RAW" />
// by looking up RAW via unioncache.ID(UNION).
//
// - No regex
// - Single pass O(n)
// - Only touches qqbot-at-user id="..."
// - If lookup misses, leaves it unchanged
func ReplaceQQBotAtUserIDUnionToRaw(markdown string) string {
	const prefix = `<qqbot-at-user id="`
	if markdown == "" {
		return markdown
	}

	// 快速路径：不存在前缀就直接返回，避免任何分配
	first := strings.Index(markdown, prefix)
	if first < 0 {
		return markdown
	}

	var b strings.Builder
	// 预分配：通常替换后长度差不多
	b.Grow(len(markdown))

	// i 是当前扫描位置
	i := 0
	for {
		// 找下一个标签前缀
		j := strings.Index(markdown[i:], prefix)
		if j < 0 {
			// 追加剩余内容
			b.WriteString(markdown[i:])
			break
		}
		j += i

		// 写入前缀之前的内容
		b.WriteString(markdown[i:j])

		// 写入前缀本身
		b.WriteString(prefix)

		// id 的起始位置（引号之后）
		idStart := j + len(prefix)

		// 找结束引号
		k := strings.IndexByte(markdown[idStart:], '"')
		if k < 0 {
			// 不完整标签：把剩余原样写回，结束
			b.WriteString(markdown[idStart:])
			break
		}
		idEnd := idStart + k

		unionID := markdown[idStart:idEnd]

		// 反查原始 id（miss 则保持 unionID）
		if rawID, ok := unioncache.ID(unionID); ok && rawID != "" {
			b.WriteString(rawID)
		} else {
			b.WriteString(unionID)
		}

		// 写入结束引号
		b.WriteByte('"')

		// 从结束引号后继续扫描（注意：我们已经写入了这个引号）
		i = idEnd + 1
	}

	return b.String()
}
