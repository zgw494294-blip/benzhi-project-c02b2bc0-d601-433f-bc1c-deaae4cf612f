package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"blast-permit/internal/application"
	"blast-permit/internal/store"
)

func TestStrictJSONAndHealth(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	handler := New(application.New(repo)).Handler()
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("healthz=%d", health.Code)
	}
	body := bytes.NewBufferString(`{"siteName":"A","workZone":"B","idempotencyKey":"12345678","unknown":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor-Role", application.RoleDesigner)
	req.Header.Set("X-Actor-Name", "测试者")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("未知字段响应=%d: %s", response.Code, response.Body.String())
	}
}
