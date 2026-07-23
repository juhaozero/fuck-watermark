package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"fuck-watermark/internal/endpoints"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
)

func TestFormatDataNestedImages(t *testing.T) {
	detail := map[string]any{
		"desc": "图文测试",
		"images": map[string]any{
			"data": []any{
				map[string]any{
					"group_id_str": "7664591259789001641",
					"data": map[string]any{
						"uri": "tos-cn-i-0813/okCEIGYPIG3ACPXCQe0DfAY9j9AFAAhgAwFqQi",
						"url_list": []any{
							"https://example.com/preview.jpg",
						},
						"download_url_list": []any{
							"https://example.com/download.jpg",
						},
						"height": float64(1440),
						"width":  float64(1541),
					},
				},
			},
		},
	}

	got := formatData(detail)
	if got.Type != model.MediaTypeImage {
		t.Fatalf("Type = %q, want image", got.Type)
	}
	if len(got.Images) != 1 {
		t.Fatalf("Images len = %d, want 1", len(got.Images))
	}
	if got.Images[0] != "https://example.com/download.jpg" {
		t.Errorf("Images[0] = %q, want download url", got.Images[0])
	}
}

func TestFormatDataFlatImages(t *testing.T) {
	detail := map[string]any{
		"desc": "旧结构图文",
		"images": []any{
			map[string]any{
				"url_list": []any{"https://example.com/a.jpg"},
			},
		},
	}
	got := formatData(detail)
	if len(got.Images) != 1 || got.Images[0] != "https://example.com/a.jpg" {
		t.Fatalf("Images = %#v", got.Images)
	}
}

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

func TestParseDouyinVideoIntegrationDetail(t *testing.T) {
	ctx := context.Background()
	rawURL := "https://v.douyin.com/otRCSROwSdc/"
	// rawURL := "https://www.douyin.com/video/7637471145263910179"
	p := New(httputil.New(30*time.Second), "")
	u := rawURL
	if needsRedirect(u) {
		final, err := p.client.GetFinalURL(ctx, u)
		if err == nil && final != "" {
			u = final
		}
	}

	id := extractID(u)
	if id == "" {
		t.Fatalf("extract id failed url=%q resolved=%q", rawURL, u)
	}
	// 方案2：iesdouyin 分享页 + _ROUTER_DATA（需移动端 UA）
	sharePage := endpoints.DouyinIesShareBase + id
	detail := p.fetchPageDetail(ctx, sharePage, "", douyinMobileUA)
	t.Logf("test detail: %v", detail)
	json, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal detail failed err=%v", err)
	}
	fmt.Println("test detail: ", string(json))
}
func TestParseDouyinVideoIntegrationFormatData(t *testing.T) {
	ctx := context.Background()
	rawURL := "https://v.douyin.com/otRCSROwSdc/"
	//rawURL := "https://www.douyin.com/video/7637471145263910179"
	p := New(httputil.New(30*time.Second), "")
	detail := p.Parse(ctx, parser.Request{
		URL: rawURL,
	})
	fmt.Println("test detail: ", detail)
	json, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal detail failed err=%v", err)
	}
	fmt.Println("test detail: ", string(json))
}
