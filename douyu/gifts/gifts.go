package gifts

import (
	log "UniBarrage/utils/trace"
	"fmt"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type FlashConfig struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Hdt  string `json:"hdt"`
}

type ConfigData struct {
	FlashConfig map[string]FlashConfig `json:"flashConfig"`
}

type DYConfig struct {
	Error    int        `json:"error"`
	Callback string     `json:"callback"`
	Data     ConfigData `json:"data"`
}

type Gift struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"himg"`
}

// 全局 giftMap 和 RWMutex 用于并发访问
var (
	giftMap = make(map[int]struct {
		Name     string
		ImageURL string
	})
	rwLock sync.RWMutex
)

// init 函数在包导入时自动执行，用于初始化礼物列表
func init() {
	go func() {
		if err := InitGiftList(); err != nil {
			log.Printf("WARN", "获取斗鱼礼物列表失败: %v", err)
		}
	}()
}

// InitGiftList 初始化并填充 giftMap 数据
func InitGiftList() error {
	urls := []struct {
		url        string
		isCallback bool
	}{
		{"https://webconf.douyucdn.cn/resource/common/gift/flash/gift_effect.json", true},
		{"https://fastly.jsdelivr.net/gh/popzoo/pop/json/dynamic_gift.json", false},
	}

	var allGifts []Gift
	for _, source := range urls {
		// 获取并解析每个 URL 的数据
		gifts, err := fetchAndParseURL(source.url, source.isCallback)
		if err != nil {
			return fmt.Errorf("从 %s 获取数据失败: %v", source.url, err)
		}
		allGifts = append(allGifts, gifts...)
	}

	// 将合并的列表按 ID 升序排序
	sort.Slice(allGifts, func(i, j int) bool {
		return allGifts[i].ID < allGifts[j].ID
	})

	// 使用写锁填充全局 giftMap
	rwLock.Lock()
	defer rwLock.Unlock()
	for _, gift := range allGifts {
		giftMap[gift.ID] = struct {
			Name     string
			ImageURL string
		}{
			Name:     gift.Name,
			ImageURL: gift.ImageURL,
		}
	}
	return nil
}

// GetGiftByID 并发安全地根据 ID 获取礼物信息
func GetGiftByID(id string) (struct {
	Name     string
	ImageURL string
}, bool) {
	_id, _ := strconv.Atoi(id)
	rwLock.RLock() // 读锁，用于并发安全读取
	defer rwLock.RUnlock()
	if value, exists := giftMap[_id]; exists {
		return value, true
	}
	return struct {
		Name     string
		ImageURL string
	}{}, false
}

// fetchAndParseURL 从指定 URL 获取数据并解析为礼物列表
func fetchAndParseURL(url string, isCallback bool) ([]Gift, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取数据出错: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体出错: %v", err)
	}

	if isCallback {
		// 移除回调包装
		trimmedBody := strings.TrimPrefix(string(body), "DYConfigCallback(")
		trimmedBody = strings.TrimSuffix(trimmedBody, ");")
		trimmedBody = strings.TrimSpace(trimmedBody)
		body = []byte(trimmedBody)
	}

	if isCallback {
		// 解析第一种结构
		var config DYConfig
		if err := json.Unmarshal(body, &config); err != nil {
			return nil, fmt.Errorf("JSON 解析出错: %v", err)
		}

		var gifts []Gift
		for _, value := range config.Data.FlashConfig {
			gifts = append(gifts, Gift{
				ID:       value.ID,
				Name:     value.Name,
				ImageURL: value.Hdt,
			})
		}
		return gifts, nil
	} else {
		// 解析第二种结构
		var data []Gift
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("JSON 解析出错: %v", err)
		}
		return data, nil
	}
}
