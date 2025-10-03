package api

import (
	"UniBarrage/bilibili"
	"UniBarrage/douyin"
	"UniBarrage/douyu"
	"UniBarrage/huya"
	"UniBarrage/kuaishou"
	uni "UniBarrage/universal"
	log "UniBarrage/utils/trace"
	"UniBarrage/web"
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/goccy/go-json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func StartServer(host string, port int, certFile string, keyFile string, expectedToken string, allowedOrigins []string) {
	r := chi.NewRouter()

	// 中间件
	// r.Use(middleware.Logger)  // 删除这行来关闭 chi 的日志
	r.Use(middleware.Recoverer)

	// 配置 CORS 中间件，使用传入的 allowedOrigins 数组
	corsOptions := cors.Options{
		AllowedOrigins:   allowedOrigins, // 使用传入的数组
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // 缓存预检请求的结果的最大时间（秒）
	}
	r.Use(cors.Handler(corsOptions))

	// Dashboard 路由（不需要认证）
	r.Get("/", ServeDashboard)

	// API 路由（需要认证）
	r.Route("/api/v1", func(r chi.Router) {
		// 仅在指定了 token 时对 API 路由使用 AuthMiddleware
		if expectedToken != "" {
			r.Use(AuthMiddleware(expectedToken))
		}
		// 欢迎信息
		r.Get("/", Hello)
		// 获取所有服务状态
		r.Get("/all", ListAllServices)
		// 获取指定平台的所有服务
		r.Get("/{platform}", ListPlatformServices)
		// 获取单个服务状态
		r.Get("/{platform}/{roomId}", GetServiceDetail)
		// 启动服务
		r.Post("/{platform}", StartService)
		// 停止服务
		r.Delete("/{platform}/{roomId}", StopService)
	})

	addr := fmt.Sprintf("%s:%d", host, port)

	if certFile != "" && keyFile != "" {
		// 启动 HTTPS 服务
		log.Printf("INFO", "API: https://%s", addr)
		log.Printf("INFO", "Dashboard: https://%s", addr)
		if err := http.ListenAndServeTLS(addr, certFile, keyFile, r); err != nil {
			log.Printf("ERROR", "服务器启动失败: %v", err)
		}
	} else {
		// 启动 HTTP 服务
		log.Printf("INFO", "API: http://%s", addr)
		log.Printf("INFO", "Dashboard: http://%s", addr)
		if err := http.ListenAndServe(addr, r); err != nil {
			log.Printf("ERROR", "服务器启动失败: %v", err)
		}
	}
}

// AuthMiddleware 用于验证 Bearer Token 的中间件
func AuthMiddleware(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 获取 Authorization 头
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				jsonError(w, http.StatusUnauthorized, "未提供 Bearer Token")
				return
			}

			// 提取 Token 并验证
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != expectedToken {
				jsonError(w, http.StatusUnauthorized, "无效的 Token")
				return
			}

			// 验证通过，继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

type ServiceStatus struct {
	Platform string        `json:"platform"`
	RoomID   string        `json:"rid"`
	StopChan chan struct{} `json:"-"`
}

// ServiceManager 服务管理器
type ServiceManager struct {
	rwMutex  sync.RWMutex
	services map[string]*ServiceStatus
}

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[string]*ServiceStatus),
	}
}

// AddService 添加服务
func (sm *ServiceManager) AddService(key string, status *ServiceStatus) error {
	sm.rwMutex.Lock()
	defer sm.rwMutex.Unlock()

	if _, exists := sm.services[key]; exists {
		return fmt.Errorf("服务已存在")
	}
	sm.services[key] = status
	return nil
}

// RemoveService 删除服务
func (sm *ServiceManager) RemoveService(key string) {
	sm.rwMutex.Lock()
	defer sm.rwMutex.Unlock()
	delete(sm.services, key)
}

// GetService 获取服务
func (sm *ServiceManager) GetService(key string) (*ServiceStatus, bool) {
	sm.rwMutex.RLock()
	defer sm.rwMutex.RUnlock()
	status, exists := sm.services[key]
	return status, exists
}

// GetAllServices 获取所有服务
func (sm *ServiceManager) GetAllServices() []*ServiceStatus {
	sm.rwMutex.RLock()
	defer sm.rwMutex.RUnlock()

	services := make([]*ServiceStatus, 0, len(sm.services))
	for _, status := range sm.services {
		services = append(services, status)
	}
	return services
}

var serviceMap = NewServiceManager()

