package platform

import (
	"short_videos/internal/parser"
	"short_videos/internal/parser/bilibili"
	"short_videos/internal/parser/doubao"
	"short_videos/internal/parser/douyin"
	"short_videos/internal/parser/kuaishou"
	"short_videos/internal/parser/toutiao"
	"short_videos/internal/parser/weibo"
	"short_videos/internal/parser/xiaohongshu"
)

// DefaultDescriptors 内置平台列表；新增平台时在此追加 Descriptor 即可。
func DefaultDescriptors() []Descriptor {
	return []Descriptor{
		{
			Name:         "douyin",
			Keywords:     []string{"douyin"},
			HostSuffixes: []string{"douyin.com", "iesdouyin.com"},
			Factory: func(d Dependencies) parser.Parser {
				return douyin.New(d.Client, d.Config.Cookie("douyin"))
			},
		},
		{
			Name:         "kuaishou",
			Keywords:     []string{"kuaishou", "chenzhongtech", "kspkg"},
			HostSuffixes: []string{"kuaishou.com", "chenzhongtech.com", "yximgs.com", "kwimgs.com"},
			Factory: func(d Dependencies) parser.Parser {
				return kuaishou.New(d.Client, d.Config.Cookie("kuaishou"))
			},
		},
		{
			Name:         "xiaohongshu",
			Keywords:     []string{"xhs", "xiaohongshu", "xhslink"},
			HostSuffixes: []string{"xiaohongshu.com", "xhslink.com", "xhscdn.com"},
			Aliases:      []string{"xhsjx"},
			Factory: func(d Dependencies) parser.Parser {
				return xiaohongshu.New(d.Client, d.Config.Cookie("xiaohongshu"))
			},
		},
		{
			Name:         "bilibili",
			Keywords:     []string{"bilibili", "b23.tv"},
			HostSuffixes: []string{"bilibili.com", "b23.tv", "bilivideo.com"},
			Factory: func(d Dependencies) parser.Parser {
				return bilibili.New(d.Client, d.Config.Cookie("bilibili"))
			},
		},
		{
			Name:         "toutiao",
			Keywords:     []string{"toutiao"},
			HostSuffixes: []string{"toutiao.com"},
			Factory: func(d Dependencies) parser.Parser {
				return toutiao.New(d.Client, d.Config.Cookie("toutiao"))
			},
		},
		{
			Name:         "weibo",
			Keywords:     []string{"weibo", "t.cn"},
			HostSuffixes: []string{"weibo.com", "weibo.cn", "t.cn"},
			Factory: func(d Dependencies) parser.Parser {
				return weibo.New(d.Client, d.Config.Cookie("weibo"), d.Config.WeiboProxyBase)
			},
		},
		{
			Name:         "doubao",
			Keywords:     []string{"doubao"},
			HostSuffixes: []string{"doubao.com"},
			Factory: func(d Dependencies) parser.Parser {
				return doubao.New(d.Client, d.Config.Cookie("doubao"))
			},
		},
	}
}
