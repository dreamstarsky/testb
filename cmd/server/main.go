package main

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"smog-detector/internal/config"
	"smog-detector/internal/handler"
	"smog-detector/internal/ipgeo"
	"smog-detector/internal/qweather"
	"smog-detector/internal/repository"
	"smog-detector/internal/service"
)

func main() {
	// 读取配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 配置数据库
	repo, err := repository.NewSQLiteRepository(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}
	defer repo.Close()

	// 天气服务
	client := qweather.NewClient(cfg.QWeatherBaseURL, cfg.QWeatherAPIKey, cfg.QWeatherToken)
	ipgeoClient := ipgeo.NewClient()
	svc := service.NewWeatherService(repo, client, ipgeoClient, cfg.CacheDuration)

	// 前端
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "index.html"))
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	h := handler.New(svc, tmpl, currentBuildVersion())
	mux := http.NewServeMux()
	h.Register(mux)
	mux.Handle("/static/", noStore(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join("web", "static"))))))

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

func currentBuildVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	version := ""
	modified := false
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			version = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if version == "" {
		return "unknown"
	}
	if modified {
		version += "-dirty"
	}
	return version
}
