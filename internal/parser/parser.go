package parser

import (
	"context"

	"short_videos/internal/model"
)

// Request 单次解析请求参数。
type Request struct {
	URL    string
	Cookie string
}

type Parser interface {
	Parse(ctx context.Context, req Request) model.Response
}

type Platform struct {
	Name         string
	Keywords     []string
	HostSuffixes []string
	Parser       Parser
}
