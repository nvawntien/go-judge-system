package main

import (
	"go-judge-system/pkg/config"
	"log"
)

func main() {
	cfg, err := config.LoadConfig("../../config")
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