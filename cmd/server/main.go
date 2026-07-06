package main

import (
	"flag"
	"log"

	"short_videos/internal/config"
	"short_videos/internal/server"
)

func main() {
	configPath := flag.String("c", "config.toml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
