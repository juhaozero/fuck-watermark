package kuaishou

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
)

// ksjx.php 使用 iPhone 移动 UA，桌面 UA 易触发风控
const kuaishouUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1 Edg/122.0.0.0"

var (
	initStatePattern   = regexp.MustCompile(`window\.INIT_STATE\s*=\s*(.*?)</script>`)
	apolloStatePattern = regexp.MustCompile(`window\.__APOLLO_STATE__\s*=\s*(.*?)</script>`)
	contentPatterns    = map[string]*regexp.Regexp{
		"short-video": regexp.MustCompile(`short-video/([^?]+)`),
		"long-video":  regexp.MustCompile(`long-video/([^?]+)`),
		"photo":       regexp.MustCompile(`photo/([^?]+)`),
	}
)

type Parser struct {
	client *httputil.Client
}

func New(client *httputil.Client) *Parser {
	return &Parser{client: client}
}

func kuaishouHTMLHeaders() map[string]string {
	return map[string]string{
		"User-Agent":                kuaishouUA,
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "zh-CN,zh;q=0.9",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
	}
}

func (p *Parser) Parse(ctx context.Context, req parser.Request) model.Response {
	rawURL := req.URL
	if strings.TrimSpace(rawURL) == "" {
		return model.Fail(400, "请输入快手链接")
	}

	headers := kuaishouHTMLHeaders()
	redirectURL, err := p.client.GetFinalURL(ctx, rawURL, headers)
	if err != nil || redirectURL == "" {
		log.Printf("[kuaishou] redirect failed url=%q err=%v", rawURL, err)
		return model.Fail(400, "无法获取有效链接")
	}
	log.Printf("[kuaishou] resolved url=%q final=%q", rawURL, redirectURL)

	page, err := p.client.Get(ctx, redirectURL, req.Cookie, headers)
	if err != nil {
		log.Printf("[kuaishou] page request failed url=%q err=%v", redirectURL, err)
		return model.Fail(500, "页面内容获取失败")
	}

	contentType, contentID := extractContentIDAndType(redirectURL)
	if contentID == "" {
		log.Printf("[kuaishou] unknown content type url=%q", redirectURL)
		return model.Fail(400, "无法识别的链接类型")
	}
	log.Printf("[kuaishou] content_type=%s content_id=%s", contentType, contentID)

	if result := extractFromInitState(string(page)); result != nil {
		log.Printf("[kuaishou] parse ok source=init_state content_id=%s", contentID)
		return *result
	}
	if result := extractFromApolloState(string(page), contentID, contentType); result != nil {
		log.Printf("[kuaishou] parse ok source=apollo_state content_id=%s", contentID)
		return *result
	}

	log.Printf("[kuaishou] parse failed content_id=%s has_cookie=%v", contentID, req.Cookie != "")
	msg := "未找到有效媒体信息"
	if req.Cookie == "" {
		msg += "（建议传入 cookie 参数以提高成功率）"
	}
	return model.Fail(404, msg)
}

func extractContentIDAndType(u string) (string, string) {
	for typ, re := range contentPatterns {
		if m := re.FindStringSubmatch(u); len(m) > 1 {
			return typ, m[1]
		}
	}
	return "", ""
}

func extractFromInitState(page string) *model.Response {
	m := initStatePattern.FindStringSubmatch(page)
	if len(m) < 2 {
		return nil
	}

	jsonStr := strings.TrimSuffix(strings.TrimSpace(m[1]), ";")
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		jsonStr = strings.ReplaceAll(jsonStr, `"{"err_msg":"launchApplication:fail"}"`, `"err_msg","launchApplication:fail"`)
		jsonStr = strings.ReplaceAll(jsonStr, `"{"err_msg":"system:access_denied"}"`, `"err_msg","system:access_denied"`)
		if err2 := json.Unmarshal([]byte(jsonStr), &data); err2 != nil {
			return &model.Response{Code: 500, Msg: "JSON解析错误: " + err2.Error()}
		}
	}

	filtered := filterMediaData(data)
	if len(filtered) == 0 {
		return nil
	}

	var first map[string]any
	for _, v := range filtered {
		first = v
		break
	}

	photo, _ := first["photo"].(map[string]any)
	if photo == nil {
		return nil
	}

	musicInfo := extractMusicInfo(photo)

	if list, ok := photo["ext_params"].(map[string]any); ok {
		if atlas, ok := list["atlas"].(map[string]any); ok {
			if images, ok := atlas["list"].([]any); ok && len(images) > 0 {
				var urls []string
				for _, img := range images {
					if s, ok := img.(string); ok {
						urls = append(urls, "http://tx2.a.yximgs.com/"+s)
					}
				}
				musicURL := ""
				if m, ok := atlas["music"].(string); ok && m != "" {
					musicURL = "http://txmov2.a.kwimgs.com" + m
				}
				return kuaishouResponse(model.MediaTypeImage, str(photo["caption"]), str(photo["userName"]), str(photo["headUrl"]), "", urls, "", musicURL, photo["likeCount"], photo["timestamp"], nil)
			}
		}
	}

	if photoType, _ := photo["photoType"].(string); photoType == "SINGLE_PICTURE" {
		if cover, ok := photo["coverUrls"].([]any); ok && len(cover) > 0 {
			if cm, ok := cover[0].(map[string]any); ok {
				imgURL := str(cm["url"])
				if imgURL != "" {
					return kuaishouResponse(model.MediaTypeImage, str(photo["caption"]), str(photo["userName"]), str(photo["headUrl"]), imgURL, []string{imgURL}, imgURL, "", photo["likeCount"], photo["timestamp"], extractMusicInfo(photo))
				}
			}
		}
	}

	videoURL := ""
	if urls, ok := photo["mainMvUrls"].([]any); ok && len(urls) > 0 {
		if vm, ok := urls[0].(map[string]any); ok {
			videoURL = str(vm["url"])
		}
	}
	if videoURL == "" {
		if manifest, ok := photo["manifest"].(map[string]any); ok {
			if sets, ok := manifest["adaptationSet"].([]any); ok && len(sets) > 0 {
				if set, ok := sets[0].(map[string]any); ok {
					if reps, ok := set["representation"].([]any); ok && len(reps) > 0 {
						if rep, ok := reps[0].(map[string]any); ok {
							videoURL = str(rep["url"])
						}
					}
				}
			}
		}
	}

	if videoURL != "" {
		cover := ""
		if coverUrls, ok := photo["coverUrls"].([]any); ok && len(coverUrls) > 0 {
			if cm, ok := coverUrls[0].(map[string]any); ok {
				cover = str(cm["url"])
			}
		}
		return kuaishouResponse(model.MediaTypeVideo, str(photo["caption"]), str(photo["userName"]), str(photo["headUrl"]), cover, nil, videoURL, "", photo["likeCount"], photo["timestamp"], musicInfo)
	}
	return nil
}

