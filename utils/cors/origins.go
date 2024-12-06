package cors

import "strings"

// ParseOrigins 解析 allowedOrigins 参数并返回字符串切片
func ParseOrigins(allowedOrigins string) []string {
	if allowedOrigins == "*" {
		return []string{"*"} // 默认允许所有来源
	}

	parts := strings.Split(allowedOrigins, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
