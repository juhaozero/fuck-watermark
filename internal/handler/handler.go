package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/model"
	"fuck-watermark/internal/parser"
	"fuck-watermark/internal/platform"
	"fuck-watermark/internal/urlutil"
)

type Handler struct {
	registry *platform.Registry
	client   *httputil.Client
}

func New(registry *platform.Registry, client *httputil.Client) *Handler {
	return &Handler{registry: registry, client: client}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "fuck-watermark",
	})
}

func (h *Handler) ParseAuto(c *gin.Context) {
	req, err := readParseRequest(c)
	if err != nil {
		log.Printf("[parse] auto client=%s err=%v", c.ClientIP(), err)
		c.JSON(http.StatusOK, model.Fail(400, err.Error()))
		return
	}

	p, ok := h.registry.Match(req.URL)
	if !ok {
		log.Printf("[parse] auto platform=unknown url=%q client=%s err=暂不支持该平台链接", req.URL, c.ClientIP())
		c.JSON(http.StatusOK, model.Fail(400, "暂不支持该平台链接"))
		return
	}

	log.Printf("[parse] auto platform=%s url=%q client=%s", p.Name, req.URL, c.ClientIP())
	resp := p.Parser.Parse(c.Request.Context(), req)
	log.Printf("[parse] auto platform=%s code=%d msg=%q", p.Name, resp.Code, resp.Msg)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ParsePlatform(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := readParseRequest(c)
		if err != nil {
			log.Printf("[parse] platform=%s client=%s err=%v", name, c.ClientIP(), err)
			c.JSON(http.StatusOK, model.Fail(400, err.Error()))
			return
		}

		p, ok := h.registry.Get(name)
		if !ok {
			log.Printf("[parse] platform=%s url=%q client=%s err=平台未配置", name, req.URL, c.ClientIP())
			c.JSON(http.StatusOK, model.Fail(404, "平台未配置"))
			return
		}

		log.Printf("[parse] platform=%s url=%q client=%s", name, req.URL, c.ClientIP())
		resp := p.Parser.Parse(c.Request.Context(), req)
		log.Printf("[parse] platform=%s code=%d msg=%q", name, resp.Code, resp.Msg)
		c.JSON(http.StatusOK, resp)
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
