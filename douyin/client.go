package douyin

import (
	"UniBarrage/douyin/generated/douyin"
	"UniBarrage/douyin/jsScript"
	"UniBarrage/douyin/utils"
	"UniBarrage/utils/trace"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/imroc/req/v3"
	"github.com/spf13/cast"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 正则表达式用于提取 roomID 和 pushID
var (
	roomIDRegexp = regexp.MustCompile(`roomId\\":\\"(\d+)\\"`)
	pushIDRegexp = regexp.MustCompile(`user_unique_id\\":\\"(\d+)\\"`)
)

// DouyinLive 结构体表示一个抖音直播连接

// NewDouyinLive 创建一个新的 DouyinLive 实例
func NewDouyinLive(liveid string) (*DouyinLive, error) {
	ua := utils.RandomUserAgent()
	c := req.C().SetUserAgent(ua)
	d := &DouyinLive{
		liveid:        liveid,
		liveurl:       "https://live.douyin.com/",
		userAgent:     ua,
		c:             c,
		eventHandlers: make([]EventHandler, 0),
		headers:       http.Header{},
		buffers: &sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			}},
		startedCh: make(chan struct{}),
		stopCh:    make(chan struct{}),
	}

	// 获取 ttwid
	var err error
	d.ttwid, err = d.fetchTTWID()
	if err != nil {
		return nil, fmt.Errorf("获取 TTWID 失败: %w", err)
	}

	// 获取 roomid
	d.roomid = d.fetchRoomID()

	// 加载 JavaScript 脚本
	err = jsScript.LoadGoja(d.userAgent)
	if err != nil {
		return nil, fmt.Errorf("加载 Goja 脚本失败: %w", err)
	}
	return d, nil
}

