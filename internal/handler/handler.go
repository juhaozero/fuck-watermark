package handler

import (
	"net/http"

	"fuck-watermark/logs"

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
		logs.Warnf("[解析] 自动识别失败 客户端=%s 错误=%v", c.ClientIP(), err)
		c.JSON(http.StatusOK, model.Fail(400, err.Error()))
		return
	}

	p, ok := h.registry.Match(req.URL)
	if !ok {
		logs.Warnf("[解析] 自动识别失败 平台=未知 链接=%q 客户端=%s 错误=暂不支持该平台链接", req.URL, c.ClientIP())
		c.JSON(http.StatusOK, model.Fail(400, "暂不支持该平台链接"))
		return
	}

	logs.Infof("[解析] 自动识别 平台=%s 链接=%q 客户端=%s", p.Name, req.URL, c.ClientIP())
	resp := p.Parser.Parse(c.Request.Context(), req)
	logs.Infof("[解析] 自动识别完成 平台=%s 状态码=%d 消息=%q", p.Name, resp.Code, resp.Msg)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ParsePlatform(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := readParseRequest(c)
		if err != nil {
			logs.Warnf("[解析] 请求无效 平台=%s 客户端=%s 错误=%v", name, c.ClientIP(), err)
			c.JSON(http.StatusOK, model.Fail(400, err.Error()))
			return
		}

		p, ok := h.registry.Get(name)
		if !ok {
			logs.Warnf("[解析] 平台未配置 平台=%s 链接=%q 客户端=%s", name, req.URL, c.ClientIP())
			c.JSON(http.StatusOK, model.Fail(404, "平台未配置"))
			return
		}

		logs.Infof("[解析] 开始 平台=%s 链接=%q 客户端=%s", name, req.URL, c.ClientIP())
		resp := p.Parser.Parse(c.Request.Context(), req)
		logs.Infof("[解析] 完成 平台=%s 状态码=%d 消息=%q", name, resp.Code, resp.Msg)
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
