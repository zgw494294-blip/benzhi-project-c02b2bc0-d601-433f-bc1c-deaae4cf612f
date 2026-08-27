package httpapi

import (
	"net/http"
	"time"

	"blast-permit/internal/application"
)

type API struct{ service *application.Service }

func New(service *application.Service) *API { return &API{service: service} }
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second}
}
