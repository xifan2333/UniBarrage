package kuaishou

import (
	ws "UniBarrage/services/websockets"
	uni "UniBarrage/universal"
	log "UniBarrage/utils/trace"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"UniBarrage/kuaishou/protobuf/proto"
	"UniBarrage/kuaishou/utils"
	webs "UniBarrage/kuaishou/websocket"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

type KuaiShouLive struct {
	CK                       string
	address                  string
	eid                      string
	uid                      string
	liveUrl                  string
	token                    string
	livestreamId             string
	wsUrl                    string
	likeCount                int
	isFirstComputedLikeCount bool // @#@ 是否是第一次计算点赞数量 @#@
	timer                    *time.Ticker
	ws                       *webs.Socket
	giftList                 []KuaiShouGiftItem
	giftMapTimer             map[string]int64
}

// MyRequestBody @#@ 定义请求体的结构@#@
type MyRequestBody struct {
	Source      int    `json:"source"`
	Eid         string `json:"eid"`
	ShareMethod string `json:"shareMethod"`
	ClientType  string `json:"clientType"`
}

type KuaiShouGiftItem struct {
	GiftTypeName           string                 `json:"giftTypeName"`
	CanPreview             bool                   `json:"canPreview"`
	PromptMessages         map[string]interface{} `json:"promptMessages"`
	LiveGiftDescriptionKey string                 `json:"liveGiftDescriptionKey"`
	DisableMockFeed        bool                   `json:"disableMockFeed"`
	DisableMockEffect      bool                   `json:"disableMockEffect"`
	CanCombo               bool                   `json:"canCombo"`
	ActionType             int                    `json:"actionType"`
	UnitPrice              int                    `json:"unitPrice"`
	LiveGiftRuleUrl        string                 `json:"liveGiftRuleUrl"`
	MaxBatchSize           int                    `json:"maxBatchSize"`
	VirtualPrice           int                    `json:"virtualPrice"`
	PicUrl                 []struct {
		Cdn string `json:"cdn"`
		Url string `json:"url"`
	} `json:"picUrl"`
	Name string `json:"name"`
	ID   uint32 `json:"id"`
	Type int    `json:"type"`
}

func NewKuaiShouLive() *KuaiShouLive {
	return &KuaiShouLive{
		address:                  "",
		eid:                      "",
		liveUrl:                  "",
		giftMapTimer:             make(map[string]int64, 1),
		likeCount:                0,
		isFirstComputedLikeCount: true,
	}
}

// ConnectKuaiShouLiveByAddress @#@ 连接直播间 @#@
func (l *KuaiShouLive) ConnectKuaiShouLiveByAddress(address string) error {
	l.address = strings.TrimSpace(address)
	gifts, err := GetKuaiShouGiftsList()
	if err != nil {
		return err
	}
	l.giftList = gifts
	err = l.getEid()
	if err != nil {
		return err
	}

	err = l.getUserInfo()
	if err != nil {
		return err
	}
	err = l.ConnectWss()
	if err != nil {
		return err
	}
	return nil
}

// @#@ 获取EID和快手号@#@
func (l *KuaiShouLive) getEid() error {
	// @#@ 创建一个新的请求 @#@
	req, err := http.NewRequest("GET", l.address, nil)
	if err != nil {
		return err
	}

	// @#@ 添加 headers @#@
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1 Edg/122.0.0.0")
	// req.Header.Add("Cookie", "clientid=3; did=web_792599d0a9930221697c1f86eb7acc6a; didv=1711687241000")

	// @#@ 使用带有 CookieJar 的 http.Client 发送请求 @#@
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.Errorf("获取直播间LiveUrl失败,状态码: %d", resp.StatusCode)
	}

	l.liveUrl = resp.Request.URL.String() // @#@获取最终的 URL@#@

	if l.liveUrl == "" {
		return errors.New("获取直播间liveUrl失败,请稍后再试...")
	}
	//@#@ 定义正则表达式，注意 Go 语言中不需要前缀@  @#@
	reg := regexp.MustCompile(`(?i)/fw/live/(?P<eid>\w+)\?`)
	match := reg.FindStringSubmatch(l.liveUrl)

	var eid string

	// @#@检查是否有匹配@#@
	if match != nil {
		eidIndex := reg.SubexpIndex("eid") // @#@获取命名捕获组的索引@#@
		eid = match[eidIndex]              // @#@通过索引获取 eid 的值@#@
	} else {
		// @#@如果第一个正则表达式没有匹配，尝试第二个@#@
		reg = regexp.MustCompile(`(?i)/fw/photo/(?P<eid>\w+)\?`)
		match = reg.FindStringSubmatch(l.liveUrl)

		if match != nil {
			eidIndex := reg.SubexpIndex("eid")
			eid = match[eidIndex]
		}
	}

	// @#@如果都没有匹配，eid 将保持为空@#@
	if eid == "" {
		return errors.New("获取直播间EID为空,请稍后再试...")
	}
	l.eid = eid
	l.uid = extractUserId(l.liveUrl)
	return nil
}

