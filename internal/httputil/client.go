package httputil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fuck-watermark/logs"
)

const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type Client struct {
	http *http.Client
}

func New(timeout time.Duration) *Client {
	return &Client{
		http: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

type RequestOptions struct {
	Method     string
	Headers    map[string]string
	Cookie     string
	Body       io.Reader
	NoRedirect bool
}

func (c *Client) Do(ctx context.Context, url string, opts RequestOptions) ([]byte, string, error) {
	method := opts.Method
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, url, opts.Body)
	if err != nil {
		return nil, "", err
	}

	ua := DefaultUserAgent
	if opts.Headers != nil {
		for k, v := range opts.Headers {
			if strings.EqualFold(k, "User-Agent") {
				ua = v
			}
			req.Header.Set(k, v)
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", ua)
	}
	if opts.Cookie != "" {
		req.Header.Set("Cookie", opts.Cookie)
	}

	client := c.http
	if opts.NoRedirect {
		client = &http.Client{
			Timeout: c.http.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		logs.Warnf("[HTTP] 请求失败 方法=%s 链接=%q 错误=%v", method, url, err)
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Warnf("[HTTP] 读取响应失败 方法=%s 链接=%q 状态码=%d 错误=%v", method, url, resp.StatusCode, err)
		return nil, "", err
	}

	finalURL := url
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	if resp.StatusCode >= 400 && !opts.NoRedirect {
		logs.Warnf("[HTTP] 异常状态码 方法=%s 链接=%q 状态码=%d 最终链接=%q 正文=%q", method, url, resp.StatusCode, finalURL, truncateBody(body))
		return body, finalURL, fmt.Errorf("http status %d", resp.StatusCode)
	}

	return body, finalURL, nil
}

func (c *Client) Get(ctx context.Context, url string, cookie string, headers map[string]string) ([]byte, error) {
	body, _, err := c.Do(ctx, url, RequestOptions{
		Method:  http.MethodGet,
		Cookie:  cookie,
		Headers: headers,
	})
	return body, err
}

// Open 发起 GET 并返回响应流，调用方须关闭 resp.Body。
func (c *Client) Open(ctx context.Context, rawURL string, cookie string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ua := DefaultUserAgent
	for k, v := range headers {
		if strings.EqualFold(k, "User-Agent") {
			ua = v
		}
		req.Header.Set(k, v)
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", ua)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		logs.Warnf("[HTTP] 打开资源失败 链接=%q 错误=%v", rawURL, err)
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		logs.Warnf("[HTTP] 打开资源状态异常 链接=%q 状态码=%d", rawURL, resp.StatusCode)
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}
	return resp, nil
}

// GetFinalURL 跟随重定向链，返回最终 URL。
// 部分站点（如 B 站）落地页会返回 412/403 等风控状态，但重定向 URL 仍然有效。
func (c *Client) GetFinalURL(ctx context.Context, url string, headers ...map[string]string) (string, error) {
	h := map[string]string{"User-Agent": DefaultUserAgent}
	if len(headers) > 0 && headers[0] != nil {
		for k, v := range headers[0] {
			h[k] = v
		}
	}

	_, finalURL, err := c.Do(ctx, url, RequestOptions{
		Method:  http.MethodHead,
		Headers: h,
	})
	if err != nil && !redirectResolved(url, finalURL) {
		logs.Warnf("[HTTP] HEAD跳转失败 链接=%q 错误=%v，改用GET", url, err)
		_, finalURL, err = c.Do(ctx, url, RequestOptions{
			Method:  http.MethodGet,
			Headers: h,
		})
	}
	if err != nil && redirectResolved(url, finalURL) {
		logs.Infof("[HTTP] 最终地址已解析(忽略错误) 原始=%q 最终=%q 错误=%v", url, finalURL, err)
		return finalURL, nil
	}
	if err != nil {
		logs.Warnf("[HTTP] 获取最终地址失败 链接=%q 错误=%v", url, err)
		return finalURL, err
	}
	logs.Infof("[HTTP] 最终地址已解析 原始=%q 最终=%q", url, finalURL)
	return finalURL, nil
}

func redirectResolved(original, final string) bool {
	return final != "" && final != original
}

func (c *Client) Post(ctx context.Context, url string, body io.Reader, cookie string, headers map[string]string) ([]byte, error) {
	data, _, err := c.Do(ctx, url, RequestOptions{
		Method:  http.MethodPost,
		Body:    body,
		Cookie:  cookie,
		Headers: headers,
	})
	return data, err
}

// HeadRedirect 获取单次重定向的 Location，不跟随跳转。
func (c *Client) HeadRedirect(ctx context.Context, rawURL string, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", DefaultUserAgent)
	}

	client := &http.Client{
		Timeout: c.http.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if loc := resp.Header.Get("Location"); loc != "" {
			return loc, nil
		}
	}
	if resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String(), nil
	}
	return rawURL, nil
}

func truncateBody(body []byte) string {
	const max = 256
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "..."
}
