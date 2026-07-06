package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"short_videos/internal/model"
	"short_videos/internal/parser"
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

func (h *Handler) ParseAuto(c *gin.Context) {
	req, err := readParseRequest(c)
	if err != nil {
		c.JSON(http.StatusOK, model.Fail(400, err.Error()))
		return
	}

	p, ok := h.registry.Match(req.URL)
	if !ok {
		c.JSON(http.StatusOK, model.Fail(400, "暂不支持该平台链接"))
		return
	}

	c.JSON(http.StatusOK, p.Parser.Parse(c.Request.Context(), req))
}

func (h *Handler) ParsePlatform(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := readParseRequest(c)
		if err != nil {
			c.JSON(http.StatusOK, model.Fail(400, err.Error()))
			return
		}

		p, ok := h.registry.Get(name)
		if !ok {
			c.JSON(http.StatusOK, model.Fail(404, "平台未配置"))
			return
		}

		c.JSON(http.StatusOK, p.Parser.Parse(c.Request.Context(), req))
	}
}

type parseInput struct {
	URL    string
	Cookie string
}

func readParseRequest(c *gin.Context) (parser.Request, error) {
	in := extractInput(c)
	if err := urlutil.ValidateParseURL(in.URL); err != nil {
		return parser.Request{}, err
	}
	return parser.Request{URL: in.URL, Cookie: in.Cookie}, nil
}

func extractInput(c *gin.Context) parseInput {
	var in parseInput

	if u := c.Query("url"); u != "" {
		in.URL = u
	}
	if ck := c.Query("cookie"); ck != "" {
		in.Cookie = ck
	}

	if in.URL == "" {
		in.URL = c.PostForm("url")
	}
	if in.Cookie == "" {
		in.Cookie = c.PostForm("cookie")
	}

	if in.URL == "" || in.Cookie == "" {
		var body struct {
			URL    string `json:"url"`
			Cookie string `json:"cookie"`
		}
		if c.ShouldBindJSON(&body) == nil {
			if in.URL == "" {
				in.URL = body.URL
			}
			if in.Cookie == "" {
				in.Cookie = body.Cookie
			}
		}
	}

	return in
}
