package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const maxBodyBytes int64 = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		return fmt.Errorf("Content-Type 必须为 application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("JSON 请求无效: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("请求体只能包含一个 JSON 对象")
	}
	return nil
}
func mergeMutationHeaders(r *http.Request, expected *int64, key *string) error {
	if h := r.Header.Get("Expected-Version"); h != "" {
		n, err := strconv.ParseInt(h, 10, 64)
		if err != nil {
			return fmt.Errorf("Expected-Version 请求头无效")
		}
		if *expected != 0 && *expected != n {
			return fmt.Errorf("请求头与正文 expectedVersion 不一致")
		}
		*expected = n
	}
	if h := r.Header.Get("Idempotency-Key"); h != "" {
		if *key != "" && *key != h {
			return fmt.Errorf("请求头与正文 idempotencyKey 不一致")
		}
		*key = h
	}
	return nil
}
func mergeKeyHeader(r *http.Request, key *string) error {
	if h := r.Header.Get("Idempotency-Key"); h != "" {
		if *key != "" && *key != h {
			return fmt.Errorf("请求头与正文 idempotencyKey 不一致")
		}
		*key = h
	}
	return nil
}
