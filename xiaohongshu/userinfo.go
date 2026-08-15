package xiaohongshu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// UserProfile is a cached public profile snippet.
type UserProfile struct {
	Nickname string
	Avatar   string
}

type userCache struct {
	mu    sync.RWMutex
	items map[string]userCacheEntry
	// session cookies for signed requests (updated on each listenOnce)
	sessMu sync.RWMutex
	sess   *GuestSession
}

type userCacheEntry struct {
	p   UserProfile
	exp time.Time
}

var globalUserCache = &userCache{items: make(map[string]userCacheEntry)}

const userCacheTTL = 30 * time.Minute

func (c *userCache) setSession(s *GuestSession) {
	c.sessMu.Lock()
	c.sess = s
	c.sessMu.Unlock()
}

func (c *userCache) getSession() *GuestSession {
	c.sessMu.RLock()
	defer c.sessMu.RUnlock()
	return c.sess
}

// Get returns cached profile or fetches via edith otherinfo.
func (c *userCache) Get(userID string) (UserProfile, bool) {
	if userID == "" {
		return UserProfile{}, false
	}
	c.mu.RLock()
	if e, ok := c.items[userID]; ok && time.Now().Before(e.exp) {
		c.mu.RUnlock()
		return e.p, true
	}
	c.mu.RUnlock()

	p, err := fetchOtherInfo(c.getSession(), userID)
	if err != nil {
		return UserProfile{}, false
	}
	c.mu.Lock()
	c.items[userID] = userCacheEntry{p: p, exp: time.Now().Add(userCacheTTL)}
	c.mu.Unlock()
	return p, true
}

// Remember stores a profile seen on the wire (chat/join/gift) so praise can reuse it.
func (c *userCache) Remember(userID, nickname, avatar string) {
	if userID == "" || nickname == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// keep existing avatar if new one empty
	if avatar == "" {
		if e, ok := c.items[userID]; ok && e.p.Avatar != "" {
			avatar = e.p.Avatar
		}
	}
	c.items[userID] = userCacheEntry{
		p:   UserProfile{Nickname: nickname, Avatar: avatar},
		exp: time.Now().Add(userCacheTTL),
	}
}

// Prefetch kicks off async fetch (non-blocking).
func (c *userCache) Prefetch(userID string) {
	if userID == "" {
		return
	}
	c.mu.RLock()
	if e, ok := c.items[userID]; ok && time.Now().Before(e.exp) {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()
	go func() { _, _ = c.Get(userID) }()
}

type otherInfoResp struct {
	Code    int  `json:"code"`
	Success bool `json:"success"`
	Data    struct {
		BasicInfo struct {
			Nickname string `json:"nickname"`
			Images   string `json:"images"`
			ImageB   string `json:"imageb"`
		} `json:"basic_info"`
	} `json:"data"`
}

func fetchOtherInfo(sess *GuestSession, userID string) (UserProfile, error) {
	if sess == nil {
		return UserProfile{}, fmt.Errorf("no session")
	}
	uri := "/api/sns/web/v1/user/otherinfo"
	params := map[string]any{"target_user_id": userID}
	cookies := map[string]string{
		"a1": sess.A1, "webId": sess.WebID, "web_session": sess.WebSession, "xsecappid": xsecAppID,
	}
	hs, err := SignHeaders(http.MethodGet, uri, cookies, params)
	if err != nil {
		return UserProfile{}, err
	}
	q := url.Values{}
	q.Set("target_user_id", userID)
	req, err := http.NewRequest(http.MethodGet, edithHost+uri+"?"+q.Encode(), nil)
	if err != nil {
		return UserProfile{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://www.xiaohongshu.com/")
	req.Header.Set("Origin", "https://www.xiaohongshu.com")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Cookie", fmt.Sprintf("a1=%s; webId=%s; web_session=%s; xsecappid=%s",
		sess.A1, sess.WebID, sess.WebSession, xsecAppID))
	for k, v := range hs {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return UserProfile{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserProfile{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return UserProfile{}, fmt.Errorf("http %d", resp.StatusCode)
	}
	var out otherInfoResp
	if err := json.Unmarshal(body, &out); err != nil {
		return UserProfile{}, err
	}
	if !out.Success || out.Code != 0 {
		return UserProfile{}, fmt.Errorf("otherinfo code=%d", out.Code)
	}
	av := out.Data.BasicInfo.Images
	if av == "" {
		av = out.Data.BasicInfo.ImageB
	}
	if out.Data.BasicInfo.Nickname == "" {
		return UserProfile{}, fmt.Errorf("empty nickname")
	}
	return UserProfile{Nickname: out.Data.BasicInfo.Nickname, Avatar: av}, nil
}
