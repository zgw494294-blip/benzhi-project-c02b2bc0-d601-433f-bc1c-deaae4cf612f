package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"blast-permit/internal/domain"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "服务内部错误"
	var de *domain.Error
	if errors.As(err, &de) {
		code = string(de.Code)
		message = de.Message
		switch de.Code {
		case domain.CodeValidation, domain.CodeBaselineNotReady, domain.CodeRemediationReference, domain.CodeMonitoringInterlock, domain.CodeReviewReasons:
			status = http.StatusBadRequest
		case domain.CodeNotFound:
			status = http.StatusNotFound
		case domain.CodeConflict, domain.CodeState, domain.CodeFrozen, domain.CodeTargetConflict, domain.CodeEmptyRevision, domain.CodeReviewUnresolved, domain.CodePermitPrecheck:
			status = http.StatusConflict
		case domain.CodeForbidden:
			status = http.StatusForbidden
		case domain.CodeCorrupt:
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message, Details: de.Details}})
		return
	}
	writeJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}
func requestError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: errorBody{Code: "invalid_request", Message: err.Error()}})
}
