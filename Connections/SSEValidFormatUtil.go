package Connections

import (
	"bytes"
)

var validSSEPrefixesLower = [][]byte{ // 直接存储小写前缀，避免重复转换
	[]byte("data:"),
	[]byte("event:"),
	[]byte("id:"),
	[]byte("retry:"),
}

// isSSEPrefixValid 检查字节切片是否以合法的 SSE 字段开头
func isSSEPrefixValid(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// 优化第一行提取逻辑，减少字节切片操作
	end := bytes.IndexByte(data, '\n') // 先找\n，大部分场景更常见
	if end == -1 {
		end = len(data)
	} else if end > 0 && data[end-1] == '\r' { // 处理\r\n情况
		end--
	}
	firstLine := data[:end]

	// 跳过前导空格（SSE 允许空行或空格）
	trimmed := bytes.TrimLeft(firstLine, " \t")
	if len(trimmed) == 0 {
		return true // 空行是合法的 SSE 行（用于分隔消息）
	}

	// 直接对字节切片进行小写前缀匹配，避免字符串转换
	for _, prefix := range validSSEPrefixesLower {
		if len(trimmed) >= len(prefix) && bytes.EqualFold(trimmed[:len(prefix)], prefix) {
			return true
		}
	}

	return false
}
