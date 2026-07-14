package douyin

import (
	"context"
	"os"
	"testing"
	"time"

	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
)

func TestExtractJSONFromIesMobileHTML(t *testing.T) {
	body, err := os.ReadFile("../../../tmp_ies_mobile.html")
	if err != nil {
		t.Skip("tmp_ies_mobile.html not found, run curl test first")
	}
	detail := extractJSON(string(body))
	if detail == nil {
		t.Fatal("expected detail from _ROUTER_DATA")
	}
	if id := str(detail["aweme_id"]); id != "7637471145263910179" {
		t.Fatalf("unexpected aweme_id: %q", id)
	}
}

func TestParseDouyinVideoIntegration(t *testing.T) {
	if os.Getenv("DOUYIN_INTEGRATION") == "" {
		t.Skip("set DOUYIN_INTEGRATION=1 to run live test")
	}
	p := New(httputil.New(30*time.Second), "")
	resp := p.Parse(context.Background(), parser.Request{
		URL: "https://www.douyin.com/video/7637471145263910179",
	})
	if resp.Code != 200 {
		t.Fatalf("parse failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	data, ok := resp.Data.(*model.VideoData)
	if !ok || data == nil || data.URL == "" {
		t.Fatal("expected video url in response")
	}
}
