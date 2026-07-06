package httputil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	finalURL := url
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	if resp.StatusCode >= 400 && !opts.NoRedirect {
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

// GetFinalURL 获取最终URL
func (c *Client) GetFinalURL(ctx context.Context, url string) (string, error) {
	_, finalURL, err := c.Do(ctx, url, RequestOptions{
		Method:     http.MethodHead,
		NoRedirect: false,
		Headers: map[string]string{
			"User-Agent": DefaultUserAgent,
		},
	})
	if err != nil {
		_, finalURL, err = c.Do(ctx, url, RequestOptions{
			Method: http.MethodGet,
			Headers: map[string]string{
				"User-Agent": DefaultUserAgent,
			},
		})
	}
	return finalURL, err
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