// 生成服务唯一标识
func generateServiceKey(platform, roomID string) string {
	return fmt.Sprintf("%s_%s", platform, roomID)
}

// DouYinRoom int 抖音房间号
func startDouyinService(DouYinRoom int, stopChan chan struct{}) error {
	serviceKey := generateServiceKey("douyin", strconv.Itoa(DouYinRoom))

	status := &ServiceStatus{
		Platform: "douyin",
		RoomID:   strconv.Itoa(DouYinRoom),
		StopChan: stopChan,
	}

	if err := serviceMap.AddService(serviceKey, status); err != nil {
		return fmt.Errorf("抖音房间 %d 已在监听中", DouYinRoom)
	}

	go func() {
		go douyin.StartListen(DouYinRoom, stopChan)
		<-stopChan
		serviceMap.RemoveService(serviceKey)
	}()

	log.Printf("DOUYIN", "提交 DouYin 监听服务 (%d)", DouYinRoom)
	return nil
}

// BiliBiliRoom int 哔哩哔哩房间号, cookie string 登录cookie (可选)
func startBilibiliService(BiliBiliRoom int, cookie string, stopChan chan struct{}) error {
	serviceKey := generateServiceKey("bilibili", strconv.Itoa(BiliBiliRoom))

	status := &ServiceStatus{
		Platform: "bilibili",
		RoomID:   strconv.Itoa(BiliBiliRoom),
		StopChan: stopChan,
	}

	if err := serviceMap.AddService(serviceKey, status); err != nil {
		return fmt.Errorf("哔哩哔哩房间 %d 已在监听中", BiliBiliRoom)
	}

	go func() {
		go bilibili.StartListen(BiliBiliRoom, cookie, stopChan)
		<-stopChan
		serviceMap.RemoveService(serviceKey)
	}()

	log.Printf("BILIBILI", "提交 BiliBili 监听服务 (%d)", BiliBiliRoom)
	return nil
}

// KuaiShouRoomLink string 快手房间链接, cookie string 登录cookie (可选)
func startKuaishouService(KuaiShouRoomLink string, cookie string, stopChan chan struct{}) error {
	serviceKey := generateServiceKey("kuaishou", KuaiShouRoomLink)

	status := &ServiceStatus{
		Platform: "kuaishou",
		RoomID:   KuaiShouRoomLink,
		StopChan: stopChan,
	}

	if err := serviceMap.AddService(serviceKey, status); err != nil {
		return fmt.Errorf("快手房间 %s 已在监听中", KuaiShouRoomLink)
	}

	go func() {
		go kuaishou.StartListen(KuaiShouRoomLink, cookie, stopChan)
		<-stopChan
		serviceMap.RemoveService(serviceKey)
	}()

	log.Printf("KUAISHOU", "提交 KuaiShou 监听服务 (%s)", KuaiShouRoomLink[strings.LastIndex(KuaiShouRoomLink, "/"):])
	return nil
}

// DouYuRoom int 斗鱼房间号
func startDouYuService(DouYuRoom int, stopChan chan struct{}) error {
	serviceKey := generateServiceKey("douyu", strconv.Itoa(DouYuRoom))

	status := &ServiceStatus{
		Platform: "douyu",
		RoomID:   strconv.Itoa(DouYuRoom),
		StopChan: stopChan,
	}

	if err := serviceMap.AddService(serviceKey, status); err != nil {
		return fmt.Errorf("斗鱼房间 %d 已在监听中", DouYuRoom)
	}

	go func() {
		go douyu.StartListen(DouYuRoom, stopChan)
		<-stopChan
		serviceMap.RemoveService(serviceKey)
	}()

	log.Printf("DOUYU", "提交 DouYu 监听服务 (%d)", DouYuRoom)
	return nil
}

// HuYaRoom string 虎牙房间号
func startHuYaService(HuYaRoom string, stopChan chan struct{}) error {
	serviceKey := generateServiceKey("huya", HuYaRoom)

	status := &ServiceStatus{
		Platform: "huya",
		RoomID:   HuYaRoom,
		StopChan: stopChan,
	}

	if err := serviceMap.AddService(serviceKey, status); err != nil {
		return fmt.Errorf("虎牙房间 %s 已在监听中", HuYaRoom)
	}

	go func() {
		go huya.StartListen(HuYaRoom, stopChan)
		<-stopChan
		serviceMap.RemoveService(serviceKey)
	}()

	log.Printf("HUYA", "提交 HuYa 监听服务 (%s)", HuYaRoom)
	return nil
}

