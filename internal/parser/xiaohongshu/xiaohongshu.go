package xiaohongshu

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"short_videos/internal/httputil"
	"short_videos/internal/model"
	"short_videos/internal/parser"
)

var (
	idPatterns = []*regexp.Regexp{
		regexp.MustCompile(`discovery/item/([a-zA-Z0-9]+)`),
		regexp.MustCompile(`explore/([a-zA-Z0-9]+)`),
		regexp.MustCompile(`item/([a-zA-Z0-9]+)`),
		regexp.MustCompile(`note/([a-zA-Z0-9]+)`),
	}
	initialStatePattern = regexp.MustCompile(`<script>\s*window\.__INITIAL_STATE__\s*=\s*({[\s\S]*?})</script>`)
	tokenPattern        = regexp.MustCompile(`"xsec_token":\s*"([^"]+)"`)
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
		return model.Fail(400, "请输入小红书链接")
	}

	u := strings.ReplaceAll(rawURL, "xhs.com", "xhslink.com")
	var id string

	if strings.Contains(u, "www.xiaohongshu.com") {
		id = extractID(u)
	} else {
		final, err := p.client.GetFinalURL(ctx, u)
		if err == nil && final != "" {
			u = final
		}
		id = extractID(u)
	}

	if id == "" {
		return model.Fail(400, "链接格式错误，无法提取ID。处理后的链接: "+u)
	}

	body, err := p.client.Get(ctx, u, req.Cookie, nil)
	if err != nil {
		return model.Fail(500, "请求失败")
	}

	data := extractJSON(string(body), id)
	if data == nil {
		body, err = p.client.Get(ctx, u, req.Cookie, map[string]string{
			"User-Agent": "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Mobile Safari/537.36",
		})
		if err == nil {
			data = extractJSON(string(body), id)
		}
	}

	if data == nil {
		token := ""
		if m := tokenPattern.FindStringSubmatch(string(body)); len(m) > 1 {
			token = m[1]
		}
		if token != "" {
			apiURL := "https://www.xiaohongshu.com/discovery/item/" + id +
				"?app_platform=android&ignoreEngage=true&app_version=8.69.5&share_from_user_hidden=true&xsec_source=app_share&type=video&xsec_token=" + token
			apiBody, err := p.client.Get(ctx, apiURL, req.Cookie, nil)
			if err == nil {
				data = extractJSON(string(apiBody), id)
			}
		}
	}

	if data == nil {
		return model.Fail(404, "解析失败，未找到有效内容")
	}
	return model.OK("解析成功", data)
}

