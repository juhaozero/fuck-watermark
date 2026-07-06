package parser

import (
	"context"

	"short_videos/internal/model"
)

type Parser interface {
	Parse(ctx context.Context, url string) model.Response
}

type Platform struct {
	Name         string
	Keywords     []string
	HostSuffixes []string
	Parser       Parser
}
