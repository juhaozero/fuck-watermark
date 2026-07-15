package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"fuck-watermark/internal/endpoints"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
)

const (
	douyinUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// iesdouyin 分享页需移动端 UA，桌面 UA 会返回 byted_acrawler 反爬页
	douyinMobileUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1 Edg/122.0.0.0"
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
	Cookie string
}

func New(client *httputil.Client, cookie string) *Parser {
	return &Parser{client: client, Cookie: cookie}
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
	if req.Cookie == "" {
		req.Cookie = p.Cookie
	}

	id := extractID(u)
	if id == "" {
		log.Printf("[douyin] extract id failed url=%q resolved=%q", rawURL, u)
		return model.Fail(400, "链接格式错误，无法提取ID。处理后的链接: "+u)
	}
	log.Printf("[douyin] aweme_id=%s url=%q", id, u)
	if req.Cookie != "" {
		// 方案1：user/self?modal_id + RENDER_DATA（通常需 cookie）
		modalPage := endpoints.DouyinUserPageBase + "?modal_id=" + id + "&showTab=like"
		if detail := p.fetchPageDetail(ctx, modalPage, req.Cookie, douyinUA); detail != nil {
			log.Printf("[douyin] parse ok aweme_id=%s source=modal_page ", id)
			return model.OK("解析成功", formatData(normalizeDetail(detail)))
		}
	}
	// 方案2：iesdouyin 分享页 + _ROUTER_DATA（需移动端 UA）
	sharePage := endpoints.DouyinIesShareBase + id
	if detail := p.fetchPageDetail(ctx, sharePage, req.Cookie, douyinMobileUA); detail != nil {
		log.Printf("[douyin] parse ok aweme_id=%s source=iesdouyin", id)
		return model.OK("解析成功", formatData(normalizeDetail(detail)))
	}

	log.Printf("[douyin] parse failed aweme_id=%s", id)
	return model.Fail(404, "解析失败，未找到有效内容（可尝试传入 cookie 参数）")
}

func douyinHTMLHeaders(referer, ua string) map[string]string {
	return map[string]string{
		"User-Agent":      ua,
		"Referer":         referer,
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
	}
}

func (p *Parser) fetchPageDetail(ctx context.Context, pageURL, cookie, ua string) map[string]any {
	referer := pageURL
	if strings.Contains(pageURL, "iesdouyin.com") {
		referer = "https://www.douyin.com/"
	}
	body, err := p.client.Get(ctx, pageURL, cookie, douyinHTMLHeaders(referer, ua))
	if err != nil {
		log.Printf("[douyin] page request failed url=%q err=%v", pageURL, err)
		return nil
	}
	html := string(body)
	if strings.Contains(html, "byted_acrawler") || strings.Contains(html, "__ac_signature") {
		log.Printf("[douyin] page anti-bot challenge url=%q", pageURL)
		return nil
	}
	detail := extractJSON(html)
	if detail == nil {
		log.Printf("[douyin] page no render data url=%q body=%q", pageURL, truncate(html, 256))
	}
	return detail
}

func normalizeDetail(detail map[string]any) map[string]any {
	if video, ok := detail["video"].(map[string]any); ok {
		if _, ok := video["bitRateList"]; !ok {
			if br, ok := video["bit_rate"].([]any); ok {
				video["bitRateList"] = br
			}
		}
	}
	return detail
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
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
		firstAuthorID(detail),
		firstAuthorAvatar(detail),
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
		for _, path := range []string{"originCover", "origin_cover", "cover", "dynamicCover", "dynamic_cover"} {
			if c, ok := video[path].(map[string]any); ok {
				if u := urlFromList(c); u != "" {
					return u
				}
			}
			if s, ok := video[path].(string); ok && s != "" {
				return s
			}
		}
	}
	if cover, ok := detail["cover"].(map[string]any); ok {
		if u := urlFromList(cover); u != "" {
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
			br := int(num(firstOfNum(rm, "bitRate", "bit_rate")))
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

	// iesdouyin 等来源常见 play_addr.url_list，优先于 playApi / snssdk 兜底
	if urls := collectURLs(video); len(urls) > 0 {
		main := pickBestURL(urls)
		var backup []string
		for _, u := range urls {
			u = strings.ReplaceAll(normalizeV26(u), "playwm", "play")
			if u != main && !contains(backup, u) {
				backup = append(backup, u)
			}
		}
		if main != "" {
			return strings.ReplaceAll(main, "playwm", "play"), backup
		}
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

func firstOfNum(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
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
func firstAuthorID(detail map[string]any) string {
	if id := firstStr(detail, "authorInfo", "uid", "author", "uid"); id != "" {
		return id
	}
	return firstStr(detail, "authorInfo", "uid", "author", "unique_id")
}

func firstAuthorAvatar(detail map[string]any) string {
	if u := firstURL(detail, "authorInfo", "avatarUri"); u != "" {
		return u
	}
	if u := firstURL(detail, "author", "avatar_thumb", "url_list"); u != "" {
		return u
	}
	return firstURL(detail, "author", "avatar_medium", "url_list")
}

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

// urlFromList 从同一对象读取 urlList 或 url_list（iesdouyin 为 snake_case）
func urlFromList(m map[string]any) string {
	for _, key := range []string{"urlList", "url_list"} {
		if list, ok := m[key].([]any); ok && len(list) > 0 {
			if u := str(list[0]); u != "" {
				return u
			}
		}
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
