package httpapi

import (
	"net/http"

	"blast-permit/internal/application"
)

func (a *API) submitRevision(w http.ResponseWriter, r *http.Request, remediation bool) {
	var cmd application.SubmitRevisionCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		requestError(w, err)
		return
	}
	if err := mergeMutationHeaders(r, &cmd.ExpectedVersion, &cmd.IdempotencyKey); err != nil {
		requestError(w, err)
		return
	}
	out, err := a.service.SubmitRevision(r.Context(), r.PathValue("caseId"), actorFrom(r), cmd, remediation)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (a *API) SubmitRevision(w http.ResponseWriter, r *http.Request)    { a.submitRevision(w, r, false) }
func (a *API) SubmitRemediation(w http.ResponseWriter, r *http.Request) { a.submitRevision(w, r, true) }
func (a *API) ReviewCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReviewCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		requestError(w, err)
		return
	}
	if err := mergeMutationHeaders(r, &cmd.ExpectedVersion, &cmd.IdempotencyKey); err != nil {
		requestError(w, err)
		return
	}
	out, err := a.service.Review(r.Context(), r.PathValue("caseId"), actorFrom(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (a *API) IssuePermit(w http.ResponseWriter, r *http.Request) {
	var cmd application.IssuePermitCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		requestError(w, err)
		return
	}
	if err := mergeMutationHeaders(r, &cmd.ExpectedVersion, &cmd.IdempotencyKey); err != nil {
		requestError(w, err)
		return
	}
	out, err := a.service.IssuePermit(r.Context(), r.PathValue("caseId"), actorFrom(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
