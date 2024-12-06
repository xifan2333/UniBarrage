package huya

import (
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

// ChatMessage 代表一条弹幕消息
type ChatMessage struct {
	Type string `json:"type"`
	Time int64  `json:"time"`
	From struct {
		Name string `json:"name"`
		Rid  string `json:"rid"`
	} `json:"from"`
	ID      string `json:"id"`
	Content string `json:"content"`
}

// GiftMessage 代表一条礼物消息
type GiftMessage struct {
	Type string `json:"type"`
	Time int64  `json:"time"`
	Name string `json:"name"`
	From struct {
		Name string `json:"name"`
		Rid  string `json:"rid"`
	} `json:"from"`
	ID    string `json:"id"`
	Count int    `json:"count"`
	Price int    `json:"price"`
	Earn  int    `json:"earn"`
}

//go:embed client/*
var clientFiles embed.FS

// StartListen 启动监听指定房间的弹幕和礼物消息
func StartListen(roomId string, stopChan chan struct{}) {
	id := roomId

	// 创建 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// readWebSocketData 从指定端口读取 WebSocket 数据并解析
	readWebSocketData := func(port int) {
		u := url.URL{Scheme: "ws", Host: "127.0.0.1:" + strconv.Itoa(port), Path: "/"}
		var conn *websocket.Conn
		var err error

		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				//log.Print("ERROR", "WebSocket connection timed out")
				log.Print("ERROR", "虎牙直播监听启动失败")
				close(stopChan)
				return
			case <-ticker.C:
				conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
				if err == nil {
					//log.Print("INFO", "WebSocket connection established")
					log.Print("HUYA", "已启动虎牙直播监听")
					break
				} else {
					log.Print("WARN", "WebSocket connection attempt failed, retrying...")
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
			if err := json.Unmarshal(message, &chatMsg); err == nil && chatMsg.Type == "chat" {
				avatar, _ := GetAvatarByUID(chatMsg.From.Rid)
				data, _ := uni.CreateUniMessage(
					id,
					uni.HuYa,
					uni.ChatMessageType,
					&uni.ChatMessage{
						Name:     chatMsg.From.Name,
						Avatar:   avatar,
						Content:  chatMsg.Content,
						Emoticon: []string{},
						Raw:      chatMsg,
					},
				)
				ws.BroadcastToClients(data)
				continue
			}

			var giftMsg GiftMessage
			if err := json.Unmarshal(message, &giftMsg); err == nil && giftMsg.Type == "gift" {
				avatar, _ := GetAvatarByUID(chatMsg.From.Rid)
				data, _ := uni.CreateUniMessage(
					id,
					uni.HuYa,
					uni.GiftMessageType,
					&uni.GiftMessage{
						Name:     giftMsg.From.Name,
						Avatar:   avatar,
						Item:     giftMsg.Name,
						Num:      giftMsg.Count,
						Price:    float64(giftMsg.Earn) / 100,
						GiftIcon: "",
						Raw:      giftMsg,
					},
				)
				ws.BroadcastToClients(data)
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
		cmd = exec.CommandContext(ctx, "cmd", "/C", nodePath, indexJsPath, roomId, strconv.Itoa(port))
	} else {
		cmd = exec.CommandContext(ctx, nodePath, indexJsPath, roomId, strconv.Itoa(port))
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		log.Print("ERROR", "Error starting command")
		return
	}

	// 启动 WebSocket 数据读取
	go readWebSocketData(port)

	// 监听 stopChan
	go func() {
		<-stopChan
		cancel()
	}()

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		return
		//log.Printf("ERROR", "Error waiting for command: %v", err)
	}
}
