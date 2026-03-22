package main

import (
	"flag"
	"log"

	"go-judge-system/pkg/config"
)

var (
	configPath = flag.String("config", "./config/config.yaml", "path to config file")
)

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app, cleanup, err := InitializeApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		log.Fatalf("app runtime error: %v", err)
	}

	if err := app.Close(); err != nil {
		log.Fatalf("app close error: %v", err)
	}
}
