package bilibili

import (
	"fmt"
	"github.com/goccy/go-json"
	"io"
	"net/http"
)

// UserData 包含 BiliBili 用户的详细信息
type UserData struct {
	Card      CardInfo `json:"card"`
	Following bool     `json:"following"`
	Follower  int      `json:"follower"`
}

// CardInfo 包含用户的基本信息
type CardInfo struct {
	MID       string    `json:"mid"`
	Name      string    `json:"name"`
	Sex       string    `json:"sex"`
	Rank      string    `json:"rank"`
	Face      string    `json:"face"`
	Sign      string    `json:"sign"`
	Fans      int       `json:"fans"`
	Attention int       `json:"attention"`
	LevelInfo LevelInfo `json:"level_info"`
	Nameplate Nameplate `json:"nameplate"`
	VIP       VIPInfo   `json:"vip"`
}

// LevelInfo 表示用户等级信息
type LevelInfo struct {
	CurrentLevel int `json:"current_level"`
}

// Nameplate 表示用户的勋章信息
type Nameplate struct {
	Name       string `json:"name"`
	Image      string `json:"image"`
	ImageSmall string `json:"image_small"`
}

// VIPInfo 表示用户的 VIP 信息
type VIPInfo struct {
	Status int `json:"status"`
	Label  struct {
		Text string `json:"text"`
	} `json:"label"`
}

// RoomInfo 包含直播间的基本信息
type RoomInfo struct {
	RoomID     int    `json:"room_id"`
	ShortID    int    `json:"short_id"`
	UID        int    `json:"uid"`
	LiveStatus int    `json:"live_status"` // 0: 未开播, 1: 直播中, 2: 轮播中
	Title      string `json:"title"`
}

// FetchRoomInfo 通过房间号获取直播间信息
func FetchRoomInfo(roomID int) (*RoomInfo, error) {
	// 构造 API 请求 URL
	url := fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%d", roomID)

	// 发送 HTTP GET 请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch room data: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 定义临时结构体解析完整响应
	var fullResponse struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    RoomInfo `json:"data"`
	}

	// 解析 JSON 数据
	if err := json.Unmarshal(body, &fullResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// 检查响应码
	if fullResponse.Code != 0 {
		return nil, fmt.Errorf("API error: %s", fullResponse.Message)
	}

	// 返回解析后的房间数据
	return &fullResponse.Data, nil
}

// FetchUserData 通过 MID 从 Bilibili API 获取用户信息，并返回 UserData 对象
func FetchUserData(mid int) (*UserData, error) {
	// 构造 API 请求 URL
	url := fmt.Sprintf("https://api.bilibili.com/x/web-interface/card?mid=%d", mid)

	// 发送 HTTP GET 请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 定义临时结构体解析完整响应
	var fullResponse struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    UserData `json:"data"`
	}

	// 解析 JSON 数据
	if err := json.Unmarshal(body, &fullResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// 检查响应码
	if fullResponse.Code != 0 {
		return nil, fmt.Errorf("API error: %s", fullResponse.Message)
	}

	// 返回解析后的用户数据
	return &fullResponse.Data, nil
}
