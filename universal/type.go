package universal

import (
	"errors"
	"fmt"
	"github.com/goccy/go-json"
)

// Platform 定义平台类型
type Platform string

const (
	DouYin   Platform = "douyin"   // 抖音
	BiliBili Platform = "bilibili" // 哔哩哔哩
	KuaiShou Platform = "kuaishou" // 快手
	HuYa     Platform = "huya"     // 虎牙
	DouYu    Platform = "douyu"    // 斗鱼
)

// MessageType 定义消息类型
type MessageType string

const (
	ChatMessageType      MessageType = "Chat"      // 聊天消息
	GiftMessageType      MessageType = "Gift"      // 礼物消息
	SubscribeMessageType MessageType = "Subscribe" // 订阅消息
	SuperChatMessageType MessageType = "SuperChat" // 超级聊天消息
	LikeMessageType      MessageType = "Like"      // 点赞消息
	EnterRoomMessageType MessageType = "EnterRoom" // 进入房间消息
	EndLiveMessageType   MessageType = "EndLive"   // 结束直播消息
)

// UniMessage 结构体，表示统一的消息结构
type UniMessage struct {
	RID      string      `json:"rid"`      // 房间 ID
	Platform Platform    `json:"platform"` // 平台类型
	Type     MessageType `json:"type"`     // 消息类型
	Data     MessageData `json:"data"`     // 消息数据
}

// MessageData 接口，所有消息类型必须实现此接口
type MessageData interface {
	IsMessageData()
}

// 各种消息类型的定义和实现

// ChatMessage 表示聊天消息
type ChatMessage struct {
	Name     string      `json:"name"`     // 发送者的名称
	Avatar   string      `json:"avatar"`   // 发送者的头像
	Content  string      `json:"content"`  // 消息内容
	Emoticon []string    `json:"emoticon"` // 表情包列表
	Raw      interface{} `json:"raw"`      // 原始数据
}

func (*ChatMessage) IsMessageData() {}

func (m *ChatMessage) MarshalJSON() ([]byte, error) {
	type Alias ChatMessage
	raw, err := handleRawField(m.Raw)
	if err != nil {
		return nil, err
	}
	m.Raw = raw
	return json.Marshal((*Alias)(m))
}

// GiftMessage 表示礼物消息
type GiftMessage struct {
	Name     string      `json:"name"`     // 送礼者的名称
	Avatar   string      `json:"avatar"`   // 送礼者的头像
	Item     string      `json:"item"`     // 礼物名称
	Num      int         `json:"num"`      // 礼物数量
	Price    float64     `json:"price"`    // 礼物价格
	GiftIcon string      `json:"giftIcon"` // 礼物图标
	Raw      interface{} `json:"raw"`      // 原始数据
}

func (*GiftMessage) IsMessageData() {}

func (m *GiftMessage) MarshalJSON() ([]byte, error) {
	type Alias GiftMessage
	raw, err := handleRawField(m.Raw)
	if err != nil {
		return nil, err
	}
	m.Raw = raw
	return json.Marshal((*Alias)(m))
}

// SubscribeMessage 表示订阅消息
type SubscribeMessage struct {
	Name   string      `json:"name"`   // 订阅者的名称
	Avatar string      `json:"avatar"` // 订阅者的头像
	Item   string      `json:"item"`   // 订阅的项目
	Num    int         `json:"num"`    // 订阅次数
	Price  float64     `json:"price"`  // 订阅费用
	Raw    interface{} `json:"raw"`    // 原始数据
}

func (*SubscribeMessage) IsMessageData() {}

func (m *SubscribeMessage) MarshalJSON() ([]byte, error) {
	type Alias SubscribeMessage
	raw, err := handleRawField(m.Raw)
	if err != nil {
		return nil, err
	}
	m.Raw = raw
	return json.Marshal((*Alias)(m))
}

