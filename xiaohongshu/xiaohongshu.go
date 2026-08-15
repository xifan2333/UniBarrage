package xiaohongshu

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	ws "UniBarrage/services/websockets"
	uni "UniBarrage/universal"
	log "UniBarrage/utils/trace"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

const (
	rwpURL     = "wss://apppush-rws.xiaohongshu.com/rwp"
	pingEvery  = 8 * time.Second
	heartEvery = 30 * time.Second
)

// StartListen starts Xiaohongshu live danmaku for roomId (livestream id string).
// cookie is optional (currently unused for tourist path; reserved for logged-in).
func StartListen(roomID string, cookie string, stopChan chan struct{}) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		log.Print("ERROR", "小红书房间 ID 为空")
		close(stopChan)
		return
	}
	_ = cookie

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopChan
		cancel()
	}()

	// reconnect loop
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			log.Print("INFO", "已停止小红书直播监听")
			return
		}
		err := listenOnce(ctx, roomID)
		if ctx.Err() != nil {
			log.Print("INFO", "已停止小红书直播监听")
			return
		}
		if err != nil {
			log.Printf("ERROR", "小红书监听中断: %v；%s 后重连", err, backoff)
		} else {
			log.Printf("WARN", "小红书连接结束，%s 后重连", backoff)
		}
		select {
		case <-ctx.Done():
			log.Print("INFO", "已停止小红书直播监听")
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func listenOnce(ctx context.Context, roomID string) error {
	sess, err := CreateGuestSession()
	if err != nil {
		return fmt.Errorf("guest session: %w", err)
	}
	log.Printf("XIAOHONGSHU", "游客会话就绪 uid=%s device=%s", sess.UserID, sess.DeviceID)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}
	header := http.Header{}
	header.Set("User-Agent", userAgent)
	header.Set("Origin", "https://www.xiaohongshu.com")
	conn, _, err := dialer.DialContext(ctx, rwpURL, header)
	if err != nil {
		return fmt.Errorf("dial rwp: %w", err)
	}
	defer conn.Close()

	var writeMu sync.Mutex

	// stop writer on ctx done
	go func() {
		<-ctx.Done()
		writeMu.Lock()
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(2*time.Second))
		writeMu.Unlock()
		_ = conn.Close()
	}()

	if err := handshake(conn, &writeMu, sess, roomID); err != nil {
		return err
	}
	log.Printf("XIAOHONGSHU", "已启动小红书直播监听 room=%s", roomID)

	// heartbeat / ping
	go keepAlive(ctx, conn, &writeMu, sess, roomID)

	conn.SetReadLimit(4 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		handleFrame(roomID, data)
	}
}

func mid() string {
	return fmt.Sprintf("%x-%x", rand.Int63(), time.Now().UnixMilli())
}

func writeJSON(conn *websocket.Conn, mu *sync.Mutex, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, b)
}

// readUntilAck reads frames until a signaling ack (t=2 with b.a) is seen.
// Interleaved t=4 push frames are dispatched immediately.
func readUntilAck(conn *websocket.Conn, roomID string, expectMid string, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		t, _ := m["t"].(float64)
		if int(t) == 4 {
			handleFrame(roomID, data)
			continue
		}
		if expectMid != "" {
			if mid, _ := m["m"].(string); mid != "" && mid != expectMid {
				// unrelated signaling; keep waiting
				continue
			}
		}
		if ackOK(m) || hasAckBody(m) {
			return m, nil
		}
	}
	return nil, fmt.Errorf("timeout waiting ack")
}

func hasAckBody(m map[string]any) bool {
	b, _ := m["b"].(map[string]any)
	if b == nil {
		return false
	}
	_, ok := b["a"].(map[string]any)
	return ok
}

func ackOK(m map[string]any) bool {
	// b.a.c == 0
	b, _ := m["b"].(map[string]any)
	if b == nil {
		return false
	}
	a, _ := b["a"].(map[string]any)
	if a == nil {
		return false
	}
	switch c := a["c"].(type) {
	case float64:
		return c == 0
	case int:
		return c == 0
	case json.Number:
		n, _ := c.Int64()
		return n == 0
	default:
		return false
	}
}

