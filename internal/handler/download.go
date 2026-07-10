package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	"fuck-watermark/internal/model"
	"fuck-watermark/internal/urlutil"
)

const maxDownloadBytes = 512 << 20 // 512MB

// Download 下载文件
func (h *Handler) Download(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	if err := urlutil.ValidateParseURL(rawURL); err != nil {
		log.Printf("[download] invalid url client=%s err=%v", c.ClientIP(), err)
		c.JSON(http.StatusOK, model.Fail(400, err.Error()))
		return
	}

	filename := sanitizeFilename(c.Query("filename"))
	if filename == "download" {
		filename = filenameFromURL(rawURL)
	}

	cookie := strings.TrimSpace(c.Query("cookie"))
	referer := strings.TrimSpace(c.Query("referer"))
	if referer == "" {
		referer = defaultReferer(rawURL)
	}

	headers := map[string]string{}
	if referer != "" {
		headers["Referer"] = referer
	}

	log.Printf("[download] start url=%q filename=%q client=%s", rawURL, filename, c.ClientIP())

	resp, err := h.client.Open(c.Request.Context(), rawURL, cookie, headers)
	if err != nil {
		log.Printf("[download] upstream failed url=%q err=%v", rawURL, err)
		c.JSON(http.StatusOK, model.Fail(502, "拉取文件失败"))
		return
	}

	defer resp.Body.Close()

	if resp.ContentLength > maxDownloadBytes {
		log.Printf("[download] file too large url=%q size=%d", rawURL, resp.ContentLength)
		c.JSON(http.StatusOK, model.Fail(413, "文件过大"))
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", contentDisposition(filename))
	if resp.ContentLength > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}

	limited := &io.LimitedReader{R: resp.Body, N: maxDownloadBytes + 1}
	written, err := io.Copy(c.Writer, limited)
	if err != nil {
		log.Printf("[download] stream failed url=%q written=%d err=%v", rawURL, written, err)
		return
	}
	if limited.N == 0 {
		log.Printf("[download] exceeded max size url=%q written=%d", rawURL, written)
		return
	}

	log.Printf("[download] ok url=%q filename=%q bytes=%d", rawURL, filename, written)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "download"
	}
	name = path.Base(name)
	var b strings.Builder
	for _, r := range name {
		if r == '/' || r == '\\' || r == '"' || r == '\n' || r == '\r' {
			continue
		}
		b.WriteRune(r)
	}
	name = strings.TrimSpace(b.String())
	if name == "" || name == "." || name == ".." {
		return "download"
	}
	return name
}

func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "download"
	}
	base := path.Base(u.Path)
	if idx := strings.Index(base, "?"); idx >= 0 {
		base = base[:idx]
	}
	return sanitizeFilename(base)
}

func contentDisposition(filename string) string {
	ascii := toASCIIFilename(filename)
	if ascii == filename {
		return fmt.Sprintf(`attachment; filename="%s"`, ascii)
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(filename))
}

func toASCIIFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r > unicode.MaxASCII || r == '"' {
			b.WriteRune('_')
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" {
		return "download"
	}
	return out
}

func defaultReferer(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case strings.Contains(host, "bilivideo.com"), strings.Contains(host, "bilibili.com"):
		return "https://www.bilibili.com"
	case strings.Contains(host, "douyin.com"), strings.Contains(host, "douyinvod.com"):
		return "https://www.douyin.com"
	case strings.Contains(host, "kuaishou.com"), strings.Contains(host, "ksapisrv.com"):
		return "https://www.kuaishou.com"
	default:
		return ""
	}
}
