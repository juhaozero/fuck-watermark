package main

import (
	"flag"
	"fmt"
	"os"

	"fuck-watermark/internal/config"
	"fuck-watermark/internal/server"
	"fuck-watermark/logs"
)

func main() {
	configPath := flag.String("c", "config.toml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	if err := logs.Init(cfg.Log); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logs.Close()

	srv, err := server.New(cfg)
	if err != nil {
		logs.Fatalf("创建服务失败: %v", err)
	}
	if err := srv.Run(); err != nil {
		logs.Fatalf("服务运行失败: %v", err)
	}
}
