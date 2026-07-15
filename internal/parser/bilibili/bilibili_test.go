package bilibili

import (
	"context"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/parser"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	client := httputil.New(30 * time.Second)
	p := New(client)
	resp := p.Parse(context.Background(), parser.Request{
		URL: "https://www.bilibili.com/video/BV1Qy4y1C7Nf",
	})
	if resp.Code != 200 {
		t.Errorf("Parse failed: %v", resp.Msg)
	}
	t.Logf("Parse response: %v", resp.Data)
}
