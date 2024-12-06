package proxy

import (
	log "UniBarrage/utils/trace"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	lru "github.com/hashicorp/golang-lru/v2"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ImageCacheItem 表示一个缓存项
type ImageCacheItem struct {
	Data        []byte
	ContentType string
	Timestamp   time.Time
}

var (
	host       string                              // 运行主机
	port       int                                 // 运行端口
	cache      *lru.Cache[string, *ImageCacheItem] // 内存缓存
	useProxy   = false                             // 是否使用代理
	useHttps   = false                             // 是否使用 Https
	badgerDB   *badger.DB                          // BadgerDB 实例
	cacheLimit = 1000                              // LRU 缓存大小限制
)

// 初始化缓存和 BadgerDB
func initCache() {
	var err error

	// 初始化内存缓存
	cache, err = lru.New[string, *ImageCacheItem](cacheLimit)
	if err != nil {
		log.Printf("WARN", "创建 LRU 缓存失败: %v", err)
		return
	}

	// 初始化 BadgerDB
	tempDir := os.TempDir()
	dbPath := fmt.Sprintf("%s/CacheBadger", tempDir)
	opts := badger.DefaultOptions(dbPath).WithLoggingLevel(badger.WARNING)

	badgerDB, err = badger.Open(opts)
	if err != nil {
		log.Printf("ERROR", "连接 BadgerDB 数据库失败: %v", err)
		return
	}
}

// 从 BadgerDB 获取缓存项
func getFromBadger(url string) (*ImageCacheItem, bool) {
	var item ImageCacheItem
	err := badgerDB.View(func(txn *badger.Txn) error {
		entry, err := txn.Get([]byte(url))
		if err != nil {
			return err
		}
		item.Data, err = entry.ValueCopy(nil)
		if err != nil {
			return err
		}
		contentType, err := entry.ValueCopy(nil)
		if err != nil {
			return err
		}
		item.ContentType = string(contentType)
		return nil
	})
	if err != nil {
		return nil, false
	}
	return &item, true
}

// 将缓存项保存到 BadgerDB
func saveToBadger(url string, item *ImageCacheItem) {
	err := badgerDB.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(url), item.Data).WithTTL(time.Hour * 24)
		return txn.SetEntry(e)
	})
	if err != nil {
		log.Printf("ERROR", "保存缓存到 BadgerDB 失败: %v", err)
	}
}

// 从缓存中获取图片数据
func getFromCache(url string) ([]byte, string, bool) {
	if item, found := cache.Get(url); found {
		return item.Data, item.ContentType, true
	}
	if item, found := getFromBadger(url); found {
		cache.Add(url, item) // 加载到内存缓存
		return item.Data, item.ContentType, true
	}
	return nil, "", false
}

// 将图片数据保存到缓存
func saveToCache(url string, data []byte, contentType string) {
	item := &ImageCacheItem{
		Data:        data,
		ContentType: contentType,
		Timestamp:   time.Now(),
	}
	cache.Add(url, item)
	saveToBadger(url, item) // 同时保存到 BadgerDB
}

// 下载图片
func downloadImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("下载图片失败: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image") {
		return nil, "", fmt.Errorf("无效的 Content-Type: %s", contentType)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取图片数据失败: %w", err)
	}

	return data, contentType, nil
}

// 设置 CORS 头部
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// 处理图片代理请求
func serveImage(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		http.Error(w, "缺少 'url' 参数", http.StatusBadRequest)
		return
	}

	if data, contentType, found := getFromCache(imageURL); found {
		w.Header().Set("Content-Type", contentType)
		w.Write(data)
		return
	}

	data, contentType, err := downloadImage(imageURL)
	if err != nil {
		log.Printf("WARN", "下载图片出错: %v", err)
		http.Error(w, "图片下载失败", http.StatusInternalServerError)
		return
	}

	saveToCache(imageURL, data, contentType)

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

// StartServer 启动图片代理服务器，自动判断是否使用 HTTPS
func StartServer(_host string, _port int, certFile string, keyFile string, allowedOrigins []string) {
	host = _host
	port = _port

	// 初始化缓存逻辑
	initCache()

	mux := http.NewServeMux()
	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头
		origin := r.Header.Get("Origin")
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		serveImage(w, r)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: mux,
	}

	useProxy = true

	// 判断是否启用 HTTPS
	if certFile != "" && keyFile != "" {
		useHttps = true
		log.Printf("INFO", "启动 本地图片代理 (%s:%d/image)", host, port)
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Printf("ERROR", "启动服务器失败: %v", err)
		}
	} else {
		log.Printf("INFO", "启动 本地图片代理 (%s:%d/image)", host, port)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("ERROR", "启动服务器失败: %v", err)
		}
	}
}

// 检查请求来源是否在允许的来源列表中
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		return true // 如果允许所有来源
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// GenerateImageURL 转换原始图片 URL 为代理 URL
func GenerateImageURL(originalURL string) (string, error) {
	if useProxy {
		// 对 URL 进行编码处理
		encodedURL := url.QueryEscape(originalURL)
		// 拼接代理 URL
		protocol := "http"
		if useHttps {
			protocol = "https"
		}
		proxyURL := fmt.Sprintf("%s://%s:%d/image?url=%s", protocol, host, port, encodedURL)
		return proxyURL, nil
	} else {
		return originalURL, nil
	}
}
