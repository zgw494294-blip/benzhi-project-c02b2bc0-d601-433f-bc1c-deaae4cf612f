package selfcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"blast-permit/internal/application"
	"blast-permit/internal/httpapi"
	"blast-permit/internal/store"
)

func Run(parent context.Context, addr string) error {
	ctx, cancel := context.WithTimeout(parent, 25*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "blast-permit-selfcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	repo, err := store.Open(filepath.Join(dir, "selfcheck.db"))
	if err != nil {
		return err
	}
	defer repo.Close()
	service := application.New(repo)
	api := httpapi.New(service)
	server := httpapi.NewServer(addr, api.Handler())
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("自检监听 %s 失败: %w", addr, err)
	}
	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			serveErr <- err
		}
		close(serveErr)
	}()
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = server.Shutdown(shutdownCtx)
	}()
	base := "http://" + listener.Addr().String()
	client := newClient(base)
	defer client.close()
	ready := time.NewTicker(40 * time.Millisecond)
	defer ready.Stop()
	for {
		select {
		case err := <-serveErr:
			if err != nil {
				return err
			}
			return fmt.Errorf("自检服务提前停止")
		case <-ctx.Done():
			return fmt.Errorf("等待自检服务就绪超时: %w", ctx.Err())
		case <-ready.C:
			var health map[string]string
			if err := client.request(ctx, http.MethodGet, "/healthz", "", "", nil, http.StatusOK, &health); err == nil {
				if err = runFlow(ctx, client); err != nil {
					return err
				}
				return nil
			}
		}
	}
}
