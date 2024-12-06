package douyu

import "fmt"

// BuildAvatarURL 拼接 pathString 为头像 URL
func BuildAvatarURL(pathString string) string {
	baseURL := "https://apic.douyucdn.cn/upload/"
	return fmt.Sprintf("%s%s_big.jpg", baseURL, pathString)
}
