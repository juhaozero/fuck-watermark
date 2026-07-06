package toutiao

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"fuck-watermark/internal/endpoints"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
)

var (
	videoIDPattern = regexp.MustCompile(`video/([0-9]+)`)
	numIDPattern   = regexp.MustCompile(`[0-9]+`)
)

type Parser struct {
	client *httputil.Client
}

func New(client *httputil.Client) *Parser {
	return &Parser{client: client}
}

func (p *Parser) Parse(ctx context.Context, req parser.Request) model.Response {
	rawURL := req.URL
	if strings.TrimSpace(rawURL) == "" {
		return model.Fail(400, "url为空")
	}

	id, err := p.extractID(ctx, rawURL)
	if err != nil || id == "" {
		return model.Fail(400, "无法解析视频 ID")
	}

	body, err := p.client.Get(ctx, endpoints.ToutiaoVideoPage+id, req.Cookie, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36",
	})
	if err != nil {
		return model.Fail(500, "请求失败")
	}

	data, err := extractRenderData(string(body))
	if err != nil {
		return model.Fail(404, err.Error())
	}

	item, ok := data["data"].(map[string]any)
	if !ok || str(item["itemId"]) == "" {
		return model.Fail(404, "当前分享链接已失效！")
	}

	initial, _ := item["initialVideo"].(map[string]any)
	cell, _ := initial["itemCell"].(map[string]any)
	user, _ := cell["userInfo"].(map[string]any)
	ability, _ := cell["videoAbility"].(map[string]any)
	playInfo, _ := initial["videoPlayInfo"].(map[string]any)

	videoURL := ""
	if list, ok := playInfo["video_list"].([]any); ok {
		for _, idx := range []int{2, 1, 0} {
			if idx < len(list) {
				if vm, ok := list[idx].(map[string]any); ok {
					if u := str(vm["main_url"]); u != "" {
						videoURL = u
						break
					}
				}
			}
		}
	}

	return model.OK("解析成功", formatToutiao(item, initial, cell, user, ability, playInfo, videoURL))
}

func formatToutiao(item, initial, cell, user, ability, playInfo map[string]any, videoURL string) *model.VideoData {
	data := model.NewVideoData(model.PlatformToutiao, model.MediaTypeVideo)
	data.VideoID = str(item["itemId"])
	data.Title = str(initial["title"])
	data.Desc = str(user["description"])
	data.Cover = str(initial["coverUrl"])
	data.URL = videoURL
	data.Author = model.AuthorOf(str(user["name"]), str(user["userID"]), str(user["avatarURL"]))
	if ability != nil {
		if music, ok := ability["music"]; ok {
			data.Music = &model.Music{Title: str(music)}
		}
	}
	_ = cell
	_ = playInfo
	return data
}

func (p *Parser) extractID(ctx context.Context, rawURL string) (string, error) {
	final, err := p.client.GetFinalURL(ctx, rawURL)
	if err != nil {
		final = rawURL
	}
	if m := videoIDPattern.FindStringSubmatch(final); len(m) > 1 {
		return m[1], nil
	}
	if m := numIDPattern.FindString(final); m != "" {
		return m, nil
	}
	return "", nil
}

func extractRenderData(html string) (map[string]any, error) {
	start := `<script id="RENDER_DATA" type="application/json">`
	end := `</script>`
	pos := strings.Index(html, start)
	if pos < 0 {
		return nil, errMsg("无法找到数据")
	}
	rest := html[pos+len(start):]
	posEnd := strings.Index(rest, end)
	if posEnd < 0 {
		return nil, errMsg("无法正确提取JSON数据，未找到结束标签")
	}
	jsonStr, err := url.QueryUnescape(rest[:posEnd])
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, errMsg("JSON解析失败：" + err.Error())
	}
	return data, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func errMsg(msg string) error { return simpleError(msg) }
