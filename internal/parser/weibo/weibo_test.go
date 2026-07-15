package weibo

import (
	"context"
	"fmt"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	client := httputil.New(30 * time.Second)
	parser := New(client, "")
	videoID := "7637471145263910179"
	resp := parser.fetchVideoInfo(context.Background(), videoID, "")
	if resp.Code != 200 {
		t.Errorf("Fetch video info failed: %v", resp.Msg)
	}
	fmt.Println("fetch video info response: ", resp.Data)
	t.Logf("Fetch video info response: %v", resp.Data)
	data, ok := resp.Data.(*model.VideoData)
	if !ok || data == nil || data.URL == "" {
		t.Fatal("expected video url in response")
	}
	t.Logf("Video URL: %s", data.URL)
}