func handshake(conn *websocket.Conn, mu *sync.Mutex, sess *GuestSession, roomID string) error {
	fp := fmt.Sprintf("%d", time.Now().UnixMilli())
	// 1 auth s=0
	authMid := mid()
	auth := map[string]any{
		"v": 1, "t": 2, "m": authMid,
		"b": map[string]any{
			"d": map[string]any{
				"a": 1, "s": 0,
				"b": map[string]any{
					"appId": "xhs-pc",
					"authInfo": map[string]any{
						"authType": "generic",
						"sid":      sess.ALt,
						"uid":      sess.UserID,
						"domain":   "red",
					},
					"deviceInfo": map[string]any{
						"deviceId":    sess.DeviceID,
						"fingerprint": fp,
						"platform":    "browser",
						"os":          "web",
						"osVersion":   "10.15",
						"deviceName":  "Chrome",
						"appVersion":  "131.0.0.0",
						"userAgent":   userAgent,
					},
					"serviceTag": "",
					"bizInfos":   []map[string]any{{"bizName": "push", "serializeType": "json"}},
					"roomInfo":   []any{},
					"roomInfos":  []any{},
					"tagInfo":    []any{},
					"extInfo":    map[string]any{},
					"state":      1,
				},
			},
		},
	}
	if err := writeJSON(conn, mu, auth); err != nil {
		return fmt.Errorf("auth send: %w", err)
	}
	resp, err := readUntilAck(conn, roomID, authMid, 15*time.Second)
	if err != nil {
		return fmt.Errorf("auth recv: %w", err)
	}
	if !ackOK(resp) {
		return fmt.Errorf("auth failed: %v", resp)
	}

	// 2 register s=1
	regMid := mid()
	reg := map[string]any{
		"v": 1, "t": 2, "m": regMid,
		"b": map[string]any{"d": map[string]any{
			"a": 1, "s": 1,
			"b": map[string]any{
				"bizInfo":  map[string]any{"bizName": "room", "serializeType": "json"},
				"register": true,
			},
		}},
	}
	if err := writeJSON(conn, mu, reg); err != nil {
		return fmt.Errorf("register send: %w", err)
	}
	resp, err = readUntilAck(conn, roomID, regMid, 15*time.Second)
	if err != nil {
		return fmt.Errorf("register recv: %w", err)
	}
	if !ackOK(resp) {
		return fmt.Errorf("register failed: %v", resp)
	}

	// 3 join s=8
	joinMid := mid()
	join := map[string]any{
		"v": 1, "t": 2, "m": joinMid,
		"b": map[string]any{"d": map[string]any{
			"a": 1, "s": 8,
			"b": map[string]any{
				"info": map[string]any{
					"bizName":  "room",
					"roomId":   roomID,
					"roomType": "LIVE",
				},
			},
		}},
	}
	if err := writeJSON(conn, mu, join); err != nil {
		return fmt.Errorf("join send: %w", err)
	}
	resp, err = readUntilAck(conn, roomID, joinMid, 15*time.Second)
	if err != nil {
		return fmt.Errorf("join recv: %w", err)
	}
	if !ackOK(resp) {
		return fmt.Errorf("join failed: %v", resp)
	}

	// initial heartbeat
	_ = sendHeartbeat(conn, mu, sess, roomID)
	return nil
}

