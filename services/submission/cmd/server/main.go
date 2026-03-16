package main

import (
	"go-judge-system/pkg/config"
	"log"
)

var (
	version   = "dev"
	buildTime = "unknown"
	commitSHA = "unknown"
)

func main() {
	cfg, err := config.LoadConfig("/app/config")
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	app, err := InitializeApp(cfg)
	if err != nil {
		log.Fatalf("initialize app failed: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("server shutdown with error: %v", err)
	}
}