func extractUserId(url string) string {
	// @#@编译正则表达式，用于匹配 userId 参数@#@
	regex := regexp.MustCompile(`userId=(\d+)`)
	// @#@使用 FindStringSubmatch 找到全部匹配的字符串@#@
	matches := regex.FindStringSubmatch(url)

	// @#@检查是否有匹配结果@#@
	if len(matches) > 1 {
		// @#@返回第一个括号中匹配到的结果，即 userId 的值@#@
		return matches[1]
	}

	return "" // @#@如果没有匹配到，则返回空字符串@#@
}

// @#@ 获取用户信息 @#@
func (l *KuaiShouLive) getUserInfo() error {
	// @#@创建请求体@#@
	body := MyRequestBody{
		Source:      6,
		Eid:         l.eid,
		ShareMethod: "card",
		ClientType:  "WEB_OUTSIDE_SHARE_H5",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return errors.Errorf("请求体序列化失败:%s", err.Error())
	}

	// @#@创建 HTTP 客户端和请求@#@
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://v.m.chenzhongtech.com/rest/k/live/byUser?kpn=KUAISHOU&kpf=OUTSIDE_IOS_H5&captchaToken=", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return errors.Errorf("创建请求失败:%s", err.Error())
	}

	// @#@添加请求头@#@
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1 Edg/122.0.0.0")
	req.Header.Set("Host", "v.m.chenzhongtech.com")
	req.Header.Set("Origin", "https://v.m.chenzhongtech.com")
	req.Header.Set("Referer", l.liveUrl)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
	// req.Header.Add("Cookie", "did=web_607d1555716144e2b3ab622c203e0532; didv=1711531744000")
	// @#@发送请求@#@
	resp, err := client.Do(req)
	if err != nil {
		return errors.Errorf("发送请求失败:%s", err.Error())
	}
	defer resp.Body.Close()

	// @#@解析响应内容@#@
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf("解析响应失败:%s", err.Error())
	}
	// 首先解压数据
	reader, err := gzip.NewReader(bytes.NewReader(responseBody))
	if err != nil {
		return errors.Errorf("解压失败:%s", err.Error())
	}
	defer reader.Close()

	decompressedBytes, err := io.ReadAll(reader)
	if err != nil {
		return errors.Errorf("reading decompressed失败:%s", err.Error())
	}

	bodystr := string(decompressedBytes)

	if gjson.Get(bodystr, "result").String() != "1" {
		return errors.New("获取用户信息失败,请稍后再试...")
	}

	l.token = gjson.Get(bodystr, "token").String()
	l.livestreamId = gjson.Get(bodystr, "liveStream.liveStreamId").String()
	if gjson.Get(bodystr, "webSocketAddresses").String() == "" {
		return errors.New("获取WebSocket地址失败,有可能下播了...")
	}
	l.wsUrl = gjson.Get(bodystr, "webSocketAddresses").Array()[0].String()

	return nil
}

