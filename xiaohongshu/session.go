package xiaohongshu

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	edithHost = "https://edith.xiaohongshu.com"
	userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"
)

// GuestSession holds tourist identity for RWP danmaku.
type GuestSession struct {
	A1        string
	WebID     string
	WebSession string
	UserID    string
	DeviceID  string
	ALt       string
	RLt       string
	ExpiredAt time.Time
	Cookies   map[string]string
}

type activateResp struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Data    struct {
		UserID         string `json:"user_id"`
		Session        string `json:"session"`
		SecureSession  string `json:"secure_session"`
		SSK            string `json:"ssk"`
	} `json:"data"`
}

type ltResp struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Data    struct {
		ALt         string `json:"a_lt"`
		RLt         string `json:"r_lt"`
		ExpiredTime int    `json:"expired_time"`
	} `json:"data"`
}

func randomDeviceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// UUID v4-ish
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func randomPubKeyB64() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func cookieHeader(m map[string]string) string {
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

// doEdith signs and sends an edith request.
// bodyJSON is the exact POST body used both for signing and on the wire (nil for GET).
func doEdith(method, uri string, cookies map[string]string, bodyJSON []byte, extra map[string]string) ([]byte, http.Header, error) {
	var payload any
	if bodyJSON != nil {
		payload = json.RawMessage(bodyJSON)
	}
	sig, err := SignHeaders(method, uri, cookies, payload)
	if err != nil {
		return nil, nil, err
	}
	var body io.Reader
	if bodyJSON != nil {
		body = bytes.NewReader(bodyJSON)
	}
	req, err := http.NewRequest(method, edithHost+uri, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://www.xiaohongshu.com/")
	req.Header.Set("Origin", "https://www.xiaohongshu.com")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cookie", cookieHeader(cookies))
	if bodyJSON != nil {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}
	for k, v := range sig {
		req.Header.Set(k, v)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 400 {
		return data, resp.Header, fmt.Errorf("edith %s %s: HTTP %d: %s", method, uri, resp.StatusCode, truncate(string(data), 200))
	}
	return data, resp.Header, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// CreateGuestSession: a1 → activate → celestial/lt
func CreateGuestSession() (*GuestSession, error) {
	a1 := GenerateA1()
	webID := GenerateWebID(a1)
	deviceID := randomDeviceID()
	cookies := map[string]string{
		"a1":         a1,
		"webId":      webID,
		"xsecappid":  xsecAppID,
		"webBuild":   "6.41.3",
		"loadts":     fmt.Sprintf("%d", time.Now().UnixMilli()),
	}

	// 1) activate — fixed key order body for stable x-s
	actURI := "/api/sns/web/v1/login/activate"
	actBody := []byte(`{"client_public_key_base64":"` + randomPubKeyB64() + `"}`)
	raw, _, err := doEdith(http.MethodPost, actURI, cookies, actBody, nil)
	if err != nil {
		return nil, fmt.Errorf("activate: %w", err)
	}
	var act activateResp
	if err := json.Unmarshal(raw, &act); err != nil {
		return nil, fmt.Errorf("activate decode: %w body=%s", err, truncate(string(raw), 200))
	}
	if !act.Success || act.Code != 0 || act.Data.Session == "" || act.Data.UserID == "" {
		return nil, fmt.Errorf("activate failed: code=%d msg=%s body=%s", act.Code, act.Msg, truncate(string(raw), 300))
	}
	cookies["web_session"] = act.Data.Session

	// 2) celestial/lt
	ltURI := "/api/sns/web/v1/celestial/lt"
	raw, _, err = doEdith(http.MethodGet, ltURI, cookies, nil /*GET*/, map[string]string{
		"c_device_id": deviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("celestial/lt: %w", err)
	}
	var lt ltResp
	if err := json.Unmarshal(raw, &lt); err != nil {
		return nil, fmt.Errorf("lt decode: %w body=%s", err, truncate(string(raw), 200))
	}
	if !lt.Success || lt.Code != 0 || lt.Data.ALt == "" {
		return nil, fmt.Errorf("lt failed: code=%d msg=%s body=%s", lt.Code, lt.Msg, truncate(string(raw), 300))
	}
	exp := lt.Data.ExpiredTime
	if exp <= 0 {
		exp = 10080
	}
	// half-life cache like frontend
	expiredAt := time.Now().Add(time.Duration(exp) * time.Second / 2)

	return &GuestSession{
		A1:         a1,
		WebID:      webID,
		WebSession: act.Data.Session,
		UserID:     act.Data.UserID,
		DeviceID:   deviceID,
		ALt:        lt.Data.ALt,
		RLt:        lt.Data.RLt,
		ExpiredAt:  expiredAt,
		Cookies:    cookies,
	}, nil
}

// RefreshToken re-fetches a_lt (same cookies/device).
func (s *GuestSession) RefreshToken() error {
	if s == nil {
		return fmt.Errorf("nil session")
	}
	raw, _, err := doEdith(http.MethodGet, "/api/sns/web/v1/celestial/lt", s.Cookies, nil, map[string]string{
		"c_device_id": s.DeviceID,
	})
	if err != nil {
		return err
	}
	var lt ltResp
	if err := json.Unmarshal(raw, &lt); err != nil {
		return err
	}
	if !lt.Success || lt.Code != 0 || lt.Data.ALt == "" {
		return fmt.Errorf("lt refresh failed: code=%d msg=%s", lt.Code, lt.Msg)
	}
	s.ALt = lt.Data.ALt
	s.RLt = lt.Data.RLt
	exp := lt.Data.ExpiredTime
	if exp <= 0 {
		exp = 10080
	}
	s.ExpiredAt = time.Now().Add(time.Duration(exp) * time.Second / 2)
	return nil
}
