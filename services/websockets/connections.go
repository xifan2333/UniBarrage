package websockets

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/goccy/go-json"
	"github.com/smallnest/chanx"

	uni "UniBarrage/universal"
	"UniBarrage/utils/ports"
	log "UniBarrage/utils/trace"
)

// Connection 包装 WebSocket 连接和写入通道
type Connection struct {
	Conn     net.Conn
	writeCh  *chanx.UnboundedChan[[]byte]
	platform uni.Platform // 连接时的过滤条件：平台
	id       string       // 连接时的过滤条件：ID
}

// 使用 map 搭配 sync.RWMutex 储存客户端连接
var (
	agentList = make(map[string]*Connection)
	mu        sync.RWMutex
)

// StartServer 启动 WebSocket 服务端，根据是否提供证书决定是启动 ws 还是 wss
func StartServer(host string, port int, certFile string, keyFile string, allowedOrigins []string) {
	_ = ports.FreePort(port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头
		origin := r.Header.Get("Origin")
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		serveWs(w, r)
	})

	if isPortAvailable(host, port) {
		go func() {
			if certFile != "" && keyFile != "" {
				log.Printf("INFO", "WebSocket (wss://%s:%d)", host, port)
				if err := http.ListenAndServeTLS(host+":"+strconv.Itoa(port), certFile, keyFile, nil); err != nil {
					log.Printf("ERROR", "服务器启动失败: %v", err)
				}
			} else {
				log.Printf("INFO", "WebSocket (ws://%s:%d)", host, port)
				if err := http.ListenAndServe(host+":"+strconv.Itoa(port), nil); err != nil {
					log.Printf("ERROR", "服务器启动失败: %v", err)
				}
			}
		}()
	}
}

// 检查请求来源是否在允许的来源列表中
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		return true
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// 检查本地端口是否可用
func isPortAvailable(host string, port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return true
	}
	_ = conn.Close()
	return false
}

// serveWs 处理 WebSocket 请求
func serveWs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	var platform uni.Platform
	var id string

	if len(parts) > 0 && parts[0] != "" {
		platform = uni.Platform(parts[0])
	}
	if len(parts) > 1 && parts[1] != "" {
		id = parts[1]
	}

	if platform != "" && !uni.IsValidPlatform(platform) {
		log.Printf("WARN", "无效的平台: %s", platform)
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}

	log.Printf("INFO", "%s 建立连接 (Total:%d)", r.RemoteAddr, getConnectionCount()+1)

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Print("ERROR", "WebSocket 升级失败")
		return
	}
	defer conn.Close()

	sec := r.Header.Get("Sec-WebSocket-Key")
	connection := newConnection(conn, platform, id)
	storeConnection(sec, connection)
	defer deleteConnection(sec)

	go connection.startWriter()

	for {
		msg, _, err := wsutil.ReadClientData(conn)
		if err != nil {
			log.Printf("WARN", "%s 断开连接 (Total:%d)", r.RemoteAddr, getConnectionCount()-1)
			break
		}
		log.Printf("INFO", "%s", msg)

		if string(msg) == "ping" {
			connection.writeMessage([]byte("pong"))
		}
	}
}

// 创建新连接时初始化通道和写入 goroutine
func newConnection(conn net.Conn, platform uni.Platform, id string) *Connection {
	c := &Connection{
		Conn:     conn,
		writeCh:  chanx.NewUnboundedChan[[]byte](context.Background(), 256),
		platform: platform,
		id:       id,
	}
	return c
}

// 启动用于发送消息的 goroutine
func (c *Connection) startWriter() {
	for msg := range c.writeCh.Out {
		if err := wsutil.WriteServerMessage(c.Conn, ws.OpText, msg); err != nil {
			log.Printf("WARN", "发送消息失败: %v", err)
			break
		}
	}
}

// 写入数据到通道，而不是直接写入连接
func (c *Connection) writeMessage(message []byte) {
	c.writeCh.In <- message
}

// 储存 WebSocket 客户端连接
func storeConnection(agentID string, conn *Connection) {
	mu.Lock()
	defer mu.Unlock()
	agentList[agentID] = conn
}

// 获取 WebSocket 客户端连接
func getConnection(agentID string) (*Connection, bool) {
	mu.RLock()
	defer mu.RUnlock()
	conn, ok := agentList[agentID]
	return conn, ok
}

// 删除 WebSocket 客户端连接
func deleteConnection(agentID string) {
	mu.Lock()
	defer mu.Unlock()
	delete(agentList, agentID)
}

// 获取当前连接数
func getConnectionCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(agentList)
}

// BroadcastToClients 广播消息到所有客户端
func BroadcastToClients(message *uni.UniMessage) {
	mu.RLock()
	connections := make([]*Connection, len(agentList))
	i := 0
	for _, conn := range agentList {
		connections[i] = conn
		i++
	}
	mu.RUnlock()

	for _, conn := range connections {
		go func(c *Connection) {
			if shouldSendMessage(c, message) {
				msgToSend, err := formatMessage(message)
				if err != nil {
					log.Printf("WARN", "消息格式化失败: %v", err)
					return
				}

				c.writeMessage([]byte(msgToSend))
			}
		}(conn)
	}
}

// 判断是否应发送消息给连接
func shouldSendMessage(conn *Connection, msg *uni.UniMessage) bool {
	if conn.platform != "" && conn.platform != msg.Platform {
		return false
	}
	if conn.id != "" && conn.id != msg.RID {
		return false
	}
	return true
}

// 格式化消息为字符串或 JSON
func formatMessage(message interface{}) (string, error) {
	switch v := message.(type) {
	case string:
		return v, nil
	default:
		jsonData, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	}
}
