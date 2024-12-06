package douyu

import (
	"UniBarrage/douyu/gifts"
	ws "UniBarrage/services/websockets"
	uni "UniBarrage/universal"
	"UniBarrage/utils/node"
	"UniBarrage/utils/ports"
	log "UniBarrage/utils/trace"
	"context"
	"embed"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// ChatMessage 表示弹幕消息的结构体
type ChatMessage struct {
	Type  string `json:"type"`           // 消息类型，固定为 chatmsg
	Rid   string `json:"rid"`            // 房间 id
	Ct    string `json:"ct,omitempty"`   // 客户端类型
	Uid   string `json:"uid"`            // 发送者 uid
	Nn    string `json:"nn"`             // 发送者昵称
	Txt   string `json:"txt"`            // 弹幕文本内容
	Cid   string `json:"cid"`            // 弹幕唯一 ID
	Ic    string `json:"ic,omitempty"`   // 用户头像
	Level int    `json:"level,string"`   // 用户等级（注意字符串类型转换）
	Sahf  string `json:"sahf,omitempty"` // 未知字段
	Cst   string `json:"cst,omitempty"`  // 时间戳
	Bnn   string `json:"bnn,omitempty"`  // 未知字段
	Bl    string `json:"bl,omitempty"`   // 未知字段
	Brid  string `json:"brid,omitempty"` // 未知字段
	Hc    string `json:"hc,omitempty"`   // 未知字段
	Lk    string `json:"lk,omitempty"`   // 未知字段
	Pdg   string `json:"pdg,omitempty"`  // 未知字段
	Pdk   string `json:"pdk,omitempty"`  // 未知字段
	Ext   string `json:"ext,omitempty"`  // 未知字段
	Nl    string `json:"nl,omitempty"`   // 未知字段
	Dms   string `json:"dms,omitempty"`  // 未知字段
	Ail   string `json:"ail,omitempty"`  // 未知字段
	Ufs   string `json:"ufs,omitempty"`  // 未知字段
	If    string `json:"if,omitempty"`   // 未知字段
	Gid   string `json:"gid,omitempty"`  // 弹幕组 id
	Gt    int    `json:"gt,omitempty"`   // 礼物头衔，默认值 0
	Col   int    `json:"col,omitempty"`  // 颜色，默认值 0
	Rg    int    `json:"rg,omitempty"`   // 房间权限组，默认值 1
	Pg    int    `json:"pg,omitempty"`   // 平台权限组，默认值 1
	Dlv   int    `json:"dlv,omitempty"`  // 酬勤等级，默认值 0
	Dc    int    `json:"dc,omitempty"`   // 酬勤数量，默认值 0
	Bdlv  int    `json:"bdlv,omitempty"` // 最高酬勤等级，默认值 0
}

// GiftMessage 表示赠送礼物消息的结构体
type GiftMessage struct {
	Type  string `json:"type"`  // 消息类型，固定为 dgb
	Rid   string `json:"rid"`   // 房间 ID
	Gid   string `json:"gid"`   // 弹幕分组 ID
	Gfid  string `json:"gfid"`  // 礼物 id
	Gs    string `json:"gs"`    // 礼物显示样式
	Uid   string `json:"uid"`   // 用户 id
	Nn    string `json:"nn"`    // 用户昵称
	Str   string `json:"str"`   // 用户战斗力
	Level int    `json:"level"` // 用户等级
	Dw    int    `json:"dw"`    // 主播体重
	Gfcnt int    `json:"gfcnt"` // 礼物个数，默认值 1
	Hits  int    `json:"hits"`  // 礼物连击次数，默认值 1
	Dlv   int    `json:"dlv"`   // 酬勤头衔，默认值 0
	Dc    int    `json:"dc"`    // 酬勤数量，默认值 0
	Bdl   int    `json:"bdl"`   // 全站最高酬勤等级，默认值 0
	Rg    int    `json:"rg"`    // 房间权限组，默认值 1
	Pg    int    `json:"pg"`    // 平台权限组，默认值 1
	Rpid  int    `json:"rpid"`  // 红包 id，默认值 0
	Slt   int    `json:"slt"`   // 红包开启剩余时间，默认值 0
	Elt   int    `json:"elt"`   // 红包销毁剩余时间，默认值 0
}

// UserEnterMessage 表示用户进入房间的消息结构体
type UserEnterMessage struct {
	Type  string `json:"type"`  // 消息类型，固定为 "uenter"
	Rid   string `json:"rid"`   // 房间 ID
	Gid   string `json:"gid"`   // 弹幕分组 ID
	Uid   string `json:"uid"`   // 用户 ID
	Nn    string `json:"nn"`    // 用户昵称
	Str   string `json:"str"`   // 用户战斗力
	Level int    `json:"level"` // 新用户等级
	Gt    int    `json:"gt"`    // 礼物头衔，默认值 0（表示没有头衔）
	Rg    int    `json:"rg"`    // 房间权限组，默认值 1（表示普通权限用户）
	Pg    int    `json:"pg"`    // 平台身份组，默认值 1（表示普通权限用户）
	Dlv   int    `json:"dlv"`   // 酬勤等级，默认值 0（表示没有酬勤）
	Dc    int    `json:"dc"`    // 酬勤数量，默认值 0（表示没有酬勤数量）
	Bdlv  int    `json:"bdlv"`  // 最高酬勤等级，默认值 0（表示全站都没有酬勤）
}

// RoomStreamStartMessage 表示房间开播提醒的消息结构体
type RoomStreamStartMessage struct {
	Type    string `json:"type"`    // 消息类型，固定为 "rss"
	Rid     string `json:"rid"`     // 房间 ID
	Gid     string `json:"gid"`     // 弹幕分组 ID
	Ss      int    `json:"ss"`      // 直播状态，1-正在直播, 2-没有直播
	Code    int    `json:"code"`    // 类型
	Rt      int    `json:"rt"`      // 开关播原因: 0-主播开关播, 其他值-其他原因
	Notify  string `json:"notify"`  // 通知类型
	Endtime int64  `json:"endtime"` // 关播时间（仅关播时有效）
}

//go:embed client/*
var clientFiles embed.FS

// StartListen 启动监听指定房间的弹幕和礼物消息
func StartListen(roomId int, stopChan chan struct{}) {
	id := strconv.Itoa(roomId)

	// 创建 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// readWebSocketData 从指定端口读取 WebSocket 数据并解析
	readWebSocketData := func(port int) {
		u := url.URL{Scheme: "ws", Host: "127.0.0.1:" + strconv.Itoa(port), Path: "/"}
		var conn *websocket.Conn
		var err error

		// 设置超时时间，例如 30 秒
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				//log.Print("ERROR", "WebSocket connection timed out")
				close(stopChan)
				log.Print("ERROR", "斗鱼直播监听启动失败")
				return
			case <-ticker.C:
				// 尝试连接 WebSocket
				conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
				if err == nil {
					log.Print("DOUYU", "已启动斗鱼直播监听")
					break
				} else {
					//log.Print("WARN", "WebSocket connection attempt failed, retrying...")
				}
			}

			if conn != nil {
				break
			}
		}

		if conn == nil {
			log.Print("ERROR", "WebSocket connection could not be established within the timeout period")
			return
		}
		defer conn.Close()

		// 添加 context 控制的连接关闭
		go func() {
			<-ctx.Done()
			if conn != nil {
				conn.Close()
			}
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				//log.Print("ERROR", "WebSocket read error:")
				break
			}

			// 解析消息
			var chatMsg ChatMessage
			if err := json.Unmarshal(message, &chatMsg); err == nil && chatMsg.Type == "chatmsg" {
				data, _ := uni.CreateUniMessage(
					id,
					uni.DouYu,
					uni.ChatMessageType,
					&uni.ChatMessage{
						Name:     chatMsg.Nn,
						Avatar:   BuildAvatarURL(chatMsg.Ic),
						Content:  chatMsg.Txt,
						Emoticon: []string{},
						Raw:      chatMsg,
					},
				)
				ws.BroadcastToClients(data)
				continue
			}

			var giftMsg GiftMessage
			if err := json.Unmarshal(message, &giftMsg); err == nil && giftMsg.Type == "dgb" {
				gift, _ := gifts.GetGiftByID(giftMsg.Gfid)
				data, _ := uni.CreateUniMessage(
					id,
					uni.DouYu,
					uni.GiftMessageType,
					&uni.GiftMessage{
						Name:     giftMsg.Nn,
						Avatar:   BuildAvatarURL(chatMsg.Ic),
						Item:     gift.Name,
						Num:      giftMsg.Gfcnt,
						Price:    float64(giftMsg.Dc),
						GiftIcon: gift.ImageURL,
						Raw:      giftMsg,
					},
				)
				ws.BroadcastToClients(data)
				continue
			}

			var enterMsg UserEnterMessage
			if err := json.Unmarshal(message, &enterMsg); err == nil && enterMsg.Type == "uenter" {
				data, _ := uni.CreateUniMessage(
					id,
					uni.DouYu,
					uni.EnterRoomMessageType,
					&uni.EnterRoomMessage{
						Name:   enterMsg.Nn,
						Avatar: "",
						Raw:    enterMsg,
					},
				)
				ws.BroadcastToClients(data)
				continue
			}

			var roomMsg RoomStreamStartMessage
			if err := json.Unmarshal(message, &roomMsg); err == nil && roomMsg.Type == "rss" {
				if roomMsg.Ss == 2 {
					data, _ := uni.CreateUniMessage(
						id,
						uni.DouYu,
						uni.EndLiveMessageType,
						&uni.EndLiveMessage{
							Raw: roomMsg,
						},
					)
					ws.BroadcastToClients(data)
				}
				continue
			}
		}
	}

	// 获取 Node.js 可执行文件路径
	nodePath := node.EnsureNodeInstalled(os.TempDir())
	if nodePath == "" {
		log.Print("ERROR", "Node.js not found or failed to install")
		return
	}

	// 监听 stopChan
	go func() {
		<-stopChan
		cancel()
	}()

	// 提取 client 目录到临时目录
	tmpDir, err := os.MkdirTemp("", "client-*")
	if err != nil {
		log.Print("ERROR", "Error creating temp directory")
		return
	}
	defer os.RemoveAll(tmpDir) // 在执行结束后删除临时目录

	err = fs.WalkDir(clientFiles, "client", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("client", path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(tmpDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		} else {
			fileData, err := clientFiles.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(targetPath, fileData, 0644)
		}
	})

	if err != nil {
		log.Print("ERROR", "Error extracting client files")
		return
	}

	indexJsPath := filepath.Join(tmpDir, "index.js")
	port, _ := ports.GetAvailablePort() // 获取一个系统可用的端口 (port: int)

	// 创建命令
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", nodePath, indexJsPath, strconv.Itoa(roomId), strconv.Itoa(port))
	} else {
		cmd = exec.CommandContext(ctx, nodePath, indexJsPath, strconv.Itoa(roomId), strconv.Itoa(port))
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		log.Print("ERROR", "Error starting command")
		return
	}

	// 启动 WebSocket 数据读取
	go readWebSocketData(port)

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		return
		//log.Printf("ERROR", "Error waiting for command: %v", err)
	}
}
