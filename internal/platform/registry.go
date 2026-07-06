package platform

import (
	"net/url"
	"strings"

	"fuck-watermark/internal/config"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/parser"
)

// Descriptor 描述一个可注册的平台解析器。
type Descriptor struct {
	Name         string                                // 平台名称
	Keywords     []string                              // 关键词
	HostSuffixes []string                              // 域名后缀
	Aliases      []string                              // 别名
	Factory      func(deps Dependencies) parser.Parser // 创建解析器
}

// Dependencies 创建 Parser 时的共享依赖。
type Dependencies struct {
	Client *httputil.Client
	Config config.Config
}

// Registry 管理平台解析器与路由别名。
type Registry struct {
	platforms []parser.Platform
	byName    map[string]parser.Platform
	aliases   map[string]string
}

func NewRegistry(descriptors []Descriptor, deps Dependencies) *Registry {
	reg := &Registry{
		byName:  make(map[string]parser.Platform, len(descriptors)),
		aliases: make(map[string]string),
	}

	for _, desc := range descriptors {
		p := parser.Platform{
			Name:         desc.Name,
			Keywords:     desc.Keywords,
			HostSuffixes: desc.HostSuffixes,
			Parser:       desc.Factory(deps),
		}
		reg.platforms = append(reg.platforms, p)
		reg.byName[desc.Name] = p
		for _, alias := range desc.Aliases {
			reg.aliases[alias] = desc.Name
		}
	}
	return reg
}

func (r *Registry) All() []parser.Platform {
	return r.platforms
}

func (r *Registry) Get(name string) (parser.Platform, bool) {
	if canonical, ok := r.aliases[name]; ok {
		name = canonical
	}
	p, ok := r.byName[name]
	return p, ok
}

func (r *Registry) RouteNames() []string {
	names := make([]string, 0, len(r.platforms)+len(r.aliases))
	seen := make(map[string]struct{}, len(r.platforms)+len(r.aliases))
	for _, p := range r.platforms {
		if _, ok := seen[p.Name]; !ok {
			names = append(names, p.Name)
			seen[p.Name] = struct{}{}
		}
	}
	for alias := range r.aliases {
		if _, ok := seen[alias]; !ok {
			names = append(names, alias)
			seen[alias] = struct{}{}
		}
	}
	return names
}

// Match 根据 URL 的 host 与关键词匹配平台，host 优先于 substring。
func (r *Registry) Match(rawURL string) (parser.Platform, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return parser.Platform{}, false
	}
	host := strings.ToLower(u.Hostname())
	lower := strings.ToLower(rawURL)

	for _, p := range r.platforms {
		for _, suffix := range p.HostSuffixes {
			suffix = strings.ToLower(suffix)
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return p, true
			}
		}
	}

	for _, p := range r.platforms {
		for _, kw := range p.Keywords {
			if strings.Contains(host, kw) || strings.Contains(lower, kw) {
				return p, true
			}
		}
	}
	return parser.Platform{}, false
}
