package httpapi

import (
	"net/http"

	"blast-permit/internal/application"
)

func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (a *API) CreateCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateCaseCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		requestError(w, err)
		return
	}
	if err := mergeKeyHeader(r, &cmd.IdempotencyKey); err != nil {
		requestError(w, err)
		return
	}
	out, err := a.service.CreateCase(r.Context(), actorFrom(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (a *API) AddTargets(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddTargetsCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		requestError(w, err)
		return
	}
	if err := mergeMutationHeaders(r, &cmd.ExpectedVersion, &cmd.IdempotencyKey); err != nil {
		requestError(w, err)
		return
	}
	out, err := a.service.AddTargets(r.Context(), r.PathValue("caseId"), actorFrom(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (a *API) CompleteBaseline(w http.ResponseWriter, r *http.Request) {
	var cmd application.CompleteBaselineCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		requestError(w, err)
		return
	}
	if err := mergeMutationHeaders(r, &cmd.ExpectedVersion, &cmd.IdempotencyKey); err != nil {
		requestError(w, err)
		return
	}
	out, err := a.service.CompleteBaseline(r.Context(), r.PathValue("caseId"), actorFrom(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
