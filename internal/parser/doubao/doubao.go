package doubao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"fuck-watermark/internal/endpoints"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
)

const doubaoUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"

type Parser struct {
	client *httputil.Client
}

func New(client *httputil.Client) *Parser {
	return &Parser{client: client}
}

func (p *Parser) Parse(ctx context.Context, req parser.Request) model.Response {
	rawURL := req.URL
	if strings.TrimSpace(rawURL) == "" {
		return model.Fail(400, "请输入豆包视频链接")
	}

	realURL, err := p.client.GetFinalURL(ctx, rawURL)
	if err != nil || realURL == "" {
		realURL = rawURL
	}

	params := extractParams(realURL)
	shareID := params["share_id"]
	videoID := params["video_id"]
	if shareID == "" || videoID == "" {
		return model.Fail(400, "无法从链接中提取必要参数")
	}

	result, err := p.requestAPI(ctx, params, req.Cookie)
	if err != nil {
		return model.Fail(500, "解析失败: "+err.Error())
	}

	code, _ := result["code"].(float64)
	if int(code) != 0 {
		return model.Fail(500, "解析失败: API返回错误")
	}

	payload, _ := result["data"].(map[string]any)
	return model.OK("解析成功", formatDoubaoData(payload, videoID))
}

// 格式化豆包视频数据。
// payload 结构示例:
//
//	play_info.data.{main,backup,definition,poster_url}
//	user_info.data.{user_id,nickname}
//	prompt.data (文案)
//	videoID
func formatDoubaoData(payload map[string]any, videoID string) *model.VideoData {
	data := model.NewVideoData(model.PlatformDoubao, model.MediaTypeVideo)
	data.VideoID = videoID
	if payload == nil {
		return data
	}
	if vid := pickStr(payload, "videoID", "video_id"); vid != "" {
		data.VideoID = vid
	}

	if prompt := unwrapData(payload["prompt"]); prompt != nil {
		if s, ok := prompt.(string); ok {
			data.Title = s
			data.Desc = s
		}
	}

	if play, ok := asMap(unwrapData(payload["play_info"])); ok {
		data.URL = pickStr(play, "main", "video_url", "play_url", "url")
		data.Cover = pickStr(play, "poster_url", "cover", "cover_url")
		data.Quality = pickStr(play, "definition")
		if backup := pickStr(play, "backup"); backup != "" {
			data.VideoBackup = model.BackupsFromURLs(backup)
		}
	}

	if user, ok := asMap(unwrapData(payload["user_info"])); ok {
		data.Author = model.AuthorOf(
			pickStr(user, "nickname", "user_name", "name"),
			anyToStr(user["user_id"]),
			pickStr(user, "avatar", "avatar_url"),
		)
	}

	return data
}

// unwrapData 解包 API 常见的 { "data": T } 包装；若本身已是目标值则原样返回。
func unwrapData(v any) any {
	if m, ok := v.(map[string]any); ok {
		if inner, exists := m["data"]; exists {
			return inner
		}
	}
	return v
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func pickStr(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, key := range keys {
		if s := anyToStr(m[key]); s != "" {
			return s
		}
	}
	return ""
}

func anyToStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		// JSON 数字默认 float64，user_id 等需原样转字符串
		if x == float64(int64(x)) {
			return fmt.Sprintf("%.0f", x)
		}
		return fmt.Sprintf("%v", x)
	case json.Number:
		return x.String()
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case uint64:
		return fmt.Sprintf("%d", x)
	default:
		return ""
	}
}

func extractParams(rawURL string) map[string]string {
	out := map[string]string{}
	u, err := url.Parse(rawURL)
	if err != nil {
		return out
	}
	for k, v := range u.Query() {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

func (p *Parser) requestAPI(ctx context.Context, params map[string]string, cookie string) (map[string]any, error) {
	postBody, _ := json.Marshal(map[string]string{
		"share_id":    params["share_id"],
		"vid":         params["video_id"],
		"creation_id": "",
	})

	referer := fmt.Sprintf(
		endpoints.DoubaoOrigin+"/video-sharing?share_id=%s&source_type=mobile&video_id=%s&share_scene=video_viewer",
		params["share_id"], params["video_id"],
	)

	if cookie == "" {
		cookie = "i18next=zh-CN"
	}

	body, err := p.client.Post(ctx, endpoints.DoubaoShareAPI, bytes.NewReader(postBody), cookie, map[string]string{
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "zh-CN,zh;q=0.9",
		"Agw-Js-Conv":     "str",
		"Content-Type":    "application/json",
		"Origin":          endpoints.DoubaoOrigin,
		"Referer":         referer,
		"User-Agent":      doubaoUA,
	})
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("响应解析失败")
	}
	return result, nil
}
