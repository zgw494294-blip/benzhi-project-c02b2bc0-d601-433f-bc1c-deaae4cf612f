package selfcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type client struct {
	base string
	http *http.Client
}

func newClient(base string) *client {
	return &client{base: base, http: &http.Client{Timeout: 4 * time.Second}}
}
func (c *client) close() { c.http.CloseIdleConnections() }
func (c *client) request(ctx context.Context, method, path, role, name string, body any, want int, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if role != "" {
		req.Header.Set("X-Actor-Role", role)
		req.Header.Set("X-Actor-Name", name)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != want {
		return fmt.Errorf("%s %s: 期望 HTTP %d，实际 %d: %s", method, path, want, resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err = json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("解析 %s 响应: %w", path, err)
		}
	}
	return nil
}
