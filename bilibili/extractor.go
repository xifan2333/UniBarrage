package bilibili

import (
	"UniBarrage/services/proxy"
	"fmt"
	regexp "github.com/wasilibs/go-re2"
)

// 预编译正则表达式以提高性能，并添加注释解释其用途
var (
	// 匹配 face 字段的 URL，例如："face":"https://example.com/face.png"
	faceURLPattern = regexp.MustCompile(`"face":"(https?://[^"]+)"`)

	// 匹配转义后的表情 URL，例如：\"url\":\"https://example.com/emoticon.png\",\"width\"
	emoticonURLPattern = regexp.MustCompile(`\\?"url\\?":\\?"(https?://[^"]+\.png)\\?",\\?"width\\?"`)
)

// ExtractFaceURL 提取输入字符串中的 face URL
func ExtractFaceURL(input string) (string, error) {
	// 使用预编译的正则表达式查找匹配
	matches := faceURLPattern.FindStringSubmatch(input)

	// 检查是否找到匹配
	if len(matches) > 1 {
		// 生成代理 URL
		proxyURL, _ := proxy.GenerateImageURL(matches[1])
		// 返回提取的 face URL
		return proxyURL, nil
	} else {
		// 如果未找到 URL，则返回错误
		return "", fmt.Errorf("no face URL found")
	}
}

// ExtractEmoticonURLs 从给定的字符串中提取所有的表情链接
func ExtractEmoticonURLs(input string) []string {
	// 使用预编译的正则表达式查找所有符合条件的匹配
	matches := emoticonURLPattern.FindAllStringSubmatch(input, -1)

	// 存储转换后的代理 URL
	var proxyURLs []string

	// 遍历匹配结果并生成代理 URL
	for _, match := range matches {
		if len(match) > 1 {
			proxyURL, _ := proxy.GenerateImageURL(match[1])
			proxyURLs = append(proxyURLs, proxyURL)
		}
	}

	return proxyURLs
}

// ExtractGuardLevel 根据输入的整数返回对应的等级名称
func ExtractGuardLevel(level int) string {
	switch level {
	case 3:
		return "舰长"
	case 2:
		return "提督"
	case 1:
		return "总督"
	default:
		return "未知等级"
	}
}
