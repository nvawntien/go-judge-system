package main

import (
	"go-judge-system/pkg/config"
	"go-judge-system/services/auth/internal/adapter/outbound/mail"
	"log"
)

// Build metadata is set by the Auth image build for release traceability.
// The server does not expose it yet, but keeping these symbols aligned with
// the provisioning command ensures both binaries carry the same identity.
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
