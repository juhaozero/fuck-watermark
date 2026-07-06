package weibo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"short_videos/internal/endpoints"
	"short_videos/internal/httputil"
	"short_videos/internal/model"
	"short_videos/internal/parser"
)

var tvPathPattern = regexp.MustCompile(`weibo\.com/tv/(show|v)/([^?&]+)`)

const weiboUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"

type Parser struct {
	client    *httputil.Client
	proxyBase string
}

func New(client *httputil.Client, proxyBase string) *Parser {
	return &Parser{client: client, proxyBase: proxyBase}
}

func (p *Parser) Parse(ctx context.Context, req parser.Request) model.Response {
	rawURL := req.URL
	if strings.TrimSpace(rawURL) == "" {
		return model.Fail(400, "参数url不能为空")
	}

	videoID, err := p.extractVideoID(ctx, rawURL)
	if err != nil || videoID == "" {
		return model.Fail(404, fmt.Sprintf("无法从URL中提取视频ID: %s", rawURL))
	}

	return p.fetchVideoInfo(ctx, videoID, req.Cookie)
}

func (p *Parser) extractVideoID(ctx context.Context, rawURL string) (string, error) {
	if strings.Contains(rawURL, "video.weibo.com/show") {
		return queryParam(rawURL, "fid"), nil
	}

	if strings.Contains(rawURL, "weibo.com/tv/") {
		if id := queryParam(rawURL, "fid"); id != "" {
			return id, nil
		}
		if m := tvPathPattern.FindStringSubmatch(rawURL); len(m) > 2 {
			return m[2], nil
		}
	}

	if strings.Contains(rawURL, "t.cn/") {
		loc, err := p.client.HeadRedirect(ctx, rawURL, map[string]string{
			"User-Agent": weiboUA,
		})
		if err != nil || loc == "" {
			return "", err
		}
		return p.extractVideoID(ctx, loc)
	}

	return "", nil
}

func (p *Parser) fetchVideoInfo(ctx context.Context, videoID, cookie string) model.Response {
	pagePath := "/tv/show/" + videoID
	apiURL := endpoints.WeiboTVAPI + "?page=" + url.QueryEscape(pagePath)

	payload := map[string]any{
		"Component_Play_Playinfo": map[string]string{"oid": videoID},
	}
	payloadJSON, _ := json.Marshal(payload)
	form := "data=" + url.QueryEscape(string(payloadJSON))

	body, err := p.client.Post(ctx, apiURL, strings.NewReader(form), cookie, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Referer":      endpoints.WeiboReferer,
		"User-Agent":   weiboUA,
	})
	if err != nil {
		return model.Fail(500, "API请求失败: "+err.Error())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			PlayInfo map[string]any `json:"Component_Play_Playinfo"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &resp) != nil {
		return model.Fail(500, "API响应解析失败")
	}

	info := resp.Data.PlayInfo
	if resp.Code != 100000 || info == nil {
		msg := "解析失败,当前链接中的视频不存在。"
		if resp.Code == 0 {
			msg = "当前官方接口已失效！"
		} else if resp.Code != 100000 {
			msg = "官方接口响应错误！"
		}
		return model.Fail(404, msg)
	}

	return model.OK("解析成功", formatVideoInfo(info, p.proxyBase))
}

func formatVideoInfo(info map[string]any, proxyBase string) *model.VideoData {
	data := model.NewVideoData(model.PlatformWeibo, model.MediaTypeVideo)
	data.Title = str(info["title"])
	data.Desc = str(info["title"])

	var backups []model.VideoBackup
	bestPriority := -1

	if urls, ok := info["urls"].(map[string]any); ok {
		for quality, raw := range urls {
			u := str(raw)
			if u == "" {
				continue
			}
			fullURL := wrapURL(u, proxyBase)
			qKey, priority := qualityMeta(quality)
			backups = append(backups, model.VideoBackup{Label: quality, Quality: qKey, URL: fullURL})
			if priority > bestPriority {
				bestPriority = priority
				data.URL = fullURL
				data.Quality = qKey
			}
		}
	}
	data.VideoBackup = backups

	if v, ok := info["duration_time"]; ok {
		switch t := v.(type) {
		case float64:
			data.Duration = int(t)
		case string:
			if n, err := parseInt(t); err == nil {
				data.Duration = n
			}
		}
	}

	data.Author = model.AuthorOf(
		str(info["author"]),
		str(info["author_id"]),
		wrapURL(str(info["avatar"]), proxyBase),
	)
	data.Cover = wrapURL(str(info["cover_image"]), proxyBase)
	data.Stats = &model.Stats{
		PlayCount:       info["play_count"],
		RepostCount:     info["reposts_count"],
		CommentCount:    info["comments_count"],
		AttitudeCount:   info["attitudes_count"],
		IPInfo:          info["ip_info_str"],
		PublishedAt:     info["date"],
	}

	return data
}

func wrapURL(raw, proxyBase string) string {
	if raw == "" {
		return ""
	}
	full := raw
	if strings.HasPrefix(raw, "//") {
		full = "https:" + raw
	} else if !strings.HasPrefix(raw, "http") {
		full = "https://" + strings.TrimPrefix(raw, "/")
	}
	if proxyBase == "" {
		return full
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(full))
	return strings.TrimRight(proxyBase, "/") + "/?type=weibo&proxyurl=" + encoded
}

func qualityMeta(label string) (string, int) {
	switch {
	case strings.Contains(label, "2K"):
		return "origin", 4
	case strings.Contains(label, "1080P"):
		return "origin", 3
	case strings.Contains(label, "720P"):
		return "origin", 2
	case strings.Contains(label, "480P"):
		return "hd", 1
	case strings.Contains(label, "360P"):
		return "sd", 0
	default:
		return "unknown", 0
	}
}

func queryParam(rawURL, key string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get(key)
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