func sendHeartbeat(conn *websocket.Conn, mu *sync.Mutex, sess *GuestSession, roomID string) error {
	custom, _ := json.Marshal(map[string]any{
		"type":     "viewer_heart",
		"priority": 0,
		"profile": map[string]any{
			"nickname": "",
			"avatar":   "",
			"user_id":  sess.UserID,
			"role":     0,
		},
		"source": "web_live",
		"desc":   "",
	})
	bodyObj := map[string]any{
		"roomId":     roomID,
		"roomType":   "LIVE",
		"command":    1,
		"customData": string(custom),
	}
	bodyRaw, _ := json.Marshal(bodyObj)
	hb := map[string]any{
		"v": 1, "t": 3, "m": mid(),
		"b": map[string]any{"d": map[string]any{
			"a":   0,
			"c":   "liveHeartBeat",
			"biz": "room",
			"b":   base64.StdEncoding.EncodeToString(bodyRaw),
			"e":   map[string]any{},
			"s":   "rrmp.o.l",
		}},
	}
	return writeJSON(conn, mu, hb)
}

func sendPing(conn *websocket.Conn, mu *sync.Mutex) error {
	ping := map[string]any{
		"v": 1, "t": 2, "m": mid(),
		"b": map[string]any{"d": map[string]any{
			"a": 1, "s": 6, "b": map[string]any{},
		}},
	}
	return writeJSON(conn, mu, ping)
}

func keepAlive(ctx context.Context, conn *websocket.Conn, mu *sync.Mutex, sess *GuestSession, roomID string) {
	pingT := time.NewTicker(pingEvery)
	heartT := time.NewTicker(heartEvery)
	defer pingT.Stop()
	defer heartT.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pingT.C:
			if err := sendPing(conn, mu); err != nil {
				return
			}
		case <-heartT.C:
			if err := sendHeartbeat(conn, mu, sess, roomID); err != nil {
				return
			}
		}
	}
}

// ---------- frame decode ----------

type rwpOuter struct {
	V int             `json:"v"`
	T int             `json:"t"`
	M string          `json:"m"`
	B json.RawMessage `json:"b"`
}

type pushBody struct {
	D struct {
		A   int `json:"a"`
		B   []struct {
			D string `json:"d"`
			M string `json:"m"`
		} `json:"b"`
		Biz string `json:"biz"`
		T   int64  `json:"t"`
	} `json:"d"`
}

type roomMsg struct {
	Command    int             `json:"command"`
	CustomData json.RawMessage `json:"customData"`
	MsgID      string          `json:"msgId"`
	Priority   int             `json:"priority"`
	RoomID     string          `json:"roomId"`
	RoomType   string          `json:"roomType"`
	Ts         int64           `json:"ts"`
	UUID       string          `json:"uuid"`
}

func handleFrame(roomID string, raw []byte) {
	var outer rwpOuter
	if err := json.Unmarshal(raw, &outer); err != nil {
		return
	}
	if outer.T != 4 {
		return
	}
	var pb pushBody
	if err := json.Unmarshal(outer.B, &pb); err != nil {
		return
	}
	for _, item := range pb.D.B {
		if item.D == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(item.D)
		if err != nil {
			// try raw URL encoding variants
			decoded, err = base64.RawStdEncoding.DecodeString(item.D)
			if err != nil {
				continue
			}
		}
		var rm roomMsg
		if err := json.Unmarshal(decoded, &rm); err != nil {
			continue
		}
		// customData may be a JSON string or object
		var cd map[string]any
		if len(rm.CustomData) > 0 {
			if rm.CustomData[0] == '"' {
				var s string
				if err := json.Unmarshal(rm.CustomData, &s); err == nil {
					_ = json.Unmarshal([]byte(s), &cd)
				}
			} else {
				_ = json.Unmarshal(rm.CustomData, &cd)
			}
		}
		if cd == nil {
			continue
		}
		emitMessage(roomID, cd, rm)
	}
}

func strField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if t != "" {
					return t
				}
			case float64:
				return fmt.Sprintf("%.0f", t)
			}
		}
	}
	return ""
}

func nest(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func intField(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case int:
				return t
			case json.Number:
				n, _ := t.Int64()
				return int(n)
			case string:
				var n int
				fmt.Sscanf(t, "%d", &n)
				return n
			}
		}
	}
	return 0
}

