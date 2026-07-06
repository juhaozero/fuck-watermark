package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"short_videos/internal/endpoints"
	"short_videos/internal/httputil"
	"short_videos/internal/model"
	"short_videos/internal/parser"
)

var (
	idPatterns = []*regexp.Regexp{
		regexp.MustCompile(`/video/(\d+)`),
		regexp.MustCompile(`modal_id=(\d+)`),
		regexp.MustCompile(`/note/(\d+)`),
		regexp.MustCompile(`/share/slides/(\d+)`),
		regexp.MustCompile(`/share/video/(\d+)`),
	}
	routerDataPattern = regexp.MustCompile(`window\._ROUTER_DATA\s*=\s*(.*?)</script>`)
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
		return model.Fail(400, "请输入抖音链接")
	}

	u := rawURL
	if needsRedirect(u) {
		final, err := p.client.GetFinalURL(ctx, u)
		if err == nil && final != "" {
			u = final
		}
	}

	id := extractID(u)
	if id == "" {
		return model.Fail(400, "链接格式错误，无法提取ID。处理后的链接: "+u)
	}

	apiURL := endpoints.DouyinUserPageBase + "?modal_id=" + id + "&showTab=like"
	body, err := p.client.Get(ctx, apiURL, req.Cookie, map[string]string{
		"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	})
	if err != nil {
		return model.Fail(500, "请求失败: "+err.Error())
	}

	detail := extractJSON(string(body))
	if detail == nil {
		return model.Fail(404, "解析失败，未找到有效内容")
	}

	return model.OK("解析成功", formatData(detail))
}

// needsRedirect 判断是否需要重定向
func needsRedirect(u string) bool {
	if strings.Contains(u, "v.douyin.com") {
		return true
	}
	if !strings.Contains(u, "douyin.com") {
		return true
	}
	return extractID(u) == ""
}

// extractID 提取视频ID
func extractID(u string) string {
	for _, re := range idPatterns {
		if m := re.FindStringSubmatch(u); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// extractJSON 提取JSON数据
func extractJSON(html string) map[string]any {
	start := `<script id="RENDER_DATA" type="application/json">`
	pos := strings.Index(html, start)
	if pos >= 0 {
		rest := html[pos+len(start):]
		end := strings.Index(rest, "</script>")
		if end < 0 {
			return nil
		}
		jsonStr, _ := url.QueryUnescape(rest[:end])
		var data map[string]any
		if json.Unmarshal([]byte(jsonStr), &data) == nil {
			if app, ok := data["app"].(map[string]any); ok {
				if detail, ok := app["videoDetail"].(map[string]any); ok {
					return detail
				}
			}
		}
	}

	if m := routerDataPattern.FindStringSubmatch(html); len(m) > 1 {
		var router map[string]any
		if json.Unmarshal([]byte(m[1]), &router) == nil {
			if loader, ok := router["loaderData"].(map[string]any); ok {
				for key, val := range loader {
					if strings.HasPrefix(key, "video_") {
						if vm, ok := val.(map[string]any); ok {
							if info, ok := vm["videoInfoRes"].(map[string]any); ok {
								if list, ok := info["item_list"].([]any); ok && len(list) > 0 {
									if item, ok := list[0].(map[string]any); ok {
										return item
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// formatData 格式化数据
func formatData(detail map[string]any) *model.VideoData {
	desc := str(detail["desc"])
	result := model.NewVideoData(model.PlatformDouyin, model.MediaTypeUnknown)
	result.Title = desc
	result.Desc = desc
	result.Author = model.AuthorOf(
		firstStr(detail, "authorInfo", "nickname", "author", "nickname"),
		firstStr(detail, "authorInfo", "uid", "author", "uid"),
		firstURL(detail, "authorInfo", "avatarUri", "author", "avatar_thumb", "url_list"),
	)
	// 提取音乐
	result.Music = extractMusic(detail)

	if video, ok := detail["video"].(map[string]any); ok {
		// 提取视频时长
		result.Duration = video["duration"]
	}

	result.Cover = extractCover(detail)

	if images, ok := detail["images"].([]any); ok && len(images) > 0 {
		result.Type = model.MediaTypeImage
		for _, img := range images {
			im, ok := img.(map[string]any)
			if !ok {
				continue
			}
			imgURL := firstURL(im, "urlList", "url_list")
			if imgURL != "" {
				result.Images = append(result.Images, imgURL)
			}
			if liveURL := extractLiveVideo(im); liveURL != "" {
				result.LivePhoto = append(result.LivePhoto, model.LivePhoto{
					Image: imgURL,
					Video: liveURL,
				})
			}
		}
		if len(result.LivePhoto) > 0 {
			result.Type = model.MediaTypeLive
		}
	} else {
		result.Type = model.MediaTypeVideo
		main, backup := extractHighestQualityVideo(detail)
		result.URL = main
		result.VideoBackup = model.BackupsFromURLs(backup...)
		if video, ok := detail["video"].(map[string]any); ok {
			result.VideoID = str(video["uri"])
		}
	}

	return result
}

// extractMusic 提取音乐
func extractMusic(detail map[string]any) *model.Music {
	music, ok := detail["music"].(map[string]any)
	if !ok {
		return nil
	}
	m := &model.Music{
		Title:  firstOf(music, "musicName", "title"),
		Author: firstOf(music, "ownerNickname", "author"),
	}
	if play, ok := music["playUrl"].(map[string]any); ok {
		m.URL = str(play["uri"])
	} else if play, ok := music["play_url"].(map[string]any); ok {
		m.URL = str(play["uri"])
	}
	if cover, ok := music["coverThumb"].(map[string]any); ok {
		if list, ok := cover["urlList"].([]any); ok && len(list) > 0 {
			m.Cover = str(list[0])
		}
	} else if cover, ok := music["cover_thumb"].(map[string]any); ok {
		if list, ok := cover["url_list"].([]any); ok && len(list) > 0 {
			m.Cover = str(list[0])
		}
	}
	return m
}

// extractCover 提取封面
func extractCover(detail map[string]any) string {
	if video, ok := detail["video"].(map[string]any); ok {
		for _, path := range []string{"originCover", "cover", "dynamicCover"} {
			if c, ok := video[path].(map[string]any); ok {
				if u := firstURL(c, "urlList", "url_list"); u != "" {
					return u
				}
			}
			if s, ok := video[path].(string); ok && s != "" {
				return s
			}
		}
	}
	if cover, ok := detail["cover"].(map[string]any); ok {
		if u := firstURL(cover, "url_list"); u != "" {
			return u
		}
	}
	return ""
}

// extractLiveVideo 提取动态视频
func extractLiveVideo(img map[string]any) string {
	video, ok := img["video"].(map[string]any)
	if !ok {
		return ""
	}

	candidates := collectURLs(video)
	return pickBestURL(candidates)
}

// extractHighestQualityVideo 提取最高质量视频
func extractHighestQualityVideo(detail map[string]any) (string, []string) {
	video, ok := detail["video"].(map[string]any)
	if !ok {
		return "", nil
	}

	if list, ok := video["bitRateList"].([]any); ok && len(list) > 0 {
		type item struct {
			bitRate int
			urls    []string
		}
		var items []item
		for _, r := range list {
			rm, ok := r.(map[string]any)
			if !ok {
				continue
			}
			urls := collectURLs(rm)
			if len(urls) == 0 {
				continue
			}
			br := int(num(rm["bitRate"]))
			items = append(items, item{bitRate: br, urls: urls})
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].bitRate > items[j].bitRate
		})

		var main string
		var backup []string
		for _, it := range items {
			best := pickBestURL(it.urls)
			if main == "" && best != "" {
				main = best
			}
			for _, u := range it.urls {
				u = normalizeV26(u)
				if u != main && !contains(backup, u) {
					backup = append(backup, u)
				}
			}
			if main != "" && len(backup) > 0 {
				break
			}
		}
		if main != "" {
			return strings.ReplaceAll(main, "playwm", "play"), backup
		}
	}

	playAPI := str(video["playApi"])
	if playAPI == "" {
		if addr, ok := video["play_addr"].(map[string]any); ok {
			if list, ok := addr["url_list"].([]any); ok && len(list) > 0 {
				playAPI = str(list[0])
			}
		}
	}
	if playAPI != "" {
		return strings.ReplaceAll(playAPI, "playwm", "play"), nil
	}

	uri := str(video["uri"])
	if uri != "" {
		return endpoints.DouyinPlayBase + "?video_id=" + uri + "&ratio=720p&line=0", nil
	}
	return "", nil
}

// collectURLs 收集URL
func collectURLs(m map[string]any) []string {
	var urls []string
	if addrs, ok := m["playAddr"].([]any); ok {
		for _, a := range addrs {
			if am, ok := a.(map[string]any); ok {
				if s := str(am["src"]); s != "" {
					urls = append(urls, s)
				}
			}
		}
	}
	if addr, ok := m["play_addr"].(map[string]any); ok {
		if list, ok := addr["url_list"].([]any); ok {
			for _, u := range list {
				if s := str(u); s != "" {
					urls = append(urls, s)
				}
			}
		}
	}
	if s := str(m["playApi"]); s != "" {
		urls = append(urls, s)
	}
	return urls
}

// pickBestURL 选择最佳URL
func pickBestURL(urls []string) string {
	var v26 string
	for _, u := range urls {
		if strings.Contains(u, "v3-web") {
			return strings.ReplaceAll(u, "playwm", "play")
		}
		// 提取v26-web URL
		if strings.Contains(u, "v26-web") {
			v26 = u
		}
	}
	if v26 != "" {
		return strings.ReplaceAll(normalizeV26(v26), "playwm", "play")
	}
	if len(urls) > 1 {
		return strings.ReplaceAll(urls[1], "playwm", "play")
	}
	if len(urls) > 0 {
		return strings.ReplaceAll(urls[0], "playwm", "play")
	}
	return ""
}

func normalizeV26(u string) string {
	if strings.Contains(u, "v26-web") {
		re := regexp.MustCompile(`://[^/]+`)
		return re.ReplaceAllString(u, "://v26-luna.douyinvod.com")
	}
	return u
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func num(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func firstOf(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := str(m[k]); s != "" {
			return s
		}
	}
	return ""
}

// firstStr 提取第一个字符串
func firstStr(m map[string]any, k1, f1, k2, f2 string) string {
	if sub, ok := m[k1].(map[string]any); ok {
		if s := str(sub[f1]); s != "" {
			return s
		}
	}
	if sub, ok := m[k2].(map[string]any); ok {
		return str(sub[f2])
	}
	return ""
}

func firstURL(m map[string]any, keys ...string) string {
	cur := any(m)
	for i, key := range keys {
		if i == len(keys)-1 {
			if cm, ok := cur.(map[string]any); ok {
				if list, ok := cm[key].([]any); ok && len(list) > 0 {
					return str(list[0])
				}
			}
			return ""
		}
		cm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = cm[key]
	}
	return ""
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
