package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/database"
	"go-judge-system/services/problem/internal/adapter/outbound/persistence/postgres"
	seedtags "go-judge-system/services/problem/internal/seed/tags"
)

func main() {
	configPath := flag.String("config", "./config", "directory containing config.yaml")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	fmt.Printf("Target database: %s:%d/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	db, err := database.ConnectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get database handle failed: %v", err)
	}
	defer sqlDB.Close()

	fmt.Println("Seeding Competitive Programming tags...")
	result, err := seedtags.Seed(context.Background(), postgres.NewTagRepository(db), seedtags.Desired(), os.Stdout)
	fmt.Printf("Created: %d\nAlready existed: %d\nFailed: %d\nTotal desired tags: %d\n", result.Created, result.AlreadyExisted, result.Failed, result.Total())
	if err != nil {
		log.Fatal(err)
	}
}
