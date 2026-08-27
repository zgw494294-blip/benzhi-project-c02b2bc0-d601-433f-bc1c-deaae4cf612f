package httpapi

import (
	"net/http"
	"sync"
	"time"

	"blast-permit/internal/application"
	"blast-permit/internal/domain"
)

type API struct {
	service   *application.Service
	caseMu    sync.RWMutex
	caseCache map[string]*domain.CaseFile
}

func New(service *application.Service) *API {
	return &API{service: service, caseCache: map[string]*domain.CaseFile{}}
}
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second}
}
