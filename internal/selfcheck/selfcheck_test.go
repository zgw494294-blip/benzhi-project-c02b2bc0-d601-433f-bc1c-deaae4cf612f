package selfcheck

import (
	"context"
	"testing"
	"time"
)

func TestCompleteHTTPFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := Run(ctx, "127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
}
