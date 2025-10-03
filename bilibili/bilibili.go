package bilibili

import (
	"UniBarrage/bilibili/gifts"
	"UniBarrage/services/proxy"
	ws "UniBarrage/services/websockets"
	uni "UniBarrage/universal"
	log "UniBarrage/utils/trace"
	"context"
	"strconv"

	"github.com/Akegarasu/blivedm-go/client"
	"github.com/Akegarasu/blivedm-go/message"
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

// StartListen 启动哔哩哔哩直播监听
func StartListen(room int, cookie string, stopChan chan struct{}) {
	id := strconv.Itoa(room)

	// 先验证房间是否存在和获取房间信息
	roomInfo, err := FetchRoomInfo(room)
	if err != nil {
		log.Printf("ERROR", "获取 B 站房间信息失败: %v", err)
		close(stopChan)
		return
	}

	// 检查房间是否存在
	if roomInfo.RoomID == 0 {
		log.Print("ERROR", "B 站房间不存在或已关闭")
		close(stopChan)
		return
	}

	// 使用真实房间ID创建客户端
	c := client.NewClient(roomInfo.RoomID)

	// 设置 Cookie（如果有）
	if cookie != "" {
		c.SetCookie(cookie)
	}

	// 创建用于控制子协程的context
	ctx, cancel := context.WithCancel(context.Background())

	// 修改这部分代码以确保正确停止
	go func() {
		<-stopChan
		cancel() // 取消context
		if c != nil {
			c.Stop() // 立即停止WebSocket客户端
			log.Print("INFO", "已停止哔哩哔哩直播监听")
		}
	}()

	// 处理弹幕事件
	handleDanmaku := func(event interface{}) {
		d := event.(*message.Danmaku)
		avatar, _ := ExtractFaceURL(d.Raw)

		data, _ := uni.CreateUniMessage(
			id,
			uni.BiliBili,
			uni.ChatMessageType,
			&uni.ChatMessage{
				Name:     d.Sender.Uname,
				Avatar:   avatar,
				Content:  d.Content,
				Emoticon: ExtractEmoticonURLs(d.Raw),
				Raw:      d,
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理礼物事件
	handleGift := func(event interface{}) {
		g := event.(*message.Gift)
		icon, _ := gifts.GetGiftDetailsByName(g.GiftName)
		avatar, _ := proxy.GenerateImageURL(g.Face)

		data, _ := uni.CreateUniMessage(
			id,
			uni.BiliBili,
			uni.GiftMessageType,
			&uni.GiftMessage{
				Name:     g.Uname,
				Avatar:   avatar,
				Item:     g.GiftName,
				Num:      g.Num,
				Price:    float64(g.Num*g.Price) / 1000,
				GiftIcon: icon.ImgBasic,
				Raw:      g,
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理上舰事件
	handleGuardBuy := func(event interface{}) {
		gb := event.(*message.GuardBuy)
		user, _ := FetchUserData(gb.Uid)
		var avatar string
		if user != nil {
			avatar, _ = proxy.GenerateImageURL(user.Card.Face)
		}

		data, _ := uni.CreateUniMessage(
			id,
			uni.BiliBili,
			uni.SubscribeMessageType,
			&uni.SubscribeMessage{
				Name:   gb.Username,
				Avatar: avatar,
				Item:   ExtractGuardLevel(gb.GuardLevel),
				Num:    1,
				Price:  float64(gb.Price) / 1000,
				Raw:    gb,
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理醒目留言事件
	handleSuperChat := func(event interface{}) {
		sc := event.(*message.SuperChat)
		avatar, _ := proxy.GenerateImageURL(sc.UserInfo.Face)

		data, _ := uni.CreateUniMessage(
			id,
			uni.BiliBili,
			uni.SuperChatMessageType,
			&uni.SuperChatMessage{
				Name:    sc.UserInfo.Uname,
				Avatar:  avatar,
				Content: sc.Message,
				Price:   float64(sc.Price),
				Raw:     sc,
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理点赞事件
	handleLike := func(event interface{}) {
		var l *message.InteractWord
		_ = json.Unmarshal([]byte(event.(string)), &l)
		user, _ := FetchUserData(l.Uid)
		var avatar string
		if user != nil {
			avatar, _ = proxy.GenerateImageURL(user.Card.Face)
		}

		data, _ := uni.CreateUniMessage(
			id,
			uni.BiliBili,
			uni.LikeMessageType,
			&uni.LikeMessage{
				Name:   l.Uname,
				Avatar: avatar,
				Count:  1,
				Raw:    l,
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理进入房间事件
	handleInteract := func(event interface{}) {
		var e *message.InteractWord
		_ = json.Unmarshal([]byte(event.(string)), &e)
		user, _ := FetchUserData(e.Uid)
		var avatar string
		if user != nil {
			avatar, _ = proxy.GenerateImageURL(user.Card.Face)
		}

		data, _ := uni.CreateUniMessage(
			id,
			uni.BiliBili,
			uni.EnterRoomMessageType,
			&uni.EnterRoomMessage{
				Name:   e.Uname,
				Avatar: avatar,
				Raw:    e,
			},
		)
		ws.BroadcastToClients(data)
	}

	// 处理下播事件
	handlePreparing := func(event interface{}) {
		var p *message.Preparing
		_ = json.Unmarshal([]byte(event.(string)), &p)

		data, _ := uni.CreateUniMessage(
			id,
			uni.BiliBili,
			uni.EndLiveMessageType,
			&uni.EndLiveMessage{
				Raw: p,
			},
		)
		ws.BroadcastToClients(data)
	}

	err = gifts.InitGiftMap(room)
	if err != nil {
		log.Print("WARN", "哔哩哔哩直播间礼物获取失败")
	}

	// 定义事件处理函数映射
	eventHandlers := map[string]func(interface{}){
		"danmaku":   handleDanmaku,
		"gift":      handleGift,
		"guardBuy":  handleGuardBuy,
		"superChat": handleSuperChat,
		"like":      handleLike,
		"interact":  handleInteract,
		"preparing": handlePreparing,
	}

	c.OnDanmaku(func(d *message.Danmaku) {
		invokeHandler(eventHandlers["danmaku"], d)
	})

	c.OnGift(func(g *message.Gift) {
		invokeHandler(eventHandlers["gift"], g)
	})

	c.OnGuardBuy(func(gb *message.GuardBuy) {
		invokeHandler(eventHandlers["guardBuy"], gb)
	})

	c.OnSuperChat(func(sc *message.SuperChat) {
		invokeHandler(eventHandlers["superChat"], sc)
	})

	c.RegisterCustomEventHandler("LIKE_INFO_V3_CLICK", func(s string) {
		data := gjson.Get(s, "data").String()
		invokeHandler(eventHandlers["like"], data)
	})

	c.RegisterCustomEventHandler("INTERACT_WORD", func(s string) {
		data := gjson.Get(s, "data").String()
		invokeHandler(eventHandlers["interact"], data)
	})

	c.RegisterCustomEventHandler("PREPARING", func(s string) {
		data := gjson.Get(s, "data").String()
		invokeHandler(eventHandlers["preparing"], data)
	})

	err = c.Start()
	if err != nil {
		log.Print("ERROR", "哔哩哔哩直播监听启动失败")
		close(stopChan)
		return
	}
	log.Print("BILIBILI", "已启动哔哩哔哩直播监听")

	// 添加阻塞等待，确保在停止信号到来前不会退出
	<-ctx.Done()
}

// invokeHandler 通用的事件处理器调用函数
func invokeHandler(handler func(interface{}), event interface{}) {
	if handler != nil {
		handler(event)
	}
}
