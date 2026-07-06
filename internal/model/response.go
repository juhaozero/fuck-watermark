package model

// MediaType 媒体类型常量。
const (
	MediaTypeVideo   = "video"
	MediaTypeImage   = "image"
	MediaTypeLive    = "live"
	MediaTypeMixed   = "mixed"
	MediaTypeUnknown = "unknown"
)

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

type Author struct {
	Name   string `json:"name,omitempty"`
	ID     string `json:"id,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

type Music struct {
	Title  string `json:"title,omitempty"`
	Author string `json:"author,omitempty"`
	URL    string `json:"url,omitempty"`
	Cover  string `json:"cover,omitempty"`
}

type LivePhoto struct {
	Image string `json:"image"`
	Video string `json:"video"`
}

// VideoBackup 备用清晰度/线路。
type VideoBackup struct {
	URL     string `json:"url"`
	Label   string `json:"label,omitempty"`
	Quality string `json:"quality,omitempty"`
}

// VideoPart 分 P 视频（如 B 站多 P）。
type VideoPart struct {
	Title          string `json:"title,omitempty"`
	URL            string `json:"url,omitempty"`
	Duration       int    `json:"duration,omitempty"`
	DurationFormat string `json:"duration_format,omitempty"`
	Index          int    `json:"index,omitempty"`
}

// Stats 平台侧互动数据（可选）。
type Stats struct {
	LikeCount     any `json:"like_count,omitempty"`
	PlayCount     any `json:"play_count,omitempty"`
	RepostCount   any `json:"repost_count,omitempty"`
	CommentCount  any `json:"comment_count,omitempty"`
	AttitudeCount any `json:"attitude_count,omitempty"`
	PublishedAt   any `json:"published_at,omitempty"`
	IPInfo        any `json:"ip_info,omitempty"`
}

// VideoData 统一解析结果 schema，所有平台成功响应均使用此结构。
type VideoData struct {
	Platform    string        `json:"platform"`               // 平台名称
	Type        string        `json:"type"`                   // 媒体类型
	Title       string        `json:"title,omitempty"`        // 标题
	Desc        string        `json:"desc,omitempty"`         // 描述
	Author      *Author       `json:"author,omitempty"`       // 作者
	Cover       string        `json:"cover,omitempty"`        // 封面
	URL         string        `json:"url,omitempty"`          // 视频URL
	Duration    any           `json:"duration,omitempty"`     // 视频时长
	Quality     string        `json:"quality,omitempty"`      // 视频质量
	VideoBackup []VideoBackup `json:"video_backup,omitempty"` // 备用清晰度/线路
	VideoID     string        `json:"video_id,omitempty"`     // 视频ID
	Images      []string      `json:"images,omitempty"`       // 图片
	LivePhoto   []LivePhoto   `json:"live_photo,omitempty"`   // 直播照片
	Music       *Music        `json:"music,omitempty"`        // 音乐
	Parts       []VideoPart   `json:"parts,omitempty"`        // 分P视频
	Stats       *Stats        `json:"stats,omitempty"`        // 平台侧互动数据
}

func NewVideoData(platform, mediaType string) *VideoData {
	return &VideoData{
		Platform: platform,
		Type:     mediaType,
	}
}

func OK(msg string, data *VideoData) Response {
	return Response{Code: 200, Msg: msg, Data: data}
}

func Fail(code int, msg string) Response {
	return Response{Code: code, Msg: msg}
}

// BackupsFromURLs 从URL列表创建备用清晰度/线路列表
func BackupsFromURLs(urls ...string) []VideoBackup {
	out := make([]VideoBackup, 0, len(urls))
	for _, u := range urls {
		if u != "" {
			out = append(out, VideoBackup{URL: u})
		}
	}
	return out
}

func AuthorOf(name, id, avatar string) *Author {
	if name == "" && id == "" && avatar == "" {
		return nil
	}
	return &Author{Name: name, ID: id, Avatar: avatar}
}

func MusicOf(title, author, url, cover string) *Music {
	if title == "" && author == "" && url == "" && cover == "" {
		return nil
	}
	return &Music{Title: title, Author: author, URL: url, Cover: cover}
}
