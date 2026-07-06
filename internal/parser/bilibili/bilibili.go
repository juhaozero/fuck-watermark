package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"short_videos/internal/endpoints"
	"short_videos/internal/httputil"
	"short_videos/internal/model"
	"short_videos/internal/parser"
)

type Parser struct {
	client *httputil.Client
	ua     string
}

func New(client *httputil.Client) *Parser {
	return &Parser{
		client: client,
		ua:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36",
	}
}

type videoPart struct {
	Title          string `json:"title"`
	Duration       int    `json:"duration"`
	DurationFormat string `json:"durationFormat"`
	URL            string `json:"url"`
	Index          int    `json:"index"`
}

func (p *Parser) Parse(ctx context.Context, req parser.Request) model.Response {
	rawURL := req.URL
	if strings.TrimSpace(rawURL) == "" {
		return model.Fail(400, "链接不能为空！")
	}

	u := cleanURL(rawURL)
	bvid, err := p.extractBVID(ctx, u)
	if err != nil || bvid == "" {
		return model.Fail(400, "视频链接好像不太对！")
	}

	viewBody, err := p.client.Get(ctx, endpoints.BilibiliViewAPI+"?bvid="+bvid, req.Cookie, map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
		"User-Agent":   p.ua,
	})
	if err != nil {
		return model.Fail(500, "请求B站接口失败")
	}

	var viewResp struct {
		Code int `json:"code"`
		Data struct {
			Title string `json:"title"`
			Pic   string `json:"pic"`
			Desc  string `json:"desc"`
			Owner struct {
				Name string `json:"name"`
				Face string `json:"face"`
			} `json:"owner"`
			Pages []struct {
				Cid      int64  `json:"cid"`
				Part     string `json:"part"`
				Duration int    `json:"duration"`
			} `json:"pages"`
		} `json:"data"`
	}
	if json.Unmarshal(viewBody, &viewResp) != nil || viewResp.Code != 0 {
		return model.Fail(404, "解析失败！")
	}

	var parts []videoPart
	for i, page := range viewResp.Data.Pages {
		playURL := fmt.Sprintf(
			endpoints.BilibiliPlayURLAPI+"?otype=json&fnver=0&fnval=3&player=3&qn=112&bvid=%s&cid=%d&platform=html5&high_quality=1",
			bvid, page.Cid,
		)
		playBody, err := p.client.Get(ctx, playURL, req.Cookie, map[string]string{
			"Content-Type": "application/json;charset=UTF-8",
			"User-Agent":   p.ua,
		})
		if err != nil {
			continue
		}

		var playResp struct {
			Data struct {
				Durl []struct {
					URL string `json:"url"`
				} `json:"durl"`
			} `json:"data"`
		}
		if json.Unmarshal(playBody, &playResp) != nil || len(playResp.Data.Durl) == 0 {
			continue
		}

		rawVideo := playResp.Data.Durl[0].URL
		videoURL := rawVideo
		if idx := strings.Index(rawVideo, ".bilivideo.com/"); idx >= 0 {
			videoURL = endpoints.BilibiliMirrorCDN + rawVideo[idx+len(".bilivideo.com/"):]
		}

		dur := page.Duration
		if dur > 0 {
			dur--
		}
		parts = append(parts, videoPart{
			Title:          page.Part,
			Duration:       page.Duration,
			DurationFormat: formatDuration(dur),
			URL:            videoURL,
			Index:          i + 1,
		})
	}

	if len(parts) == 0 {
		return model.Fail(404, "解析失败！")
	}

	modelParts := make([]model.VideoPart, len(parts))
	for i, part := range parts {
		modelParts[i] = model.VideoPart{
			Title:          part.Title,
			URL:            part.URL,
			Duration:       part.Duration,
			DurationFormat: part.DurationFormat,
			Index:          part.Index,
		}
	}

	data := model.NewVideoData(model.PlatformBilibili, model.MediaTypeVideo)
	data.VideoID = bvid
	data.Title = viewResp.Data.Title
	data.Desc = viewResp.Data.Desc
	data.Cover = viewResp.Data.Pic
	data.URL = parts[0].URL
	data.Duration = parts[0].Duration
	data.Author = model.AuthorOf(viewResp.Data.Owner.Name, "", viewResp.Data.Owner.Face)
	data.Parts = modelParts

	return model.OK("解析成功", data)
}

func (p *Parser) extractBVID(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	path := strings.TrimRight(parsed.Path, "/")
	host := parsed.Host

	if host == "b23.tv" {
		final, err := p.client.GetFinalURL(ctx, rawURL)
		if err != nil {
			return "", err
		}
		parsed, err = url.Parse(final)
		if err != nil {
			return "", err
		}
		path = strings.TrimRight(parsed.Path, "/")
	}

	if !strings.Contains(path, "/video/") {
		return "", fmt.Errorf("not a video path")
	}
	return strings.TrimPrefix(path, "/video/"), nil
}

func cleanURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String()
}

func formatDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	t := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(seconds) * time.Second)
	return t.Format("15:04:05")
}
