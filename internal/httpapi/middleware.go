package httpapi

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"blast-permit/internal/application"
)

func actorFrom(r *http.Request) application.Actor {
	return application.Actor{Role: r.Header.Get("X-Actor-Role"), Name: r.Header.Get("X-Actor-Name")}
}
func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				_ = fmt.Sprint(v)
				_ = debug.Stack()
				writeError(w, fmt.Errorf("panic"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
