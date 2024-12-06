package gifts

import (
	"UniBarrage/services/proxy"
	"errors"
	"fmt"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"sync"
)

type Gift struct {
	GiftName       string `json:"name"`
	ImgBasic       string `json:"img_basic"`
	ImgDynamic     string `json:"img_dynamic"`
	FrameAnimation string `json:"frame_animation"`
	Gif            string `json:"gif"`
	Webp           string `json:"webp"`
}

type GiftDetails struct {
	ImgBasic       string
	ImgDynamic     string
	FrameAnimation string
	Gif            string
	Webp           string
}

var (
	giftMap = make(map[string]GiftDetails) // 使用普通 map
	rwLock  sync.RWMutex                   // 使用 sync.RWMutex 来实现读写锁
)

func InitGiftMap(roomID int) error {
	url := fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/giftPanel/giftConfig?platform=pc&room_id=%d", roomID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP 请求出错: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("收到非 200 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体出错: %w", err)
	}

	var apiResponse struct {
		Data struct {
			List []Gift `json:"list"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return fmt.Errorf("解析 JSON 出错: %w", err)
	}

	rwLock.Lock() // 加锁进行写操作
	defer rwLock.Unlock()
	for _, gift := range apiResponse.Data.List {
		// 检查礼物是否已经存在于 map 中
		if _, exists := giftMap[gift.GiftName]; !exists {
			imgBasic, _ := proxy.GenerateImageURL(gift.ImgBasic)
			imgDynamic, _ := proxy.GenerateImageURL(gift.ImgDynamic)
			frameAnimation, _ := proxy.GenerateImageURL(gift.FrameAnimation)
			gif, _ := proxy.GenerateImageURL(gift.Gif)
			webp, _ := proxy.GenerateImageURL(gift.Webp)

			giftMap[gift.GiftName] = GiftDetails{
				ImgBasic:       imgBasic,
				ImgDynamic:     imgDynamic,
				FrameAnimation: frameAnimation,
				Gif:            gif,
				Webp:           webp,
			}
		}
	}

	return nil
}

func GetGiftDetailsByName(giftName string) (GiftDetails, error) {
	rwLock.RLock() // 加锁进行读操作
	defer rwLock.RUnlock()
	if details, found := giftMap[giftName]; found {
		return details, nil
	}
	return GiftDetails{}, errors.New("找不到匹配的礼物")
}