// StartService HTTP 处理函数
func StartService(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")

	var req struct {
		RoomID string `json:"rid"`
		Cookie string `json:"cookie,omitempty"`
	}

	defer r.Body.Close() // 确保请求体关闭，避免资源泄露

	// 解码请求体
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证 rid 是否为空
	if req.RoomID == "" {
		jsonError(w, http.StatusBadRequest, "房间 ID 不能为空")
		return
	}

	// 使用上下文设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopChan := make(chan struct{})
	var startErr error

	// 根据平台启动服务
	go func() {
		switch uni.Platform(platform) {
		case uni.DouYin:
			// 需要将 RoomID 转为整数
			roomID, err := strconv.Atoi(req.RoomID)
			if err != nil {
				startErr = fmt.Errorf("房间 ID 格式错误，必须为整数")
				cancel()
				return
			}
			startErr = startDouyinService(roomID, stopChan)

		case uni.BiliBili:
			// 需要将 RoomID 转为整数
			roomID, err := strconv.Atoi(req.RoomID)
			if err != nil {
				startErr = fmt.Errorf("房间 ID 格式错误，必须为整数")
				cancel()
				return
			}
			startErr = startBilibiliService(roomID, req.Cookie, stopChan)

		case uni.KuaiShou:
			// KuaiShou 使用字符串 RoomID
			startErr = startKuaishouService(req.RoomID, req.Cookie, stopChan)

		case uni.DouYu:
			// 需要将 RoomID 转为整数
			roomID, err := strconv.Atoi(req.RoomID)
			if err != nil {
				startErr = fmt.Errorf("房间 ID 格式错误，必须为整数")
				cancel()
				return
			}
			startErr = startDouYuService(roomID, stopChan)

		case uni.HuYa:
			// HuYa 使用字符串 RoomID
			startErr = startHuYaService(req.RoomID, stopChan)

		default:
			// 不支持的平台
			startErr = fmt.Errorf("不支持的平台")
		}
		cancel() // 服务启动完成后，取消上下文
	}()

	// 等待启动完成或超时
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			jsonError(w, http.StatusGatewayTimeout, "服务启动超时")
			return
		}
	}

	// 检查服务启动中的错误
	if startErr != nil {
		jsonError(w, http.StatusBadRequest, startErr.Error())
		return
	}

	// 服务启动成功的响应
	jsonResponse(w, http.StatusCreated, "服务启动成功", map[string]string{
		"platform": platform,
		"rid":      req.RoomID,
	})
}

func StopService(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")
	roomID := chi.URLParam(r, "roomId")

	serviceKey := generateServiceKey(platform, roomID)
	if status, exists := serviceMap.GetService(serviceKey); exists {
		close(status.StopChan)
		//serviceMap.RemoveService(serviceKey) // 直接移除服务
		jsonResponse(w, http.StatusOK, "服务已停止", map[string]string{
			"platform": platform,
			"rid":      roomID,
		})
		log.Printf(platform, "已停止 (%s) 的监听服务", roomID)
		return
	}

	jsonError(w, http.StatusNotFound, "服务未找到")
}

func Hello(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, "Hello, UniBarrage!", nil)
}

func ListAllServices(w http.ResponseWriter, r *http.Request) {
	services := serviceMap.GetAllServices()
	jsonResponse(w, http.StatusOK, "获取成功", services)
}

func ListPlatformServices(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")

	allServices := serviceMap.GetAllServices()
	platformServices := make([]*ServiceStatus, 0)

	for _, service := range allServices {
		if service.Platform == platform {
			platformServices = append(platformServices, service)
		}
	}

	jsonResponse(w, http.StatusOK, "获取成功", platformServices)
}

func GetServiceDetail(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")
	roomID := chi.URLParam(r, "roomId")

	serviceKey := generateServiceKey(platform, roomID)
	if status, exists := serviceMap.GetService(serviceKey); exists {
		jsonResponse(w, http.StatusOK, "获取成功", status)
		return
	}

	jsonError(w, http.StatusNotFound, "服务未找到")
}

// Response 统一的API响应格式
type Response struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 响应信息
	Data    interface{} `json:"data"`    // 响应数据
}

// 响应处理函数
func jsonResponse(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// 错误响应处理函数
func jsonError(w http.ResponseWriter, code int, message string) {
	jsonResponse(w, code, message, nil)
}

// ServeDashboard 提供嵌入的 dashboard.html
func ServeDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(web.DashboardHTML)
}
