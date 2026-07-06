package server

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"short_videos/internal/config"
	"short_videos/internal/handler"
	"short_videos/internal/httputil"
	"short_videos/internal/middleware"
	"short_videos/internal/platform"
)

type Server struct {
	engine *gin.Engine
	cfg    config.Config
}

func New(cfg config.Config) (*Server, error) {
	client := httputil.New(cfg.RequestTimeout)
	deps := platform.Dependencies{Client: client, Config: cfg}
	registry := platform.NewRegistry(platform.DefaultDescriptors(), deps)
	h := handler.New(registry)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.Security.AllowOrigins))
	r.Use(middleware.MaxBodySize(cfg.Security.MaxBodyBytes))

	r.GET("/health", h.Health)
	r.GET("/", h.Health)

	api := r.Group("/api")
	if cfg.RateLimit.Enabled {
		api.Use(middleware.RateLimit(cfg.RateLimit))
	}
	if cfg.Security.APIKey != "" {
		api.Use(middleware.APIKeyAuth(cfg.Security.APIKey))
	}
	api.Use(gin.Logger())

	api.GET("/parse", h.ParseAuto)
	api.POST("/parse", h.ParseAuto)
	for _, routeName := range registry.RouteNames() {
		api.GET("/"+routeName, h.ParsePlatform(routeName))
		api.POST("/"+routeName, h.ParsePlatform(routeName))
	}

	return &Server{engine: r, cfg: cfg}, nil
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.Addr)
	log.Printf("API已启动，监听地址：%v", s.cfg.Addr)
	if err := s.engine.Run(addr); err != nil {
		return fmt.Errorf("服务启动失败: %w", err)
	}
	return nil
}
