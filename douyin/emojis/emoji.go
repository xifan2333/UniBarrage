package emojis

import (
	log "UniBarrage/utils/trace"
	"github.com/goccy/go-json"
	regexp "github.com/wasilibs/go-re2"
	"io"
	"net/http"
	"sync"
)

// Emoji 定义结构体以匹配 JSON 数据结构
type Emoji struct {
	OriginUri   string `json:"origin_uri"`
	DisplayName string `json:"display_name"`
	Hide        int    `json:"hide"`
	EmojiUrl    struct {
		Uri     string   `json:"uri"`
		UrlList []string `json:"url_list"`
	} `json:"emoji_url"`
}

type Response struct {
	StatusCode int     `json:"status_code"`
	Version    int64   `json:"version"`
	EmojiList  []Emoji `json:"emoji_list"`
}

// 使用普通 map 和 sync.RWMutex 来实现线程安全
var (
	emojiMap = make(map[string]string)
	rwLock   sync.RWMutex
)

// init 函数在包导入时自动执行
func init() {
	go func() {
		if err := fetchEmojiList(); err != nil {
			log.Printf("WARN", "获取抖音表情列表失败: %v", err)
		}
	}()
}

// 从 API 获取 emoji 数据并保存到内存
func fetchEmojiList() error {
	resp, err := http.Get("https://www.douyin.com/aweme/v1/web/emoji/list")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	// 写入时加写锁
	rwLock.Lock()
	defer rwLock.Unlock()
	for _, emoji := range response.EmojiList {
		if len(emoji.EmojiUrl.UrlList) > 0 {
			emojiMap[emoji.DisplayName] = emoji.EmojiUrl.UrlList[0]
		}
	}

	return nil
}

// ParseEmojiURL 匹配字符串中的 emoji 标签并转换为 URL
func ParseEmojiURL(input string) []string {
	var urls []string
	regex := regexp.MustCompile(`\[[^\[\]]+\]`) // 匹配 [标签] 格式

	matches := regex.FindAllString(input, -1)
	rwLock.RLock() // 读操作加读锁
	defer rwLock.RUnlock()
	for _, match := range matches {
		if url, exists := emojiMap[match]; exists {
			urls = append(urls, url)
		}
	}

	return urls
}