func extractFromApolloState(page, contentID, contentType string) *model.Response {
	m := apolloStatePattern.FindStringSubmatch(page)
	if len(m) < 2 {
		return nil
	}

	cleaned := regexp.MustCompile(`function\s*\([^)]*\)\s*{[^}]*}`).ReplaceAllString(m[1], ":")
	cleaned = regexp.MustCompile(`,\s*(?=}|])`).ReplaceAllString(cleaned, "")
	cleaned = strings.ReplaceAll(cleaned, ";(:());", "")

	var apollo map[string]any
	if json.Unmarshal([]byte(cleaned), &apollo) != nil {
		return nil
	}

	videoInfo, ok := apollo["defaultClient"].(map[string]any)
	if !ok {
		return nil
	}

	key := "VisionVideoDetailPhoto:" + contentID
	videoData, ok := videoInfo[key].(map[string]any)
	if !ok {
		return nil
	}

	var authorData map[string]any
	for k, v := range videoInfo {
		if strings.HasPrefix(k, "VisionVideoDetailAuthor:") {
			authorData, _ = v.(map[string]any)
			break
		}
	}

	videoURL := ""
	if contentType == "long-video" {
		if manifest, ok := videoData["manifestH265"].(map[string]any); ok {
			if j, ok := manifest["json"].(map[string]any); ok {
				if sets, ok := j["adaptationSet"].([]any); ok && len(sets) > 0 {
					if set, ok := sets[0].(map[string]any); ok {
						if reps, ok := set["representation"].([]any); ok && len(reps) > 0 {
							if rep, ok := reps[0].(map[string]any); ok {
								if backups, ok := rep["backupUrl"].([]any); ok && len(backups) > 0 {
									videoURL = str(backups[0])
								}
							}
						}
					}
				}
			}
		}
	} else {
		videoURL = str(videoData["photoUrl"])
	}

	if videoURL == "" {
		return nil
	}

	typ := model.MediaTypeVideo
	if contentType == "photo" {
		typ = model.MediaTypeImage
	}

	return kuaishouResponse(typ, str(videoData["caption"]), str(authorData["name"]), str(authorData["headerUrl"]), str(videoData["coverUrl"]), nil, videoURL, "", nil, nil, nil)
}

func filterMediaData(data map[string]any) map[string]map[string]any {
	filtered := make(map[string]map[string]any)
	for key, value := range data {
		if !strings.HasPrefix(key, "tusjoh") {
			continue
		}
		vm, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if _, hasFID := vm["fid"]; hasFID {
			filtered[key] = vm
			continue
		}
		if _, hasPhoto := vm["photo"]; hasPhoto {
			filtered[key] = vm
		}
	}
	return filtered
}

func extractMusicInfo(photo map[string]any) map[string]string {
	var source map[string]any
	if m, ok := photo["music"].(map[string]any); ok {
		source = m
	} else if m, ok := photo["soundTrack"].(map[string]any); ok {
		source = m
	}
	if source == nil {
		return map[string]string{}
	}
	info := map[string]string{
		"name":   str(source["name"]),
		"artist": str(source["artist"]),
	}
	if imgs, ok := source["imageUrls"].([]any); ok && len(imgs) > 0 {
		if im, ok := imgs[0].(map[string]any); ok {
			info["cover"] = str(im["url"])
		}
	}
	if audios, ok := source["audioUrls"].([]any); ok && len(audios) > 0 {
		if am, ok := audios[0].(map[string]any); ok {
			info["url"] = str(am["url"])
		}
	}
	return info
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

func kuaishouResponse(mediaType, title, authorName, avatar, cover string, images []string, mainURL, musicURL string, likeCount, publishedAt any, music map[string]string) *model.Response {
	data := model.NewVideoData(model.PlatformKuaishou, mediaType)
	data.Title = title
	data.Cover = cover
	data.URL = mainURL
	data.Images = images
	data.Author = model.AuthorOf(authorName, "", avatar)
	if musicURL != "" {
		data.Music = model.MusicOf("", "", musicURL, "")
	} else if len(music) > 0 {
		data.Music = model.MusicOf(music["name"], music["artist"], music["url"], music["cover"])
	}
	if likeCount != nil || publishedAt != nil {
		data.Stats = &model.Stats{LikeCount: likeCount, PublishedAt: publishedAt}
	}
	return &model.Response{Code: 200, Msg: "解析成功", Data: data}
}