func extractID(u string) string {
	for _, re := range idPatterns {
		if m := re.FindStringSubmatch(u); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func extractJSON(html, id string) *model.VideoData {
	m := initialStatePattern.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil
	}

	jsonStr := strings.ReplaceAll(m[1], "undefined", "null")
	var root map[string]any
	if json.Unmarshal([]byte(jsonStr), &root) != nil {
		return nil
	}

	var note map[string]any
	if noteMap, ok := root["note"].(map[string]any); ok {
		if detailMap, ok := noteMap["noteDetailMap"].(map[string]any); ok {
			if item, ok := detailMap[id].(map[string]any); ok {
				note, _ = item["note"].(map[string]any)
			}
		}
	}
	if note == nil {
		if noteData, ok := root["noteData"].(map[string]any); ok {
			if data, ok := noteData["data"].(map[string]any); ok {
				note, _ = data["noteData"].(map[string]any)
			}
		}
	}
	if note == nil {
		return nil
	}
	return formatNoteData(note)
}

func formatNoteData(note map[string]any) *model.VideoData {
	typ := str(note["type"])
	if typ == "normal" {
		typ = model.MediaTypeImage
	}

	result := model.NewVideoData(model.PlatformXiaohongshu, typ)
	result.Title = str(note["title"])
	result.Desc = str(note["desc"])

	if user, ok := note["user"].(map[string]any); ok {
		result.Author = model.AuthorOf(
			firstStr(user, "nickname", "nickName"),
			str(user["userId"]),
			str(user["avatar"]),
		)
	}

	result.Cover = extractCover(note, typ)

	if typ == model.MediaTypeVideo || typ == "video" {
		result.Type = model.MediaTypeVideo
		main, backup := extractVideoURLs(note)
		result.URL = main
		if backup != "" {
			result.VideoBackup = model.BackupsFromURLs(backup)
		}
	}

	if list, ok := note["imageList"].([]any); ok {
		for _, item := range list {
			img, ok := item.(map[string]any)
			if !ok {
				continue
			}
			imageURL := firstStr(img, "url", "urlDefault", "urlPre")
			if imageURL != "" {
				result.Images = append(result.Images, processImageURL(imageURL))
			}
			liveURL := extractLiveStream(img)
			if liveURL != "" {
				result.LivePhoto = append(result.LivePhoto, model.LivePhoto{
					Image: processImageURL(imageURL),
					Video: liveURL,
				})
			}
		}
		if len(result.LivePhoto) > 0 {
			result.Type = "live"
		}
	}

	return result
}

func extractCover(note map[string]any, typ string) string {
	if list, ok := note["imageList"].([]any); ok && len(list) > 0 {
		if img, ok := list[0].(map[string]any); ok {
			if u := firstStr(img, "urlPre", "urlDefault", "url"); u != "" {
				return processImageURL(u)
			}
		}
	}
	if typ == "video" {
		if video, ok := note["video"].(map[string]any); ok {
			if image, ok := video["image"].(map[string]any); ok {
				if fid := str(image["thumbnailFileid"]); fid != "" {
					return "https://sns-img-hw.xhscdn.com/" + fid
				}
			}
		}
	}
	if cover, ok := note["cover"].(map[string]any); ok {
		if u := str(cover["url"]); u != "" {
			return processImageURL(u)
		}
		if fid := str(cover["fileId"]); fid != "" {
			return "https://sns-img-hw.xhscdn.com/" + fid + "?imageView2/2/w/0/format/jpg"
		}
	}
	return ""
}

func extractVideoURLs(note map[string]any) (string, string) {
	type stream struct {
		codec   string
		bitrate float64
		url     string
	}
	var streams []stream

	if video, ok := note["video"].(map[string]any); ok {
		if media, ok := video["media"].(map[string]any); ok {
			if streamMap, ok := media["stream"].(map[string]any); ok {
				for _, codec := range []string{"h265", "h264"} {
					if items, ok := streamMap[codec].([]any); ok {
						for _, item := range items {
							if sm, ok := item.(map[string]any); ok {
								streams = append(streams, stream{
									codec:   codec,
									bitrate: num(sm["avgBitrate"]),
									url:     str(sm["masterUrl"]),
								})
							}
						}
					}
				}
			}
		}
		if consumer, ok := video["consumer"].(map[string]any); ok {
			if key := str(consumer["originVideoKey"]); key != "" && len(streams) == 0 {
				return "http://sns-video-bd.xhscdn.com/" + key, ""
			}
		}
	}

	sort.Slice(streams, func(i, j int) bool {
		if streams[i].codec != streams[j].codec {
			if streams[i].codec == "h265" {
				return true
			}
			if streams[j].codec == "h265" {
				return false
			}
		}
		return streams[i].bitrate > streams[j].bitrate
	})

	if len(streams) == 0 {
		return "", ""
	}
	if len(streams) == 1 {
		return streams[0].url, ""
	}
	return streams[0].url, streams[1].url
}

func extractLiveStream(img map[string]any) string {
	stream, ok := img["stream"].(map[string]any)
	if !ok {
		return ""
	}
	for _, codec := range []string{"h264", "h265"} {
		if items, ok := stream[codec].([]any); ok && len(items) > 0 {
			if sm, ok := items[0].(map[string]any); ok {
				if u := str(sm["masterUrl"]); u != "" {
					return u
				}
			}
		}
	}
	return ""
}

func processImageURL(u string) string {
	if u == "" {
		return ""
	}
	patterns := []struct {
		re  *regexp.Regexp
		fmt string
	}{
		{regexp.MustCompile(`/oss-sg/([a-zA-Z0-9_]+)/([a-zA-Z0-9]+)!`), "https://sns-img-hw.xhscdn.com/oss-sg/%s/%s?imageView2/2/w/0/format/jpg"},
		{regexp.MustCompile(`/(notes_pre_post|spectrum|notes_uhdr)/([a-zA-Z0-9]+)`), "https://sns-img-hw.xhscdn.com/%s/%s?imageView2/2/w/0/format/jpg"},
		{regexp.MustCompile(`/([a-zA-Z0-9]+)!`), "https://ci.xiaohongshu.com/%s?imageView2/2/w/0/format/jpg"},
	}
	for _, p := range patterns {
		if m := p.re.FindStringSubmatch(u); len(m) > 2 {
			dir := m[1]
			if matched, _ := regexp.MatchString(`^[a-f0-9]{32}$`, dir); matched {
				continue
			}
			if matched, _ := regexp.MatchString(`^\d+$`, dir); matched {
				continue
			}
			return strings.Replace(strings.Replace(p.fmt, "%s", m[1], 1), "%s", m[2], 1)
		}
		if m := p.re.FindStringSubmatch(u); len(m) > 1 && len(p.fmt) > 0 && strings.Count(p.fmt, "%s") == 1 {
			return strings.Replace(p.fmt, "%s", m[1], 1)
		}
	}
	return u
}

func firstStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := str(m[k]); s != "" {
			return s
		}
	}
	return ""
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

func num(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