// ConnectWss @#@ 连接websocket @#@
func (l *KuaiShouLive) ConnectWss() error {
	socket := webs.New(l.wsUrl)

	socket.OnConnected = func(socket webs.Socket) {
		if l.ws == nil {
			l.ws = &socket
		}
		l.enterRoom()
		l.heartBeat()
	}

	socket.OnConnectError = func(err error, socket webs.Socket) {
		//println("Received connect error ", err.Error())
		l.ConnectKuaiShouLiveByAddress(l.address)
	}

	socket.OnTextMessage = func(message string, socket webs.Socket) {
		//println("Received message " + message)
	}

	socket.OnBinaryMessage = func(data []byte, socket webs.Socket) {
		l.parseMsg(data)
	}

	socket.OnPingReceived = func(data string, socket webs.Socket) {
		//println("Received ping " + data)
	}

	socket.OnPongReceived = func(data string, socket webs.Socket) {
		//println("Received pong " + data)
	}

	socket.OnDisconnected = func(err error, socket webs.Socket) {
		//println("Disconnected from server：", err.Error())
		log.Print("WARN", "与快手服务器断开连接, 尝试重新连接")
		// @#@ 直播间连接中断，正在重连... @#@
		if strings.Contains(err.Error(), "websocket: close 1006 (abnormal closure): unexpected EOF") {
			//fmt.Print("可能没开播哦......")
			log.Print("ERROR", "未开启快手直播")
			return
		}
		l.ConnectKuaiShouLiveByAddress(l.address)
	}

	socket.Connect()

	l.ws = &socket
	return nil
}

// @#@ 进入房间 @#@
func (l *KuaiShouLive) enterRoom() {
	// @#@ 构建消息内容 @#@
	part1 := []byte{0x08, 0xC8, 0x01, 0x1A, 0xC5, 0x01, 0x0A, 0x98, 0x01}
	part2 := []byte(l.token) // 替换成实际的 Token
	part3 := []byte{0x12, 0x0B}
	part4 := []byte(l.livestreamId) // 替换成实际的 LiveStreamId
	part5 := []byte{0x42, 0x0B}
	part6 := []byte("KUAISHOU_H5")
	part7 := []byte{0x4A, 0x0E}
	part8 := []byte("OUTSIDE_IOS_H5")
	// @#@ 合并所有部分 @#@
	var message []byte
	message = append(message, part1...)
	message = append(message, part2...)
	message = append(message, part3...)
	message = append(message, part4...)
	message = append(message, part5...)
	message = append(message, part6...)
	message = append(message, part7...)
	message = append(message, part8...)
	l.ws.SendBinary(message)
	//fmt.Println("发送进入房间消息")
}

// @#@ 发送心跳 @#@
func (l *KuaiShouLive) heartBeat() {
	if l.timer != nil {
		return
	}
	ctx, _ := context.WithCancel(context.Background())
	// @#@ 20秒发一次心跳包 @#@
	l.timer = time.NewTicker(20 * time.Second)
	go func() {
		for {
			select {
			case <-l.timer.C:

				l.ws.SendBinary([]byte{0x08, 0x01, 0x1A, 0x07, 0x08, 0xE7, 0xB5, 0xBA, 0xC7, 0xE8, 0x31})
			case <-ctx.Done(): // @#@ 检查context是否已经被取消 @#@
				return // @#@ 如果已经被取消，就退出协程 @#@
			}
		}
	}()
}

