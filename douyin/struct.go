package douyin

import (
	"UniBarrage/douyin/generated/douyin"
	"compress/gzip"
	"github.com/gorilla/websocket"
	"github.com/imroc/req/v3"
	"net/http"
	"sync"
)

const (
	WebcastChatMessage        = "WebcastChatMessage"
	WebcastGiftMessage        = "WebcastGiftMessage"
	WebcastLikeMessage        = "WebcastLikeMessage"
	WebcastMemberMessage      = "WebcastMemberMessage"
	WebcastSocialMessage      = "WebcastSocialMessage"
	WebcastRoomUserSeqMessage = "WebcastRoomUserSeqMessage"
	WebcastFansclubMessage    = "WebcastFansclubMessage"
	WebcastControlMessage     = "WebcastControlMessage"
	WebcastEmojiChatMessage   = "WebcastEmojiChatMessage"
	WebcastRoomStatsMessage   = "WebcastRoomStatsMessage"
	WebcastRoomMessage        = "WebcastRoomMessage"
	WebcastRoomRankMessage    = "WebcastRoomRankMessage"

	Default = "Default"
)

type EventHandler func(eventData *douyin.Message)
type DouyinLive struct {
	ttwid         string
	roomid        string
	liveid        string
	liveurl       string
	userAgent     string
	c             *req.Client
	eventHandlers []EventHandler
	headers       http.Header
	buffers       *sync.Pool
	gzip          *gzip.Reader
	Conn          *websocket.Conn
	wssurl        string
	pushid        string
	isLiveClosed  bool
	// 增加锁防止过早退出
	startWg       sync.WaitGroup
	startedCh    chan struct{}
	stopCh       chan struct{}
}
