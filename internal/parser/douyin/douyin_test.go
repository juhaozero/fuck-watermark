package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"fuck-watermark/internal/endpoints"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
)

func TestFormatDataNestedImages(t *testing.T) {
	detail := map[string]any{
		"desc":     "图文测试",
		"aweme_id": "7664591259789001641",
		"images": map[string]any{
			"data": []any{
				map[string]any{
					"group_id_str": "7664591259789001641",
					"data": map[string]any{
						"uri": "tos-cn-i-0813/okCEIGYPIG3ACPXCQe0DfAY9j9AFAAhgAwFqQi",
						"url_list": []any{
							"https://example.com/preview.webp",
							"https://example.com/preview.jpeg",
						},
						"download_url_list": []any{
							"https://example.com/download-water.webp",
						},
						"height": float64(1440),
						"width":  float64(1541),
					},
				},
			},
		},
		"statistics": map[string]any{
			"digg_count":    float64(100),
			"comment_count": float64(20),
			"share_count":   float64(5),
		},
		"create_time": float64(1784551717),
	}

	got := formatData(detail)
	if got.Type != model.MediaTypeImage {
		t.Fatalf("Type = %q, want image", got.Type)
	}
	if got.VideoID != "7664591259789001641" {
		t.Errorf("VideoID = %q", got.VideoID)
	}
	if len(got.Images) != 1 {
		t.Fatalf("Images len = %d, want 1", len(got.Images))
	}
	if got.Images[0] != "https://example.com/preview.jpeg" {
		t.Errorf("Images[0] = %q, want no-water jpeg", got.Images[0])
	}
	if got.Stats == nil || got.Stats.LikeCount != float64(100) {
		t.Errorf("Stats = %+v", got.Stats)
	}
}

func TestFormatDataIesFlatImagesWithMusic(t *testing.T) {
	detail := map[string]any{
		"aweme_id": "7664591259789001641",
		"author": map[string]any{
			"nickname":  "测试作者",
			"short_id":  "87118461779",
			"unique_id": "Ge4ever",
			"avatar_thumb": map[string]any{
				"url_list": []any{"https://example.com/avatar.jpg"},
			},
		},
		"images": []any{
			map[string]any{
				"url_list": []any{
					"https://example.com/a.webp",
					"https://example.com/a.jpeg",
				},
				"download_url_list": []any{
					"https://example.com/a-water.webp",
				},
			},
		},
		"music": map[string]any{
			"title":  "原声",
			"author": "歌手",
			"cover_thumb": map[string]any{
				"url_list": []any{"https://example.com/music.jpg"},
			},
		},
		"video": map[string]any{
			"duration": float64(0),
			"cover": map[string]any{
				"url_list": []any{"https://example.com/cover.jpg"},
			},
			"play_addr": map[string]any{
				"uri": "https://lf26-music-east.douyinstatic.com/obj/ies-music-hj/demo.mp3",
				"url_list": []any{
					"https://aweme.snssdk.com/aweme/v1/playwm/?video_id=https://lf26-music-east.douyinstatic.com/obj/ies-music-hj/demo.mp3",
				},
			},
		},
		"statistics": map[string]any{
			"digg_count": float64(4846),
		},
	}
	got := formatData(detail)
	if got.Type != model.MediaTypeImage {
		t.Fatalf("Type = %q", got.Type)
	}
	if got.Duration != nil {
		t.Errorf("Duration should be omitted for zero, got %#v", got.Duration)
	}
	if got.Author == nil || got.Author.ID != "87118461779" {
		t.Errorf("Author.ID = %#v, want short_id", got.Author)
	}
	if got.Music == nil || got.Music.URL != "https://lf26-music-east.douyinstatic.com/obj/ies-music-hj/demo.mp3" {
		t.Errorf("Music.URL = %#v", got.Music)
	}
	if !strings.Contains(got.Images[0], "a.jpeg") || strings.Contains(got.Images[0], "water") {
		t.Errorf("Images[0] = %q", got.Images[0])
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

func TestFormatDataVideoPlayAddr(t *testing.T) {
	detail := map[string]any{
		"aweme_id": "7637471145263910179",
		"desc":     "测试视频",
		"author": map[string]any{
			"nickname": "作者",
			"short_id": "123",
		},
		"video": map[string]any{
			"duration": float64(1196118), // 毫秒
			"width":    float64(1280),
			"height":   float64(720),
			"cover": map[string]any{
				"url_list": []any{
					"https://example.com/cover.webp",
					"https://example.com/cover.jpeg",
				},
			},
			"play_addr": map[string]any{
				"uri": "v0d00fg10000d7urevnog65rvkh60870",
				"url_list": []any{
					"https://aweme.snssdk.com/aweme/v1/playwm/?line=0&ratio=720p&video_id=v0d00fg10000d7urevnog65rvkh60870",
				},
			},
			"bit_rate": nil,
		},
		"music": map[string]any{
			"title":  "原声",
			"author": "作者",
		},
		"statistics": map[string]any{
			"digg_count": float64(10),
		},
	}
	got := formatData(normalizeDetail(detail))
	if got.Type != model.MediaTypeVideo {
		t.Fatalf("Type = %q", got.Type)
	}
	if got.Duration != 1196 {
		t.Errorf("Duration = %#v, want seconds 1196", got.Duration)
	}
	if got.Quality != "720p" {
		t.Errorf("Quality = %q", got.Quality)
	}
	if !strings.Contains(got.URL, "play/?") || strings.Contains(got.URL, "playwm") {
		t.Errorf("URL should be de-watermarked play link, got %q", got.URL)
	}
	if !strings.Contains(got.Cover, "cover.jpeg") {
		t.Errorf("Cover = %q, want jpeg", got.Cover)
	}
	if got.Music != nil && got.Music.URL != "" {
		t.Errorf("Music.URL should stay empty for video without play_url, got %q", got.Music.URL)
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
	//rawURL := "https://v.douyin.com/otRCSROwSdc/"
	rawURL := "https://www.douyin.com/video/7637471145263910179"
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