func floatField(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return t
			case int:
				return float64(t)
			case json.Number:
				n, _ := t.Float64()
				return n
			}
		}
	}
	return 0
}

func profileOf(cd map[string]any) (name, avatar, uid string) {
	p := nest(cd, "profile")
	if p == nil {
		p = nest(cd, "user_info")
	}
	if p == nil {
		p = nest(cd, "send_user_info")
	}
	if p == nil {
		return "", "", ""
	}
	name = strField(p, "nickname", "nick_name", "name")
	avatar = strField(p, "avatar")
	uid = strField(p, "user_id", "id", "userId")
	return
}

func emitMessage(roomID string, cd map[string]any, raw roomMsg) {
	typ := strField(cd, "type")
	switch typ {
	case "text", "text_message":
		name, avatar, _ := profileOf(cd)
		content := strField(cd, "desc", "content", "text")
		if content == "" {
			return
		}
		data, _ := uni.CreateUniMessage(roomID, uni.XiaoHongShu, uni.ChatMessageType, &uni.ChatMessage{
			Name:     name,
			Avatar:   avatar,
			Content:  content,
			Emoticon: nil,
			Raw:      cd,
		})
		ws.BroadcastToClients(data)

	case "audience_join_v2", "fansgroup_join_room_effect":
		name, avatar, _ := profileOf(cd)
		if name == "" {
			// fansgroup uses user_info
			ui := nest(cd, "user_info")
			name = strField(ui, "nickname", "nick_name")
			avatar = strField(ui, "avatar")
		}
		data, _ := uni.CreateUniMessage(roomID, uni.XiaoHongShu, uni.EnterRoomMessageType, &uni.EnterRoomMessage{
			Name:   name,
			Avatar: avatar,
			Raw:    cd,
		})
		ws.BroadcastToClients(data)

	case "praise", "like", "combo_praise", "light", "like_comment", "live_like", "live_common_msg_action":
		name, avatar, _ := profileOf(cd)
		count := 1
		if pi := nest(cd, "praise_info"); pi != nil {
			if c := intField(pi, "count"); c > 0 {
				count = c
			}
		}
		data, _ := uni.CreateUniMessage(roomID, uni.XiaoHongShu, uni.LikeMessageType, &uni.LikeMessage{
			Name:   name,
			Avatar: avatar,
			Count:  count,
			Raw:    cd,
		})
		ws.BroadcastToClients(data)

	case "gift_dock_and_effect", "gift_comment", "gift_settle":
		su := nest(cd, "send_user_info")
		name := strField(su, "nick_name", "nickname", "name")
		avatar := strField(su, "avatar")
		gi := nest(cd, "base_gift_info")
		item := strField(gi, "name")
		icon := strField(gi, "icon")
		coins := floatField(gi, "coins", "price")
		num := intField(cd, "count", "num")
		if num <= 0 {
			num = 1
		}
		data, _ := uni.CreateUniMessage(roomID, uni.XiaoHongShu, uni.GiftMessageType, &uni.GiftMessage{
			Name:     name,
			Avatar:   avatar,
			Item:     item,
			Num:      num,
			Price:    coins * float64(num),
			GiftIcon: icon,
			Raw:      cd,
		})
		ws.BroadcastToClients(data)

	case "follow_emcee":
		name, avatar, _ := profileOf(cd)
		data, _ := uni.CreateUniMessage(roomID, uni.XiaoHongShu, uni.SubscribeMessageType, &uni.SubscribeMessage{
			Name:   name,
			Avatar: avatar,
			Item:   "follow",
			Num:    1,
			Price:  0,
			Raw:    cd,
		})
		ws.BroadcastToClients(data)

	case "letter_refresh", "viewer_heart", "refresh", "room_func_state_change",
		"live_banner_resource", "goods_rank_entrance_im", "linkmic_score_change",
		"linkmic_switch_audio":
		// ignore noise
		return
	default:
		// unknown — ignore quietly
		_ = raw
		return
	}
}
