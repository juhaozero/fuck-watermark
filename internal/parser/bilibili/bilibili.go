package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"fuck-watermark/logs"

	"fuck-watermark/internal/endpoints"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
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
		logs.Warnf("[B站] 提取BV号失败 链接=%q 错误=%v", u, err)
		return model.Fail(400, "视频链接好像不太对！")
	}
	logs.Infof("[B站] BV号=%s 链接=%q", bvid, u)

	// 获取视频信息
	viewBody, err := p.client.Get(ctx, endpoints.BilibiliViewAPI+"?bvid="+bvid, req.Cookie, map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
		"User-Agent":   p.ua,
	})
	if err != nil {
		logs.Warnf("[B站] 详情接口请求失败 BV号=%s 错误=%v", bvid, err)
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
		logs.Warnf("[B站] 详情接口解析失败 BV号=%s 接口码=%d 正文=%q", bvid, viewResp.Code, truncate(string(viewBody), 256))
		return model.Fail(404, "解析失败！")
	}
	logs.Infof("[B站] 详情接口成功 BV号=%s 标题=%q 分P数=%d", bvid, viewResp.Data.Title, len(viewResp.Data.Pages))

	// 获取视频分P信息
	var parts []videoPart
	for i, page := range viewResp.Data.Pages {
		playURL := fmt.Sprintf(
			endpoints.BilibiliPlayURLAPI+"?otype=json&fnver=0&fnval=3&player=3&qn=112&bvid=%s&cid=%d&platform=html5&high_quality=1",
			bvid, page.Cid,
		)
		// 获取视频播放地址
		playBody, err := p.client.Get(ctx, playURL, req.Cookie, map[string]string{
			"Content-Type": "application/json;charset=UTF-8",
			"User-Agent":   p.ua,
		})
		if err != nil {
			logs.Warnf("[B站] 播放地址请求失败 BV号=%s cid=%d 分P=%d 错误=%v", bvid, page.Cid, i+1, err)
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
			logs.Warnf("[B站] 播放地址解析失败 BV号=%s cid=%d 分P=%d 正文=%q", bvid, page.Cid, i+1, truncate(string(playBody), 256))
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
		logs.Warnf("[B站] 无可播放分P BV号=%s 总分P=%d", bvid, len(viewResp.Data.Pages))
		return model.Fail(404, "解析失败！")
	}
	logs.Infof("[B站] 解析成功 BV号=%s 分P数=%d", bvid, len(parts))

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
	data.URL = parts[0].URL           // 视频播放地址
	data.Duration = parts[0].Duration // 视频时长
	data.Author = model.AuthorOf(viewResp.Data.Owner.Name, "", viewResp.Data.Owner.Face)
	data.Parts = modelParts

	return model.OK("解析成功", data)
}

// extractBVID 提取B站视频ID
func (p *Parser) extractBVID(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	path := strings.TrimRight(parsed.Path, "/")
	host := parsed.Hostname()

	// 短链接处理
	if host == "b23.tv" || strings.HasSuffix(host, ".b23.tv") {
		logs.Infof("[B站] 正在解析短链 链接=%q 域名=%q", rawURL, host)
		final, err := p.client.HeadRedirect(ctx, rawURL, map[string]string{
			"User-Agent": p.ua,
		})
		if err != nil || final == "" || final == rawURL {
			logs.Infof("[B站] HEAD跳转不可用 链接=%q 错误=%v，改用GET最终地址", rawURL, err)
			final, err = p.client.GetFinalURL(ctx, rawURL)
		}
		if err != nil {
			logs.Warnf("[B站] 短链解析失败 链接=%q 错误=%v", rawURL, err)
			return "", err
		}
		logs.Infof("[B站] 短链已解析 原始=%q 最终=%q", rawURL, final)
		parsed, err = url.Parse(final)
		if err != nil {
			return "", err
		}
		path = strings.TrimRight(parsed.Path, "/")
	}

	if !strings.Contains(path, "/video/") {
		logs.Warnf("[B站] 非视频路径 链接=%q 路径=%q", rawURL, path)
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