// fetchTTWID 获取 ttwid
func (d *DouyinLive) fetchTTWID() (string, error) {
	if d.ttwid != "" {
		return d.ttwid, nil
	}

	res, err := d.c.R().Get(d.liveurl)
	if err != nil {
		return "", fmt.Errorf("获取直播 URL 失败: %w", err)
	}

	for _, cookie := range res.Cookies() {
		if cookie.Name == "ttwid" {
			d.ttwid = cookie.Value
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("未找到 ttwid cookie")
}

// fetchRoomID 获取 roomID
func (d *DouyinLive) fetchRoomID() string {
	if d.roomid != "" {
		return d.roomid
	}

	t, _ := d.fetchTTWID()
	ttwid := &http.Cookie{
		Name:  "ttwid",
		Value: "ttwid=" + t + "&msToken=" + utils.GenerateMsToken(107),
	}
	acNonce := &http.Cookie{
		Name:  "__ac_nonce",
		Value: "0123407cc00a9e438deb4",
	}
	res, err := d.c.R().SetCookies(ttwid, acNonce).Get(d.liveurl + d.liveid)
	if err != nil {
		log.Printf("获取房间 ID 失败: %v", err)
		return ""
	}

	d.roomid = extractMatch(roomIDRegexp, res.String())
	d.pushid = extractMatch(pushIDRegexp, res.String())
	return d.roomid
}

// extractMatch 从字符串中提取正则表达式匹配的内容
func extractMatch(re *regexp.Regexp, s string) string {
	match := re.FindStringSubmatch(s)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// GzipUnzipReset 使用 Reset 方法解压 gzip 数据
func (d *DouyinLive) GzipUnzipReset(compressedData []byte) ([]byte, error) {
	var err error
	buffer := d.buffers.Get().(*bytes.Buffer)
	defer func() {
		buffer.Reset()
		d.buffers.Put(buffer)
	}()

	_, err = buffer.Write(compressedData)
	if err != nil {
		return nil, err
	}

	if d.gzip != nil {
		err = d.gzip.Reset(buffer)
		if err != nil {
			d.gzip.Close()
			d.gzip = nil
			return nil, err
		}
	} else {
		d.gzip, err = gzip.NewReader(buffer)
		if err != nil {
			return nil, err
		}
	}
	defer d.gzip.Close()

	uncompressedBuffer := &bytes.Buffer{}
	_, err = io.Copy(uncompressedBuffer, d.gzip)
	if err != nil {
		return nil, err
	}

	return uncompressedBuffer.Bytes(), nil
}

// Start 开始连接和处理消息
func (d *DouyinLive) Start() error {
	d.startWg.Add(1)
	defer d.startWg.Done()

	var err error
	d.wssurl = d.StitchUrl()
	d.headers.Add("user-agent", d.userAgent)
	d.headers.Add("cookie", fmt.Sprintf("ttwid=%s", d.ttwid))
	//var response *http.Response
	//d.Conn, response, err = websocket.DefaultDialer.Dial(d.wssurl, d.headers)
	d.Conn, _, err = websocket.DefaultDialer.Dial(d.wssurl, d.headers)
	if err != nil {
		trace.Print("ERROR", "与抖音服务链接失败")
		//log.Printf("链接失败: err:%v\nroomid:%v\n ttwid:%v\nwssurl:----%v\nresponse:%v\n", err, d.roomid, d.ttwid, d.wssurl, response)
		return err
	}
	//log.Println("链接成功")
	trace.Print("INFO", "与抖音服务链接成功")
	d.isLiveClosed = true
	defer func() {
		if d.gzip != nil {
			err := d.gzip.Close()
			if err != nil {
				//log.Println("gzip关闭失败:", err)
			} else {
				//log.Println("gzip关闭")
			}
		}
		if d.Conn != nil {
			err = d.Conn.Close()
			if err != nil {
				//log.Println("关闭ws链接失败", err)
			} else {
				//log.Println("抖音ws链接关闭")
			}
		}
	}()
	var pbPac = &douyin.PushFrame{}
	var pbResp = &douyin.Response{}
	var pbAck = &douyin.PushFrame{}

	// 连接成功后发送信号
	close(d.startedCh)

	// 修改主循环，添加对 stopCh 的监听
	for d.isLiveClosed {
		select {
		case <-d.stopCh:
			return nil
		default:
			_, message, err := d.Conn.ReadMessage()
			// messageType, message, err := d.Conn.ReadMessage()
			if err != nil {
				//log.Println("读取消息失败-", err, message, messageType)
				//trace.Printf("WARN", "读取消息失败 %d", messageType)
				if d.reconnect(5) {
					continue
				} else {
					break
				}
			} else {
				if message != nil {
					err := proto.Unmarshal(message, pbPac)
					if err != nil {
						log.Println("解析消息失败：", err)
						continue
					}
					n := utils.HasGzipEncoding(pbPac.HeadersList)
					if n && pbPac.PayloadType == "msg" {
						uncompressedData, err := d.GzipUnzipReset(pbPac.Payload)
						if err != nil {
							log.Println("Gzip解压失败:", err)
							continue
						}

						err = proto.Unmarshal(uncompressedData, pbResp)
						if err != nil {
							log.Println("解析消息失败：", err)
							continue
						}
						if pbResp.NeedAck {
							pbAck.Reset()
							pbAck.LogId = pbPac.LogId
							pbAck.PayloadType = "ack"
							pbAck.Payload = []byte(pbResp.InternalExt)

							serializedAck, err := proto.Marshal(pbAck)
							if err != nil {
								log.Println("proto心跳包序列化失败:", err)
								continue
							}
							err = d.Conn.WriteMessage(websocket.BinaryMessage, serializedAck)
							if err != nil {
								log.Println("心跳包发送失败：", err)
								continue
							}
						}
						d.ProcessingMessage(pbResp)
					}
				}
			}
		}
	}
	return nil
}

// reconnect 尝试重新连接
func (d *DouyinLive) reconnect(i int) bool {
	var err error
	//log.Println("尝试重新连接...")
	//trace.Print("INFO", "尝试重新连接...")
	for attempt := 0; attempt < i; attempt++ {
		if d.Conn != nil {
			err := d.Conn.Close()
			if err != nil {
				//log.Printf("关闭连接失败: %v", err)
			}
		}
		d.Conn, _, err = websocket.DefaultDialer.Dial(d.wssurl, d.headers)
		if err != nil {
			//log.Printf("重连失败: %v", err)
			//trace.Print("INFO", "重连失败")
			//log.Printf("正在尝试第 %d 次重连...", attempt+1)
			//trace.Printf("INFO", "正在尝试第 %d 次重连...", attempt+1)
			time.Sleep(5 * time.Second)
		} else {
			//log.Println("重连成功")
			//trace.Print("INFO", "重连成功")
			return true
		}
	}
	//log.Println("重连失败，退出程序")
	//trace.Print("INFO", "重连失败，退出服务")
	return false
}

// StitchUrl 构建 WebSocket 连接的 URL
func (d *DouyinLive) StitchUrl() string {
	smap := utils.NewOrderedMap(d.roomid, d.pushid)
	signaturemd5 := utils.GetxMSStub(smap)
	signature := jsScript.ExecuteJS(signaturemd5)
	browserInfo := strings.Split(d.userAgent, "Mozilla")[1]
	parsedURL := strings.Replace(browserInfo[1:], " ", "%20", -1)
	fetchTime := time.Now().UnixNano() / int64(time.Millisecond)
	return "wss://webcast5-ws-web-lf.douyin.com/webcast/im/push/v2/?app_name=douyin_web&version_code=180800&" +
		"webcast_sdk_version=1.0.14-beta.0&update_version_code=1.0.14-beta.0&compress=gzip&device_platform" +
		"=web&cookie_enabled=true&screen_width=1920&screen_height=1080&browser_language=zh-CN&browser_platform=Win32&" +
		"browser_name=Mozilla&browser_version=" + parsedURL + "&browser_online=true" +
		"&tz_name=Asia/Shanghai&cursor=d-1_u-1_fh-7383731312643626035_t-1719159695790_r-1&internal_ext" +
		"=internal_src:dim|wss_push_room_id:" + d.roomid + "|wss_push_did:" + d.pushid + "|first_req_ms:" + cast.ToString(fetchTime) + "|fetch_time:" + cast.ToString(fetchTime) + "|seq:1|wss_info:0-" + cast.ToString(fetchTime) + "-0-0|" +
		"wrds_v:7382620942951772256&host=https://live.douyin.com&aid=6383&live_id=1&did_rule=3" +
		"&endpoint=live_pc&support_wrds=1&user_unique_id=" + d.pushid + "&im_path=/webcast/im/fetch/" +
		"&identity=audience&need_persist_msg_count=15&insert_task_id=&live_reason=&room_id=" + d.roomid + "&heartbeatDuration=0&signature=" + signature
}

// emit 触发事件处理器
func (d *DouyinLive) emit(eventData *douyin.Message) {
	for _, handler := range d.eventHandlers {
		handler(eventData)
	}
}

// ProcessingMessage 处理接收到的消息
func (d *DouyinLive) ProcessingMessage(response *douyin.Response) {
	for _, data := range response.MessagesList {
		d.emit(data)
		if data.Method == "WebcastControlMessage" {
			msg := &douyin.ControlMessage{}
			err := proto.Unmarshal(data.Payload, msg)
			if err != nil {
				//log.Println("解析protobuf失败", err)
				return
			}
			if msg.Status == 3 {
				d.isLiveClosed = false
				//log.Println("关闭ws链接成功")
			}
		}
	}
}

// Subscribe 订阅事件处理器
func (d *DouyinLive) Subscribe(handler EventHandler) {
	d.eventHandlers = append(d.eventHandlers, handler)
}

// Stop 停止 DouyinLive 并清理所有资源
func (d *DouyinLive) Stop() {
	// 等待 Start 完成初始化
	select {
	case <-d.startedCh:
		// Start 已完成初始化
	case <-time.After(5 * time.Second):
		// 超时处理，可能 Start 还未被调用
		return
	}

	// 发送停止信号
	close(d.stopCh)

	// 等待 Start 完全退出
	d.startWg.Wait()

	// 停止直播处理逻辑
	d.isLiveClosed = false

	// 关闭 WebSocket 连接
	if d.Conn != nil {
		err := d.Conn.Close()
		if err != nil {
			//log.Printf("关闭 WebSocket 连接失败: %v", err)
		} else {
			//log.Println("WebSocket 连接已关闭")
		}
		d.Conn = nil
	}

	// 关闭 Gzip 解压器
	if d.gzip != nil {
		err := d.gzip.Close()
		if err != nil {
			//log.Printf("关闭 Gzip 解压器失败: %v", err)
		} else {
			//log.Println("Gzip 解压器已关闭")
		}
		d.gzip = nil
	}

	// 清理缓冲区池
	if d.buffers != nil {
		d.buffers = nil
		//log.Println("缓冲区池已清理")
	}

	// 清理事件处理器
	if len(d.eventHandlers) > 0 {
		d.eventHandlers = nil
		//log.Println("事件处理器已清理")
	}

	// 清空 HTTP 求头
	if d.headers != nil {
		d.headers = nil
		//log.Println("HTTP 请求头已清理")
	}

	// 重置对象字段
	d.ttwid = ""
	d.roomid = ""
	d.liveid = ""
	d.liveurl = ""
	d.userAgent = ""
	d.wssurl = ""
	d.pushid = ""

	// 清理 HTTP 客户端
	if d.c != nil {
		d.c = nil
		//log.Println("HTTP 客户端已清理")
	}

	//log.Println("DouyinLive 对象已停止并销毁")
}
