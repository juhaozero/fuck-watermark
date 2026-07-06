package platform

import (
	"context"
	"testing"
	"time"

	"short_videos/internal/config"
	"short_videos/internal/httputil"
	"short_videos/internal/model"
	"short_videos/internal/parser"
)

type stubParser struct {
	name string
}

func (s stubParser) Parse(_ context.Context, _ string) model.Response {
	return model.OK("ok", model.NewVideoData(s.name, model.MediaTypeVideo))
}

func TestRegistryMatch(t *testing.T) {
	descriptors := []Descriptor{
		{
			Name:         "douyin",
			Keywords:     []string{"douyin"},
			HostSuffixes: []string{"douyin.com"},
			Factory: func(_ Dependencies) parser.Parser {
				return stubParser{name: "douyin"}
			},
		},
		{
			Name:         "bilibili",
			Keywords:     []string{"bilibili", "b23.tv"},
			HostSuffixes: []string{"bilibili.com", "b23.tv"},
			Factory: func(_ Dependencies) parser.Parser {
				return stubParser{name: "bilibili"}
			},
		},
	}

	deps := Dependencies{
		Client: httputil.New(5 * time.Second),
		Config: config.Config{},
	}
	reg := NewRegistry(descriptors, deps)

	tests := []struct {
		url      string
		wantName string
	}{
		{url: "https://v.douyin.com/abc/", wantName: "douyin"},
		{url: "https://www.bilibili.com/video/BV1", wantName: "bilibili"},
		{url: "https://b23.tv/BV1", wantName: "bilibili"},
	}

	for _, tt := range tests {
		p, ok := reg.Match(tt.url)
		if !ok {
			t.Fatalf("Match(%q) = false", tt.url)
		}
		if p.Name != tt.wantName {
			t.Fatalf("Match(%q) = %q, want %q", tt.url, p.Name, tt.wantName)
		}
	}
}

func TestRegistryAlias(t *testing.T) {
	descriptors := []Descriptor{
		{
			Name:     "xiaohongshu",
			Keywords: []string{"xiaohongshu"},
			Aliases:  []string{"xhsjx"},
			Factory: func(_ Dependencies) parser.Parser {
				return stubParser{name: "xiaohongshu"}
			},
		},
	}
	reg := NewRegistry(descriptors, Dependencies{Client: httputil.New(time.Second), Config: config.Config{}})

	p, ok := reg.Get("xhsjx")
	if !ok || p.Name != "xiaohongshu" {
		t.Fatalf("Get(xhsjx) = %#v, ok=%v", p, ok)
	}
}
