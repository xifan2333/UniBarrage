package douyin

import (
	"UniBarrage/douyin/emojis"
	"UniBarrage/douyin/generated/douyin"
	"UniBarrage/douyin/utils"
	ws "UniBarrage/services/websockets"
	uni "UniBarrage/universal"
	log "UniBarrage/utils/trace"
	"fmt"
	"google.golang.org/protobuf/proto"
	"strconv"
)

// StartListen 启动抖音直播监听
func StartListen(room int, stopChan chan struct{}) {
	d, err := NewDouyinLive(strconv.Itoa(room))
	if err != nil {
		log.Printf("ERROR", "抖音直播监听启动失败: %v", err)
		close(stopChan)
		return
	}

	//// 创建一个 done channel 用于通知清理
	//done := make(chan struct{})
	//defer func() {
	//	close(done) // 确保通道在退出时关闭
	//}()

	// 监听 stopChan
	go func() {
		<-stopChan
		d.Stop()
	}()

	log.Print("DOUYIN", "已启动抖音直播监听")
	d.Subscribe(func(eventData *douyin.Message) { SubscribeDouYin(eventData, room) })
	err = d.Start()
	if err != nil {
		close(stopChan)
	}
}

// SubscribeDouYin 处理抖音的事件
func SubscribeDouYin(eventData *douyin.Message, room int) {
	id := strconv.Itoa(room)

	// 处理聊天消息事件
	handleChatMessage := func(msg interface{}) {
		m := msg.(*douyin.ChatMessage)
		data, _ := uni.CreateUniMessage(
			id,
			uni.DouYin,
			uni.ChatMessageType,
			&uni.ChatMessage{
				Name:     m.User.NickName,
				Avatar:   m.User.AvatarThumb.UrlList[0],
				Content:  m.Content,
				Emoticon: emojis.ParseEmojiURL(m.Content),
				Raw:      SafeJSON(m),
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理表情消息事件
	handleEmojiChatMessage := func(msg interface{}) {
		m := msg.(*douyin.EmojiChatMessage)
		emoticon := ExtractEmojiImageURLs(m.EmojiContent.String())
		data, _ := uni.CreateUniMessage(
			id,
			uni.DouYin,
			uni.ChatMessageType,
			&uni.ChatMessage{
				Name:     m.User.NickName,
				Avatar:   m.User.AvatarThumb.UrlList[0],
				Content:  m.DefaultContent,
				Emoticon: emoticon,
				Raw:      SafeJSON(m),
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理礼物消息事件
	handleGiftMessage := func(msg interface{}) {
		m := msg.(*douyin.GiftMessage)

		// 检查是否符合 combo 和 repeat_end 条件
		if (m.Gift.Combo && m.RepeatEnd == 1) || !m.Gift.Combo {
			// 过滤符合条件的礼物消息
			num, _ := ExtractGiftCount(m.String())
			data, _ := uni.CreateUniMessage(
				id,
				uni.DouYin,
				uni.GiftMessageType,
				&uni.GiftMessage{
					Name:     m.User.NickName,
					Avatar:   m.User.AvatarThumb.UrlList[0],
					Item:     m.Gift.Name,
					Num:      num,
					Price:    float64(m.Gift.DiamondCount) * 0.1 * float64(num),
					GiftIcon: m.Gift.Image.UrlList[0],
					Raw:      SafeJSON(m),
				},
			)
			ws.BroadcastToClients(data)
		}
		// 不符合 combo 和 repeat_end 条件的消息将被跳过
	}

	// 处理会员订阅消息事件
	handleRoomMessage := func(msg interface{}) {
		m := msg.(*douyin.RoomMessage)
		info, err := ExtractSubscriptionInfo(m.String())
		if err != nil {
			return
		}
		data, _ := uni.CreateUniMessage(
			id,
			uni.DouYin,
			uni.SubscribeMessageType,
			&uni.SubscribeMessage{
				Name:   info.NickName,
				Avatar: info.AvatarURL,
				Item:   info.PeriodType,
				Num:    1,
				Price:  0,
				Raw:    SafeJSON(m),
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理点赞消息事件
	handleLikeMessage := func(msg interface{}) {
		l := msg.(*douyin.LikeMessage)
		data, _ := uni.CreateUniMessage(
			id,
			uni.DouYin,
			uni.LikeMessageType,
			&uni.LikeMessage{
				Name:   l.User.NickName,
				Avatar: l.User.AvatarThumb.UrlList[0],
				Count:  int(l.Count),
				Raw:    SafeJSON(l),
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理进入房间消息事件
	handleMemberMessage := func(msg interface{}) {
		e := msg.(*douyin.MemberMessage)
		data, _ := uni.CreateUniMessage(
			id,
			uni.DouYin,
			uni.EnterRoomMessageType,
			&uni.EnterRoomMessage{
				Name:   e.User.NickName,
				Avatar: e.User.AvatarThumb.UrlList[0],
				Raw:    SafeJSON(e),
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理结束直播消息事件
	handleControlMessage := func(msg interface{}) {
		e := msg.(*douyin.ControlMessage)
		if e.Status == 3 {
			data, _ := uni.CreateUniMessage(
				id,
				uni.DouYin,
				uni.EndLiveMessageType,
				&uni.EndLiveMessage{
					Raw: SafeJSON(e),
				},
			)
			ws.BroadcastToClients(data)
		}
	}

	// 匹配消息方法
	msg, err := utils.MatchMethod(eventData.Method)
	if err != nil {
		//log.Printf("DOUYIN", "未实现的事件: %s", eventData.Method)
		return
	}

	// 反序列化 Payload
	if err := proto.Unmarshal(eventData.Payload, msg); err != nil {
		log.Printf("ERROR", "反序列化失败: %v", err)
		return
	}

	// 消息处理函数映射
	messageHandlers := map[string]func(interface{}){
		"*douyin.ChatMessage":      handleChatMessage,
		"*douyin.EmojiChatMessage": handleEmojiChatMessage,
		"*douyin.GiftMessage":      handleGiftMessage,
		"*douyin.RoomMessage":      handleRoomMessage,
		"*douyin.LikeMessage":      handleLikeMessage,
		"*douyin.MemberMessage":    handleMemberMessage,
		"*douyin.ControlMessage":   handleControlMessage,
	}

	// 根据消息类型调用相应处理函数
	if handler, ok := messageHandlers[fmt.Sprintf("%T", msg)]; ok {
		handler(msg)
	}
}
