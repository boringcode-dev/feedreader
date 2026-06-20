package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"feedreader/internal/config"
	"feedreader/internal/db"
	"feedreader/internal/repository"
	"feedreader/internal/service"
	webapp "feedreader/internal/web"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	command := "serve"
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		command = args[0]
		args = args[1:]
	}
	switch command {
	case "serve":
		return serve(cfg, args)
	case "fetch":
		return fetch(cfg)
	case "healthcheck":
		return healthcheck(cfg)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		return 2
	}
}

func serve(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	host := fs.String("host", cfg.Host, "host")
	port := fs.Int("port", cfg.Port, "port")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg.Host = *host
	cfg.Port = *port
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer database.Close()
	repo := repository.NewSQLiteRepository(database)
	if err := repo.NormalizeGitHubItems(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	svc := service.New(cfg, repo)
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	app, err := webapp.New(cfg, svc, baseDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	svc.StartScheduler(ctx)
	server := &http.Server{Addr: cfg.Addr(), Handler: app.Handler()}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("feedreader listening on http://%s", cfg.Addr())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func fetch(cfg config.Config) int {
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer database.Close()
	repo := repository.NewSQLiteRepository(database)
	if err := repo.NormalizeGitHubItems(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	svc := service.New(cfg, repo)
	outcomes := svc.RefreshAll(context.Background())
	payload, _ := json.MarshalIndent(outcomes, "", "  ")
	fmt.Println(string(payload))
	for _, outcome := range outcomes {
		if !outcome.OK {
			return 1
		}
	}
	return 0
}

func healthcheck(cfg config.Config) int {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", cfg.Port)
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "unexpected status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}

func mustBaseDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Clean(wd)
}