// SuperChatMessage 表示超级聊天消息
type SuperChatMessage struct {
	Name    string      `json:"name"`    // 发送者的名称
	Avatar  string      `json:"avatar"`  // 发送者的头像
	Content string      `json:"content"` // 消息内容
	Price   float64     `json:"price"`   // 超级聊天金额
	Raw     interface{} `json:"raw"`     // 原始数据
}

func (*SuperChatMessage) IsMessageData() {}

func (m *SuperChatMessage) MarshalJSON() ([]byte, error) {
	type Alias SuperChatMessage
	raw, err := handleRawField(m.Raw)
	if err != nil {
		return nil, err
	}
	m.Raw = raw
	return json.Marshal((*Alias)(m))
}

// LikeMessage 表示点赞消息
type LikeMessage struct {
	Name   string      `json:"name"`   // 点赞者的名称
	Avatar string      `json:"avatar"` // 点赞者的头像
	Count  int         `json:"count"`  // 点赞次数
	Raw    interface{} `json:"raw"`    // 原始数据
}

func (*LikeMessage) IsMessageData() {}

func (m *LikeMessage) MarshalJSON() ([]byte, error) {
	type Alias LikeMessage
	raw, err := handleRawField(m.Raw)
	if err != nil {
		return nil, err
	}
	m.Raw = raw
	return json.Marshal((*Alias)(m))
}

// EnterRoomMessage 表示进入房间消息
type EnterRoomMessage struct {
	Name   string      `json:"name"`   // 用户名称
	Avatar string      `json:"avatar"` // 用户头像
	Raw    interface{} `json:"raw"`    // 原始数据
}

func (*EnterRoomMessage) IsMessageData() {}

func (m *EnterRoomMessage) MarshalJSON() ([]byte, error) {
	type Alias EnterRoomMessage
	raw, err := handleRawField(m.Raw)
	if err != nil {
		return nil, err
	}
	m.Raw = raw
	return json.Marshal((*Alias)(m))
}

// EndLiveMessage 表示结束直播消息
type EndLiveMessage struct {
	Raw interface{} `json:"raw"` // 原始数据
}

func (*EndLiveMessage) IsMessageData() {}

func (m *EndLiveMessage) MarshalJSON() ([]byte, error) {
	type Alias EndLiveMessage
	raw, err := handleRawField(m.Raw)
	if err != nil {
		return nil, err
	}
	m.Raw = raw
	return json.Marshal((*Alias)(m))
}

// 通用的处理 Raw 字段的函数，用于处理消息中的 Raw 字段，确保其以 JSON 格式保存
func handleRawField(raw interface{}) (json.RawMessage, error) {
	switch rawField := raw.(type) {
	case string:
		return json.RawMessage(rawField), nil
	default:
		rawJSON, err := json.Marshal(rawField)
		if err != nil {
			return nil, err
		}
		return rawJSON, nil
	}
}

// IsValidPlatform 验证 Platform 是否有效
func IsValidPlatform(platform Platform) bool {
	switch platform {
	case DouYin, BiliBili, KuaiShou, HuYa, DouYu:
		return true
	default:
		return false
	}
}

// CreateUniMessage 创建 UniMessage 的工厂函数
func CreateUniMessage(rid string, platform Platform, msgType MessageType, data MessageData) (*UniMessage, error) {
	if !IsValidPlatform(platform) {
		return nil, errors.New("无效的平台，必须是 'DouYin', 'BiliBili', 'KuaiShou', 'HuYa', 'DouYu' 中之一")
	}

	switch msgType {
	case ChatMessageType, GiftMessageType, SubscribeMessageType, SuperChatMessageType, LikeMessageType, EnterRoomMessageType, EndLiveMessageType:
		return &UniMessage{
			RID:      rid,
			Platform: platform,
			Type:     msgType,
			Data:     data,
		}, nil
	default:
		return nil, fmt.Errorf("无效的消息类型: %s", msgType)
	}
}
