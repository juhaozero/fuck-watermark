package server

import (
	"fmt"

	"fuck-watermark/logs"

	"github.com/gin-gonic/gin"

	"fuck-watermark/internal/config"
	"fuck-watermark/internal/handler"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/middleware"
	"fuck-watermark/internal/platform"
)

type Server struct {
	engine *gin.Engine
	cfg    config.Config
}

func New(cfg config.Config) (*Server, error) {
	client := httputil.New(cfg.RequestTimeout)
	deps := platform.Dependencies{Client: client, Config: cfg}
	registry := platform.NewRegistry(platform.DefaultDescriptors(), deps)
	h := handler.New(registry, client)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.Security.AllowOrigins))
	r.Use(middleware.MaxBodySize(cfg.Security.MaxBodyBytes))

	r.GET("/health", h.Health)
	api := r.Group("/api")
	if cfg.RateLimit.Enabled {
		api.Use(middleware.RateLimit(cfg.RateLimit))
	}
	if cfg.Security.APIKey != "" {
		api.Use(middleware.APIKeyAuth(cfg.Security.APIKey))
	}
	api.Use(middleware.RequestLogger())

	api.GET("/parse", h.ParseAuto)
	api.POST("/parse", h.ParseAuto)
	// api.GET("/download", h.Download)
	// api.HEAD("/download", h.Download)
	for _, routeName := range registry.RouteNames() {
		api.GET("/"+routeName, h.ParsePlatform(routeName))
		api.POST("/"+routeName, h.ParsePlatform(routeName))
	}

	return &Server{engine: r, cfg: cfg}, nil
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.Addr)
	logs.Infof("API已启动，监听地址：%v", s.cfg.Addr)
	if err := s.engine.Run(addr); err != nil {
		return fmt.Errorf("服务启动失败: %w", err)
	}
	return nil
}