// @#@ 解析消息 @#@
func (l *KuaiShouLive) parseMsg(data []byte) {
	if len(data) == 0 || data[0] != 0x08 {
		return
	}
	receiveMessage := &proto.SocketMessage{}
	err := receiveMessage.Unmarshal(data)
	if err != nil {
		fmt.Print("解析消息失败:", err.Error())
		return
	}

	switch receiveMessage.CompressionType {
	case proto.CompressionType_NONE:
		break
	case proto.CompressionType_GZIP:
		payload, err := utils.GzipDecode(receiveMessage.Payload)
		if err != nil {
			return
		}
		receiveMessage.Payload = payload
	}
	if receiveMessage.PayloadType == proto.PayloadType_SC_FEED_PUSH {
		msg := &proto.SCWebFeedPush{}
		err := msg.Unmarshal(receiveMessage.Payload)
		if err != nil {
			return
		}
		// @#@ 弹幕 @#@
		if msg.CommentFeeds != nil && len(msg.CommentFeeds) > 0 {
			for _, c := range msg.CommentFeeds {
				// 格式化时间为 yyyy-mm-dd hh:mm:ss
				//t := time.Unix(time.Now().Unix(), 0)
				//fmt.Print(t.Format("2006-01-02 15:04:05"), " ", "评论消息:", c.User.UserName, "说：", c.Content, "---用户ID：", c.User.PrincipalId, "\n")
				user, _ := GetUserInfo(c.User.PrincipalId, l.CK)
				data, _ := uni.CreateUniMessage(
					l.address[strings.LastIndex(l.address, "/"):],
					uni.KuaiShou,
					uni.ChatMessageType,
					&uni.ChatMessage{
						Name:     c.User.UserName,
						Avatar:   user.Data.VisionProfile.UserProfile.Profile.HeadURL,
						Content:  c.Content,
						Emoticon: []string{},
						Raw:      c,
					},
				)
				ws.BroadcastToClients(data)
			}
		}
		// @#@ 组合弹幕 @#@
		if msg.ComboCommentFeed != nil && len(msg.ComboCommentFeed) > 0 {
			//for _, cc := range msg.ComboCommentFeed {
			//	fmt.Print("组合评论消息:", cc.Content, " X ", cc.ComboCount, "\n")
			//}
		}
		// @#@ 点赞 @#@
		if msg.DisplayLikeCount != "" && msg.DisplayLikeCount != "0" {
			count, _ := strconv.Atoi(msg.DisplayLikeCount)
			incre := count - l.likeCount // @#@ 增量 @#@
			if l.isFirstComputedLikeCount {
				incre = 0
			}
			if incre != 0 {
				//fmt.Sprintf("【点赞】主播收到了%v个赞", incre)
			}
			l.isFirstComputedLikeCount = false
			l.likeCount = count
		}

		// @#@ 点亮❤️ 好像同一个人点亮一次以后，就不会触发了 @#@
		if msg.LikeFeeds != nil && len(msg.LikeFeeds) > 0 {
			for _, like := range msg.LikeFeeds {
				fmt.Print("点赞消息:", like.User.UserName, "给主播点了赞", "\n")
				user, _ := GetUserInfo(like.User.PrincipalId, l.CK)
				data, _ := uni.CreateUniMessage(
					l.address[strings.LastIndex(l.address, "/"):],
					uni.KuaiShou,
					uni.LikeMessageType,
					&uni.LikeMessage{
						Name:   like.User.UserName,
						Avatar: user.Data.VisionProfile.UserProfile.Profile.HeadURL,
						Count:  1,
						Raw:    like,
					},
				)
				ws.BroadcastToClients(data)
			}
		}

		// @#@ 礼物 @#@
		if msg.GiftFeeds != nil && len(msg.GiftFeeds) > 0 {
			for _, gift := range msg.GiftFeeds {
				giftName := ""
				price := 0
				giftIcon := ""
				for i := 0; i < len(l.giftList); i++ {
					item := l.giftList[i]
					if item.ID == gift.GiftId {
						giftName = item.Name
						price = item.UnitPrice
						giftIcon = item.PicUrl[0].Url
						break
					}
				}
				// 格式化时间为 yyyy-mm-dd hh:mm:ss
				// t := time.Unix(time.Now().Unix(), 0)
				// fmt.Print(t.Format("2006-01-02 15:04:05"), " ", "礼物消息:", gift.User.UserName, "送给主播【", giftName, "】，共：", gift.ComboCount, "个", "---用户ID：", gift.User.PrincipalId, "\n")
				user, _ := GetUserInfo(gift.User.PrincipalId, l.CK)
				data, _ := uni.CreateUniMessage(
					l.address[strings.LastIndex(l.address, "/"):],
					uni.KuaiShou,
					uni.GiftMessageType,
					&uni.GiftMessage{
						Name:     gift.User.UserName,
						Avatar:   user.Data.VisionProfile.UserProfile.Profile.HeadURL,
						Item:     giftName,
						Num:      int(gift.ComboCount),
						Price:    float64(price),
						GiftIcon: giftIcon,
						Raw:      gift,
					},
				)
				ws.BroadcastToClients(data)
			}
		}
		// if msg.ShareFeeds != nil && len(msg.ShareFeeds) > 0 { {
		// 	fmt.Print("分享消息:", msg.ShareFeeds.User.UserName, "分享直播间到", msg.ShareFeeds.ThirdPartyPlatform, "\n")
		// }
		// if msg.SystemNoticeFeeds != nil && len(msg.SystemNoticeFeeds) > 0 {
		// 	fmt.Print("系统通知消息:", msg.SystemNoticeFeeds.Content, "\n")
		// }
	}
	// @#@ 下播？？ @#@
	if receiveMessage.PayloadType == proto.PayloadType_SC_LIVE_CHAT_ENDED {
		// @#@ 直播间状态变更 @#@
		//println(">>>>>>>>>>>>>>>>>>>>>>直播间已关闭，直播已经结束了<<<<<<<<<<<<<<<<<<<<<<<")
		data, _ := uni.CreateUniMessage(
			l.address[strings.LastIndex(l.address, "/"):],
			uni.KuaiShou,
			uni.EndLiveMessageType,
			&uni.EndLiveMessage{
				Raw: proto.PayloadType_SC_LIVE_CHAT_ENDED,
			},
		)
		ws.BroadcastToClients(data)
	}

	// if receiveMessage.PayloadType == proto.PayloadType_CS_ENTER_ROOM {
	// 	room := &proto.CSWebEnterRoom{}
	// 	err := room.Unmarshal(receiveMessage.Payload)
	// 	if err != nil {
	// 		return
	// 	}
	// }
	// if receiveMessage.PayloadType == proto.PayloadType_SC_ENTER_ROOM_ACK {
	// 	scWebEnterRoomAck := &proto.SCWebEnterRoomAck{}
	// 	err := scWebEnterRoomAck.Unmarshal(receiveMessage.Payload)
	// 	if err != nil {
	// 		return
	// 	}
	// }
	// if receiveMessage.PayloadType == proto.PayloadType_SC_LIVE_WATCHING_LIST {
	// 	// @#@ 直播观看列表 @#@
	// 	res := &proto.SCWebLiveWatchingUsers{}
	// 	err := res.Unmarshal(receiveMessage.Payload)
	// 	if err != nil {
	// 		return
	// 	}
	// }
}

