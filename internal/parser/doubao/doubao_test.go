package doubao

import (
	"context"
	"encoding/json"
	"fmt"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/parser"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	client := httputil.New(30 * time.Second)
	p := New(client)

	resp := p.Parse(context.Background(), parser.Request{
		URL: "https://www.doubao.com/video-sharing?share_id=41356597786354690&source_type=mobile&video_id=v0d69cg10004d6978e2ljht0i4fdpp00&share_scene=video_viewer",
	})
	// "https://www.doubao.com/video-sharing?share_id=41356597786354690\u0026video_id=v0d69cg10004d6978e2ljht0i4fdpp00"
	jsonData, _ := json.Marshal(resp.Data)
	fmt.Println("parse doubao video response: ", string(jsonData))
	if resp.Code != 200 {
		t.Errorf("Parse failed: %v", resp.Msg)
	}
	t.Logf("Parse response: %v", string(jsonData))
}
