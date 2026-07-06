package doubao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"short_videos/internal/endpoints"
	"short_videos/internal/httputil"
	"short_videos/internal/model"
)

const doubaoUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"

type Parser struct {
	client *httputil.Client
	cookie string
}

func New(client *httputil.Client, cookie string) *Parser {
	return &Parser{client: client, cookie: cookie}
}

func (p *Parser) Parse(ctx context.Context, rawURL string) model.Response {
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

	result, err := p.requestAPI(ctx, params)
	if err != nil {
		return model.Fail(500, "解析失败: "+err.Error())
	}

	code, _ := result["code"].(float64)
	if int(code) != 0 {
		return model.Fail(500, "解析失败: API返回错误")
	}

	payload, _ := result["data"].(map[string]any)
	return model.OK("解析成功", formatDoubaoData(payload, videoID, shareID))
}

func formatDoubaoData(payload map[string]any, videoID, shareID string) *model.VideoData {
	data := model.NewVideoData(model.PlatformDoubao, model.MediaTypeVideo)
	data.VideoID = videoID
	if payload == nil {
		return data
	}

	data.Title = pickStr(payload, "title", "video_title", "desc", "description")
	data.Desc = pickStr(payload, "desc", "description", "video_desc")
	data.URL = pickStr(payload, "video_url", "play_url", "url", "download_url")
	data.Cover = pickStr(payload, "cover", "cover_url", "cover_url_list", "poster")

	if info, ok := payload["video_info"].(map[string]any); ok {
		if data.Title == "" {
			data.Title = pickStr(info, "title", "desc")
		}
		if data.URL == "" {
			data.URL = pickStr(info, "video_url", "play_url", "url")
		}
		if data.Cover == "" {
			data.Cover = pickStr(info, "cover", "cover_url")
		}
	}

	if author, ok := payload["author"].(map[string]any); ok {
		data.Author = model.AuthorOf(pickStr(author, "name", "nickname"), pickStr(author, "id", "uid"), pickStr(author, "avatar", "avatar_url"))
	} else {
		data.Author = model.AuthorOf(pickStr(payload, "author_name", "author"), "", pickStr(payload, "author_avatar", "avatar"))
	}

	if data.URL == "" && shareID != "" && videoID != "" {
		data.URL = fmt.Sprintf("%s/video-sharing?share_id=%s&video_id=%s", endpoints.DoubaoOrigin, shareID, videoID)
	}
	return data
}

func pickStr(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
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

func (p *Parser) requestAPI(ctx context.Context, params map[string]string) (map[string]any, error) {
	postBody, _ := json.Marshal(map[string]string{
		"share_id":    params["share_id"],
		"vid":         params["video_id"],
		"creation_id": "",
	})

	referer := fmt.Sprintf(
		endpoints.DoubaoOrigin+"/video-sharing?share_id=%s&source_type=mobile&video_id=%s&share_scene=video_viewer",
		params["share_id"], params["video_id"],
	)

	cookie := p.cookie
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