// GetKuaiShouGiftsList @#@ 获取快手礼物列表 @#@
func GetKuaiShouGiftsList() ([]KuaiShouGiftItem, error) {
	// @#@创建 HTTP 客户端和请求@#@
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://v.m.chenzhongtech.com/rest/wd/live/gift/all", nil)
	if err != nil {
		return nil, errors.Errorf("创建获取礼物列表请求失败:%s", err.Error())
	}
	// @#@发送请求@#@
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Errorf("发送获取礼物列表请求失败:%s", err.Error())
	}
	defer resp.Body.Close()

	// @#@解析响应内容@#@
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("解析响应失败:%s", err.Error())
	}
	bodystr := string(responseBody)
	if gjson.Get(bodystr, "result").String() != "1" {
		return nil, errors.New("获取礼物列表失败,请稍后再试...")
	}

	giftJson := gjson.Get(bodystr, "gifts").String()
	// 将giftJson转化为[]KuaiShouGiftItem
	var giftList []KuaiShouGiftItem
	err = json.Unmarshal([]byte(giftJson), &giftList)
	if err != nil {
		fmt.Println("解析JSON时出现错误:", err)
		return nil, errors.Errorf("解析礼物列表JSON时出现错误:%s", err.Error())
	}
	var giftMap = make(map[string]bool, 1)
	var gifts []KuaiShouGiftItem
	for _, v := range giftList {
		_, exist := giftMap[v.Name]
		if !exist {
			gifts = append(gifts, v)
			giftMap[v.Name] = true
		}
	}
	return gifts, nil
}

type UpdateLiveAddressJSON struct {
	Uid         string `json:"uid"`
	Liveaddress string `json:"liveaddress"`
}

// @#@ 快手直播每次开播前，需要更新直播间地址 @#@

func (l *KuaiShouLive) Stop() {
	// 停止计时器
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}

	// 关闭 WebSocket
	if l.ws != nil {
		l.ws.Close()
		l.ws = nil
	}

	// 清空礼物列表
	l.giftList = nil

	// 清空礼物计时器
	l.giftMapTimer = nil

	//// 打印日志
	//fmt.Println("All resources for KuaiShouLive have been cleaned up.")
}

func StartListen(liveAddress string, cookie string, stopChan chan struct{}) {
	var live = NewKuaiShouLive()
	live.CK = cookie

	// 创建一个 context 用于控制
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听 stopChan
	go func() {
		<-stopChan
		cancel()
		live.Stop()
		//if live.ws != nil {
		//	live.ws.Close()
		//}
	}()

	err := live.ConnectKuaiShouLiveByAddress("https://v.kuaishou.com/" + liveAddress)
	if err != nil {
		log.Printf("ERROR", "快手直播监听启动失败: %s", err.Error())
		close(stopChan)
		return
	}

	log.Print("KUAISHOU", "已启动快手直播监听")

	// 等待结束信号
	<-ctx.Done()
}
