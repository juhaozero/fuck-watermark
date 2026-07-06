package urlutil

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var blockedHosts = map[string]struct{}{
	"localhost": {},
}

// ValidateParseURL 校验用户提交的解析链接，降低 SSRF 风险。
func ValidateParseURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("url 不能为空")
	}

	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("无效的链接格式")
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("仅支持 http/https 链接")
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("无效的链接 host")
	}

	if _, blocked := blockedHosts[host]; blocked {
		return fmt.Errorf("不允许访问该 host")
	}

	if err := validateHostIP(host); err != nil {
		return err
	}

	return nil
}

func validateHostIP(host string) error {
	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("不允许访问内网地址")
	}
	if ip.String() == "169.254.169.254" {
		return fmt.Errorf("不允许访问 metadata 地址")
	}
	return nil
}
