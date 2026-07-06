package platform

import (
	"fuck-watermark/internal/parser"

	"fuck-watermark/internal/parser/bilibili"

	"fuck-watermark/internal/parser/doubao"

	"fuck-watermark/internal/parser/douyin"

	"fuck-watermark/internal/parser/kuaishou"

	"fuck-watermark/internal/parser/toutiao"

	"fuck-watermark/internal/parser/weibo"

	"fuck-watermark/internal/parser/xiaohongshu"
)

func DefaultDescriptors() []Descriptor {

	return []Descriptor{

		{

			Name: "douyin",

			Keywords: []string{"douyin"},

			HostSuffixes: []string{"douyin.com", "iesdouyin.com"},

			Factory: func(d Dependencies) parser.Parser {

				return douyin.New(d.Client)

			},
		},

		{

			Name: "kuaishou",

			Keywords: []string{"kuaishou", "chenzhongtech", "kspkg"},

			HostSuffixes: []string{"kuaishou.com", "chenzhongtech.com", "yximgs.com", "kwimgs.com"},

			Factory: func(d Dependencies) parser.Parser {

				return kuaishou.New(d.Client)

			},
		},

		{

			Name: "xiaohongshu",

			Keywords: []string{"xhs", "xiaohongshu", "xhslink"},

			HostSuffixes: []string{"xiaohongshu.com", "xhslink.com", "xhscdn.com"},

			Aliases: []string{"xhsjx"},

			Factory: func(d Dependencies) parser.Parser {

				return xiaohongshu.New(d.Client)

			},
		},

		{

			Name: "bilibili",

			Keywords: []string{"bilibili", "b23.tv"},

			HostSuffixes: []string{"bilibili.com", "b23.tv", "bilivideo.com"},

			Factory: func(d Dependencies) parser.Parser {

				return bilibili.New(d.Client)

			},
		},

		{

			Name: "toutiao",

			Keywords: []string{"toutiao"},

			HostSuffixes: []string{"toutiao.com"},

			Factory: func(d Dependencies) parser.Parser {

				return toutiao.New(d.Client)

			},
		},

		{

			Name: "weibo",

			Keywords: []string{"weibo", "t.cn"},

			HostSuffixes: []string{"weibo.com", "weibo.cn", "t.cn"},

			Factory: func(d Dependencies) parser.Parser {

				return weibo.New(d.Client, d.Config.WeiboProxyBase)

			},
		},

		{

			Name: "doubao",

			Keywords: []string{"doubao"},

			HostSuffixes: []string{"doubao.com"},

			Factory: func(d Dependencies) parser.Parser {

				return doubao.New(d.Client)

			},
		},
	}

}
