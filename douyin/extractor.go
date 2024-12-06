package douyin

import (
	"fmt"
	"github.com/goccy/go-json"
	regexp "github.com/wasilibs/go-re2"
	"strconv"
	"strings"
)

// SubscriptionInfo 订阅信息结构体
type SubscriptionInfo struct {
	NickName   string
	AvatarURL  string
	DisplayID  string
	PeriodType string
}

// 预编译正则表达式以提高性能，并添加注释解释每个正则表达式的用途
var (
	// 匹配 nickName 字段内容，如：nickName:"某用户名"
	nickNamePattern = regexp.MustCompile(`nickName:"([^"]+)"`)

	// 匹配 AvatarThumb 的 URL，如：AvatarThumb:{url_list:"https://example.com/avatar.png"}
	avatarPattern = regexp.MustCompile(`AvatarThumb:\{url_list:"([^"]+)"`)

	// 匹配 displayId 字段内容，如：displayId:"123456"
	displayIDPattern = regexp.MustCompile(`displayId:"([^"]+)"`)

	// 匹配 string_value 字段中的订阅类型，如：string_value:"月度"
	periodPattern = regexp.MustCompile(`string_value:"(月度|季度|年度)"`)

	// 匹配以 .png 结尾的 URL，用于提取 emoji 图片的链接
	emojiURLPattern = regexp.MustCompile(`https:\/\/[a-zA-Z0-9\/\.\-]+\.png`)

	// 匹配 “空格 + 数字 + 个” 中的礼物数量，如：" 12个"
	giftCountPattern = regexp.MustCompile(`\s(\d+)\s*个`)

	// 匹配 JSON 中的 msg_id 字段，将其转换为字符串格式
	msgIDRegex = regexp.MustCompile(`"msg_id":\s*([0-9]+)`)

	// 匹配 JSON 中的 room_id 字段，将其转换为字符串格式
	roomIDRegex = regexp.MustCompile(`"room_id":\s*([0-9]+)`)
)

// ExtractSubscriptionInfo 通过正则解析订阅信息
func ExtractSubscriptionInfo(data string) (*SubscriptionInfo, error) {
	// 检查是否存在"开通"和"会员"
	if !strings.Contains(data, "开通") || !strings.Contains(data, "会员") {
		return nil, fmt.Errorf("未找到订阅关键字")
	}

	// 提取信息
	nickNameMatch := nickNamePattern.FindStringSubmatch(data)
	avatarMatch := avatarPattern.FindStringSubmatch(data)
	displayIDMatch := displayIDPattern.FindStringSubmatch(data)
	periodMatch := periodPattern.FindStringSubmatch(data)

	// 检查是否成功匹配到所有必需信息
	if len(nickNameMatch) < 2 || len(avatarMatch) < 2 || len(displayIDMatch) < 2 || len(periodMatch) < 2 {
		return nil, fmt.Errorf("解析订阅信息失败")
	}

	// 创建并返回订阅信息结构体
	info := &SubscriptionInfo{
		NickName:   nickNameMatch[1],
		AvatarURL:  avatarMatch[1],
		DisplayID:  displayIDMatch[1],
		PeriodType: periodMatch[1],
	}
	return info, nil
}

// ExtractEmojiImageURLs 从输入的文本中提取 emoji 的图片 URL
func ExtractEmojiImageURLs(input string) []string {
	// 查找所有符合条件的 URL
	matches := emojiURLPattern.FindAllString(input, -1)

	// 返回匹配的结果
	return matches
}

// ExtractGiftCount 从输入文本中提取礼物数量
func ExtractGiftCount(input string) (int, error) {
	// 查找第一个匹配项
	match := giftCountPattern.FindStringSubmatch(input)

	// 如果找到匹配项，提取并转换为整数
	if len(match) > 1 {
		count, err := strconv.Atoi(strings.TrimSpace(match[1]))
		if err != nil {
			return 0, fmt.Errorf("failed to convert gift count to integer: %v", err)
		}
		return count, nil
	}

	// 如果未找到匹配项，返回错误
	return 0, fmt.Errorf("no gift count found")
}

// SafeJSON 将输入结构体的 m.Common.RoomId 和 m.Common.MsgId 字段转换为字符串格式
func SafeJSON(v interface{}) string {
	// 将输入结构体序列化为 JSON
	data, err := json.Marshal(v)
	if err != nil {
		// 返回空字符串或适当处理错误
		return ""
	}
	jsonStr := string(data)

	// 将 msg_id 和 room_id 转换为字符串格式
	jsonStr = msgIDRegex.ReplaceAllString(jsonStr, `"msg_id": "$1"`)
	jsonStr = roomIDRegex.ReplaceAllString(jsonStr, `"room_id": "$1"`)

	return jsonStr
}
