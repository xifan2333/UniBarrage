package xiaohongshu

import (
	"errors"
	"fmt"
	"testing"
)

func TestAckErrorRoomClosed(t *testing.T) {
	resp := map[string]any{
		"b": map[string]any{
			"a": map[string]any{"c": float64(9001), "m": "房间已关闭"},
		},
	}
	err := ackError("join", resp)
	var pe *permanentError
	if !errors.As(err, &pe) {
		t.Fatalf("want permanent, got %v", err)
	}
	if pe.msg != "小红书房间不存在或已关闭" {
		t.Fatalf("msg=%q", pe.msg)
	}
}

func TestShortErrNoMap(t *testing.T) {
	s := shortErr(fmt.Errorf("join: map[b:map[a:map[c:9001]]]"))
	if s != "join" {
		t.Fatalf("got %q", s)
	}
}
