package main

import (
	"go-judge-system/pkg/config"
	"go-judge-system/services/auth/internal/adapter/outbound/mail"
	"log"
)

func main() {
	cfg, err := config.LoadConfig("/app/config")
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	if err := mail.ValidateConfig(cfg.SMTP, cfg.App, cfg.Server); err != nil {
		log.Fatalf("invalid email configuration: %v", err)
	}

	app, err := InitializeApp(cfg)
	if err != nil {
		log.Fatalf("initialize app failed: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("server shutdown with error: %v", err)
	}
}
