package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"short_videos/internal/model"
	"short_videos/internal/platform"
	"short_videos/internal/urlutil"
)

type Handler struct {
	registry *platform.Registry
}

func New(registry *platform.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "short_videos",
	})
}

// ParseAuto 自动解析链接
func (h *Handler) ParseAuto(c *gin.Context) {
	rawURL, err := readURLParam(c)
	if err != nil {
		c.JSON(http.StatusOK, model.Fail(400, err.Error()))
		return
	}

	p, ok := h.registry.Match(rawURL)
	if !ok {
		c.JSON(http.StatusOK, model.Fail(400, "暂不支持该平台链接"))
		return
	}

	c.JSON(http.StatusOK, p.Parser.Parse(c.Request.Context(), rawURL))
}

// ParsePlatform 解析指定平台链接
func (h *Handler) ParsePlatform(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawURL, err := readURLParam(c)
		if err != nil {
			c.JSON(http.StatusOK, model.Fail(400, err.Error()))
			return
		}

		p, ok := h.registry.Get(name)
		if !ok {
			c.JSON(http.StatusOK, model.Fail(404, "平台未配置"))
			return
		}

		c.JSON(http.StatusOK, p.Parser.Parse(c.Request.Context(), rawURL))
	}
}

// readURLParam 读取URL参数，并进行校验
func readURLParam(c *gin.Context) (string, error) {
	rawURL := extractURLParam(c)
	if err := urlutil.ValidateParseURL(rawURL); err != nil {
		return "", err
	}
	return rawURL, nil
}

// extractURLParam 提取URL参数，支持GET、POST、JSON三种方式
func extractURLParam(c *gin.Context) string {
	if u := c.Query("url"); u != "" {
		return u
	}
	if u := c.PostForm("url"); u != "" {
		return u
	}
	var body struct {
		URL string `json:"url"`
	}
	if c.ShouldBindJSON(&body) == nil && body.URL != "" {
		return body.URL
	}
	return ""
}
