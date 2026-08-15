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
	"go-judge-system/services/problem/internal/application/backfill"
)

func main() {
	configPath := flag.String("config", "./config", "directory containing config.yaml")
	apply := flag.Bool("apply", false, "write recognized legacy sections; omit for dry-run")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	db, err := database.ConnectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get database handle failed: %v", err)
	}
	defer sqlDB.Close()

	if !*apply {
		fmt.Fprintln(os.Stdout, "Dry run: no rows will be written. Re-run with --apply after reviewing READY/SKIP output.")
	}
	result, err := backfill.Run(context.Background(), postgres.NewProblemRepository(db), *apply, os.Stdout)
	fmt.Fprintf(os.Stdout, "Summary: scanned=%d migrated=%d skipped=%d populated=%d no_headers=%d ambiguous=%d empty_part=%d concurrent=%d\n",
		result.Scanned, result.Migrated, result.Skipped(), result.SkippedPopulated, result.SkippedNoHeaders, result.SkippedAmbiguous, result.SkippedEmptyPart, result.ConcurrentSkipped)
	if err != nil {
		log.Fatal(err)
	}
}
