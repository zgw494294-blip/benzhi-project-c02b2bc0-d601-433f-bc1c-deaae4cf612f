package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blast-permit/internal/application"
	"blast-permit/internal/httpapi"
	"blast-permit/internal/selfcheck"
	"blast-permit/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "blast-permit:", err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := parseConfig()
	if err != nil {
		return err
	}
	if cfg.selfcheck {
		if err = selfcheck.Run(context.Background(), cfg.addr); err != nil {
			return fmt.Errorf("自检失败: %w", err)
		}
		fmt.Println("blast-permit 自检通过")
		return nil
	}
	repo, err := store.Open(cfg.database)
	if err != nil {
		return err
	}
	defer repo.Close()
	service := application.New(repo)
	api := httpapi.New(service)
	server := httpapi.NewServer(cfg.addr, api.Handler())
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()
	fmt.Printf("blast-permit 正在监听 %s\n", cfg.addr)
	select {
	case err = <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-signals.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err = server.Shutdown(ctx); err != nil {
			return fmt.Errorf("关闭 HTTP 服务: %w", err)
		}
		err = <-serveErr
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
