package xiaohongshu

import "testing"

func TestUserCacheRemember(t *testing.T) {
	c := &userCache{items: make(map[string]userCacheEntry)}
	c.Remember("u1", "Alice", "http://a")
	p, ok := c.Get("u1")
	if !ok || p.Nickname != "Alice" || p.Avatar != "http://a" {
		t.Fatalf("got %+v ok=%v", p, ok)
	}
	// no session → Get miss on unknown
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss")
	}
}
